#!/system/bin/sh
# JunkClean 共享更新逻辑：下载 → 临时解压 → 校验 → 原子覆盖 → 重启 daemon
# 用法: sh update.sh [MODDIR]  默认 MODDIR=/data/adb/modules/junkclean
MODDIR=${1:-/data/adb/modules/junkclean}
mkdir -p /data/adb/junkclean

if command -v curl >/dev/null 2>&1; then
  get() { curl -s --max-time 8 "$1" 2>/dev/null; }
  dl() { curl -s -L --max-time 60 -o "$2" "$1" 2>/dev/null; }
else
  get() { busybox wget -q -O - -T 8 "$1" 2>/dev/null; }
  dl() { busybox wget -q -O "$2" -T 60 "$1" 2>/dev/null; }
fi

fetch_ver() { get https://raw.githubusercontent.com/ce11kjw/junkclean/main/update.json; }

NEW=$(fetch_ver | grep -o '"versionCode": *[0-9]*' | grep -o '[0-9]*')
CUR=$(grep '^versionCode=' "$MODDIR/module.prop" 2>/dev/null | cut -d= -f2)
[ -n "$NEW" ] || { echo "无法获取远程版本"; exit 1; }
[ "$NEW" -gt "${CUR:-0}" ] || { echo "已是最新 v${CUR:-?}"; exit 1; }

ZIP=$(fetch_ver | grep -o '"zipUrl": *"[^"]*"' | cut -d'"' -f4)
[ -n "$ZIP" ] || { echo "无法获取下载地址"; exit 1; }

TMP=/data/local/tmp/jc-update
TMPZIP=/data/local/tmp/jc-update.zip
rm -rf "$TMP" "$TMPZIP"
dl "$ZIP" "$TMPZIP" || { echo "下载失败"; exit 1; }
mkdir -p "$TMP"
unzip -o -q "$TMPZIP" -d "$TMP" 2>/dev/null || { rm -rf "$TMP" "$TMPZIP"; echo "解压失败"; exit 1; }
# 校验 zip 完整性（防损坏覆盖）
if [ ! -f "$TMP/module.prop" ] || [ ! -f "$TMP/system/bin/junkclean" ]; then
  rm -rf "$TMP" "$TMPZIP"; echo "zip 无效，已取消"; exit 1
fi
# 备份旧二进制（可回滚）
[ -f "$MODDIR/system/bin/junkclean" ] && cp "$MODDIR/system/bin/junkclean" /data/adb/junkclean/junkclean.bak 2>/dev/null
# 原子覆盖
cp -rf "$TMP/." "$MODDIR/" 2>/dev/null
chmod -R 777 "$MODDIR" 2>/dev/null
rm -rf "$TMP" "$TMPZIP"
# 重启 daemon（不重启手机）
pkill -f "junkclean[ ]daemon" 2>/dev/null; sleep 0.5
setsid "$MODDIR/system/bin/junkclean" daemon >/dev/null 2>&1 &
echo "已更新到 v$NEW，daemon 已重启"
exit 0
