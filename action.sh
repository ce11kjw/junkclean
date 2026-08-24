#!/system/bin/sh
# JunkClean action.sh - 手动启动/停止（APatch 按钮调用）
MODDIR=${0%/*}
ADR=/data/adb/junk-cleaner
BIN=$MODDIR/bin
LOG="$ADR/daemon.log"

ts() { date '+%Y-%m-%d %H:%M:%S'; }
log() {
  echo "$(ts) [INFO] $*" >> "$LOG" 2>/dev/null \
    || echo "$(ts) [INFO] $*" >> /data/local/tmp/jc_daemon.log
}

log "action: $1"
case "$1" in
  start)
    if [ ! -x "$BIN/cleand" ]; then
      log "ERR action: bin/cleand 无执行权限，先 chmod 755 $BIN/cleand"
      exit 1
    fi
    pidof cleand >/dev/null 2>&1 || "$BIN/cleand" -d "$ADR" -m "$MODDIR" >> "$LOG" 2>&1 &
    sleep 1
    pidof cleand >/dev/null 2>&1 && log "action: cleand running" || log "ERR action: 启动失败"
    ;;
  stop)
    pkill -f "$BIN/cleand" 2>/dev/null
    log "action: cleand stopped"
    ;;
  *)
    log "action: 用法 $0 start|stop"
    ;;
esac
