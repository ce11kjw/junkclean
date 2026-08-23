#!/system/bin/sh
# JunkClean service.sh - late_start：启动 cleand 守护 + 目录重定向 + 磁盘维护初始化
MODDIR=${0%/*}
ADR=/data/adb/junk-cleaner
BIN=$MODDIR/bin

# 1) 目录重定向（失败仅记日志，不影响其他）
sh "$MODDIR/binder.sh" mount >> "$ADR/cleaner.log" 2>&1 &

# 2) 启动 cleand 守护（HTTP :46780 + 内置定时 + WebUI 静态服务）
if [ -x "$BIN/cleand" ]; then
  # 重复启动保护
  if ! pidof cleand >/dev/null 2>&1; then
    "$BIN/cleand" -d "$ADR" -m "$MODDIR" >> "$ADR/daemon.log" 2>&1 &
  fi
fi

# 3) 开机完成后再等存储就绪执行一次快速体检预热（非阻塞）
( sleep 90
  [ "$(cat "$ADR/config.conf" 2>/dev/null | grep -c '^cat_prewarm=1')" = "1" ] && "$BIN/junkclean" scan >/dev/null 2>&1
) &
exit 0
