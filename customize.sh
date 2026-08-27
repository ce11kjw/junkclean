#!/system/bin/sh
# JunkClean customize.sh — 安装流程：冲突检测 + 运行时目录 + 777 权限 + 结果反馈
MODPATH=${MODPATH:-$(dirname "$0")}
ADR=/data/adb/junkclean

ui_print "• JunkClean v4.0.0 安装流程开始"

# ========== 冲突检测：同类别 GC/Trim/清理模块 ==========
CONFLICT_IDS="
  wipe.cache.module clear_box
  f2fs.optimizer f2fs_optimizer f2fsopt
  android.auto.fstrim auto_fstrim fstrim fstrimmer frequently_fstrimmer
  clear.optimization CLEAR clear-rubbish Clear_Rubbish
  basiccleaner basic_cleaner cache-cleaner ccforksum cachecleaner ccleaner
  cleanerroyall cleaner-royall xiaojiu xiaojiu9 xiaojiu_cleaner
  cleaner-cache taamarin-cleaner smartclear smarter_cleaner
"
if [ -d /data/adb/modules ]; then
  found=""
  for cf in /data/adb/modules/*/module.prop; do
    [ -f "$cf" ] || continue
    mid=$(grep -m1 '^id=' "$cf" 2>/dev/null | cut -d= -f2 | tr 'A-Z' 'a-z')
    [ -n "$mid" ] || continue
    for bad in $CONFLICT_IDS; do
      [ "$mid" = "$bad" ] && found="$found $mid"
    done
  done
  if [ -n "$found" ]; then
    ui_print "!! 检测到同类别冲突模块:$found"
    ui_print "   (GC/清理类模块并存可能导致冲突/异常耗电)"
    abort "请先在模块管理器禁用/卸载这些模块，再重新安装 JunkClean"
  fi
  ui_print "✓ 未检测到冲突模块"
fi

# ========== 运行时数据目录（模块外，升级保留） ==========
mkdir -p "$ADR"
[ -f "$ADR/config.json" ] || cat > "$ADR/config.json" <<'CFG'
{
  "whitelist": [],
  "aiEndpoint": "",
  "aiKey": "",
  "aiModel": ""
}
CFG
chmod 600 "$ADR/config.json" 2>/dev/null

# ========== 全部文件 777 权限 + 反馈结果 ==========
chmod -R 777 "$MODPATH" 2>/dev/null
if [ -x "$MODPATH/system/bin/junkclean" ]; then
  ui_print "✓ 777 权限设置成功（全部文件可读写执行）"
else
  ui_print "✗ 权限设置失败！请手动执行: chmod -R 777 $MODPATH"
fi

ui_print "✓ 运行时目录就绪: $ADR"

# ========== 立即部署到最终目录（安装即生效，不重启手机） ==========
INSTALL=/data/adb/modules/junkclean
mkdir -p "$INSTALL"
cp -rf "$MODPATH/." "$INSTALL/" 2>/dev/null
chmod -R 777 "$INSTALL" 2>/dev/null

# ========== 自动升级检查（autoupdate=1 开启） ==========
if [ "$(grep '^autoupdate=' "$INSTALL/module.prop" 2>/dev/null | cut -d= -f2)" = "1" ]; then
  ui_print "• 检查最新版本..."
  NEW=$(curl -s --max-time 8 https://raw.githubusercontent.com/ce11kjw/junkclean/main/update.json 2>/dev/null | grep -o '"versionCode":[0-9]*' | grep -o '[0-9]*')
  CUR=$(grep '^versionCode=' "$INSTALL/module.prop" 2>/dev/null | cut -d= -f2)
  if [ -n "$NEW" ] && [ "$NEW" -gt "${CUR:-0}" ]; then
    ui_print "• 发现新版本 v$NEW，下载中..."
    ZIP=$(curl -s --max-time 8 https://raw.githubusercontent.com/ce11kjw/junkclean/main/update.json 2>/dev/null | grep -o '"zipUrl":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$ZIP" ]; then
      curl -s -L --max-time 60 -o /data/local/tmp/jc-latest.zip "$ZIP" 2>/dev/null
      unzip -o -q /data/local/tmp/jc-latest.zip -d "$INSTALL" 2>/dev/null         && ui_print "• 已升级到 v$NEW" || ui_print "• 升级失败，使用当前版本"
      rm -f /data/local/tmp/jc-latest.zip
    fi
  else
    ui_print "• 已是最新版本"
  fi
fi

# ========== 重启 daemon（不重启手机） ==========
pkill -f "junkclean[ ]daemon" 2>/dev/null
sleep 0.5
setsid "$INSTALL/system/bin/junkclean" daemon >/dev/null 2>&1 &
sleep 1
if pgrep -f "junkclean[ ]daemon" >/dev/null 2>&1; then
  ui_print "✅ 模块已生效（daemon 运行中），无需重启手机"
else
  ui_print "⚠ daemon 启动失败（重启后模块仍会正常生效）"
fi
ui_print "• JunkClean 安装完成"
