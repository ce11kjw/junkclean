#!/system/bin/sh
# JunkClean service.sh - late_start：启动 cleand + 目录重定向 + 预热
# 关键：启动日志由 shell 写，cleand 起不来也能记录失败原因
MODDIR=${0%/*}
ADR=/data/adb/junk-cleaner
BIN=$MODDIR/bin
LOG="$ADR/daemon.log"

ts() { date '+%Y-%m-%d %H:%M:%S'; }
log() {
  echo "$(ts) [INFO] $*" >> "$LOG" 2>/dev/null \
    || echo "$(ts) [INFO] $*" >> /data/local/tmp/jc_daemon.log
}

log "service: starting"

# 1) 目录重定向（失败仅记日志，不影响其他）
sh "$MODDIR/binder.sh" mount >> "$ADR/cleaner.log" 2>&1 &

# 2) 启动 cleand 守护（HTTP :46780 + 定时 + WebUI）
if [ ! -x "$BIN/cleand" ]; then
  log "ERR service: bin/cleand 不存在或无执行权限，启动失败"
  exit 1
fi
if pidof cleand >/dev/null 2>&1; then
  log "service: cleand 已在运行 (pid $(pidof cleand))"
else
  "$BIN/cleand" -d "$ADR" -m "$MODDIR" >> "$LOG" 2>&1 &
  sleep 1
  if pidof cleand >/dev/null 2>&1; then
    log "service: cleand running (pid $(pidof cleand))"
  else
    log "ERR service: cleand 启动失败，详见上方 stderr"
  fi
fi

# 3) 开机 90s 后存储就绪，执行一次快速体检预热（非阻塞）
( sleep 90
  [ "$(cat "$ADR/config.conf" 2>/dev/null | grep -c '^cat_prewarm=1')" = "1" ] && "$BIN/junkclean" scan >/dev/null 2>&1
) &
exit 0
