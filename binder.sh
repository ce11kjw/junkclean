#!/system/bin/sh
# JunkClean binder.sh - 目录重定向（bind mount）
# 用法: binder.sh {mount|unmount}
# 规则: /data/adb/junk-cleaner/rules/bind.list  每行: 源相对路径 目标路径（相对 /sdcard）
MODDIR=${0%/*}
ADR=${JC_ADR:-/data/adb/junk-cleaner}
LOG="$ADR/cleaner.log"
RL="$ADR/rules/bind.list"
[ -f "$RL" ] || { [ -f "$MODDIR/rules/bind.list" ] && RL="$MODDIR/rules/bind.list" || exit 0; }

# Android 12+ / 非 sdcardfs 检测
sdk=$(getprop ro.build.version.sdk 2>/dev/null)
has_sdcardfs=$(grep -c 'sdcardfs' /proc/filesystems 2>/dev/null)
[ -n "$sdk" ] && [ "$sdk" -ge 31 ] || { echo "[binder] 需要 Android 12+ (sdk=$sdk)，已跳过挂载" >>"$LOG"; exit 0; }
[ "$has_sdcardfs" = "1" ] && { echo "[binder] sdcardfs 设备不支持目录重定向，跳过" >>"$LOG"; exit 0; }

# sdcard 根：支持多用户下软链，取真实路径
SD=$(readlink -f /sdcard 2>/dev/null || echo /sdcard)

case "$1" in
  unmount)
    while read -r src dst; do
      case "$src" in \#*|"") continue;; esac
      case "$dst" in /*) ;; *) dst="/$dst";; esac
      tgt="$SD$dst"
      umount "$tgt" 2>/dev/null && echo "[binder] unmounted $tgt" >>"$LOG"
    done < "$RL"
    ;;
  *)
    n=0
    while read -r src dst; do
      case "$src" in \#*|"") continue;; esac
      src="$SD/$src"; case "$dst" in /*) ;; *) dst="/$dst";; esac
      tgt="$SD$dst"
      [ -d "$src" ] || { echo "[binder] 源缺失: $src" >>"$LOG"; continue; }
      mkdir -p "$tgt" 2>/dev/null
      if mount --bind "$src" "$tgt" 2>/dev/null; then
        n=$((n+1)); echo "[binder] OK $src -> $tgt" >>"$LOG"
      else
        echo "[binder] FAIL $src -> $tgt (仅记录，不影响其他功能)" >>"$LOG"
      fi
    done < "$RL"
    echo "[binder] done $n mounts" >>"$LOG"
    ;;
esac
exit 0
