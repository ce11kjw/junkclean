#!/system/bin/sh
# JunkClean action.sh — 管理器 Action 按钮：启动/停止/状态（KSU/AP/Magisk 通用）
MODDIR=${0%/*}
ADR=/data/adb/junkclean
BIN="$MODDIR/system/bin/junkclean"
LOG="$ADR/daemon.log"
[ -d "$ADR" ] || mkdir -p "$ADR"

ts() { date '+%Y-%m-%d %H:%M:%S'; }
log() { echo "$(ts) [action] $*" >> "$LOG" 2>/dev/null; }

case "$1" in
  start)
    if ! pgrep -f "junkclean[ ]daemon" >/dev/null 2>&1; then
      setsid "$BIN" daemon >> "$LOG" 2>&1 &
      sleep 1
      if pgrep -f "junkclean[ ]daemon" >/dev/null 2>&1; then
        echo "✅ daemon 已启动 (127.0.0.1:46780)"; log "started"
      else
        echo "❌ 启动失败，查看 $LOG"; log "start FAILED"
      fi
    else
      echo "✅ daemon 已在运行"; log "already running"
    fi
    ;;
  stop)
    pkill -f "junkclean[ ]daemon" 2>/dev/null
    echo "⏹ daemon 已停止"; log "stopped"
    ;;
  update)
    echo "🔍 检查更新（热更新，不重启手机）..."
    sh "$MODDIR/scripts/update.sh" "$MODDIR"
    ;;
  status|*)
    if pgrep -f "junkclean[ ]daemon" >/dev/null 2>&1; then
      echo "🟢 运行中 (127.0.0.1:46780)"
      echo "WebUI: 浏览器打开 http://127.0.0.1:46780 或使用模块 WebUI 入口"
    else
      echo "⚪ 未运行"
      echo "用法: sh action.sh start|stop|status"
    fi
    ;;
esac
