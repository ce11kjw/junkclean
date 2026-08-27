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
    if ! pgrep -f "junkclean daemon" >/dev/null 2>&1; then
      "$BIN" daemon >> "$LOG" 2>&1 &
      sleep 1
      if pgrep -f "junkclean daemon" >/dev/null 2>&1; then
        echo "✅ daemon 已启动 (127.0.0.1:46780)"; log "started"
      else
        echo "❌ 启动失败，查看 $LOG"; log "start FAILED"
      fi
    else
      echo "✅ daemon 已在运行"; log "already running"
    fi
    ;;
  stop)
    pkill -f "junkclean daemon" 2>/dev/null
    echo "⏹ daemon 已停止"; log "stopped"
    ;;
  update)
    echo "🔍 检查更新（热更新，不重启手机）..."
    NEW=$(curl -s --max-time 8 https://raw.githubusercontent.com/ce11kjw/junkclean/main/update.json 2>/dev/null | grep -o '"versionCode":[0-9]*' | grep -o '[0-9]*')
    CUR=$(grep '^versionCode=' "$MODDIR/module.prop" 2>/dev/null | cut -d= -f2)
    if [ -n "$NEW" ] && [ "$NEW" -gt "${CUR:-0}" ]; then
      echo "发现新版本 v$NEW，下载中..."
      ZIP=$(curl -s --max-time 8 https://raw.githubusercontent.com/ce11kjw/junkclean/main/update.json 2>/dev/null | grep -o '"zipUrl":"[^"]*"' | cut -d'"' -f4)
      curl -s -L --max-time 60 -o /data/local/tmp/jc-update.zip "$ZIP" 2>/dev/null
      if unzip -o -q /data/local/tmp/jc-update.zip -d "$MODDIR" 2>/dev/null; then
        chmod -R 777 "$MODDIR" 2>/dev/null
        pkill -f "junkclean[ ]daemon" 2>/dev/null
        sleep 0.5
        setsid "$MODDIR/system/bin/junkclean" daemon >/dev/null 2>&1 &
        echo "✅ 已升级到 v$NEW，daemon 已重启，无需重启手机"
      else
        echo "❌ 下载/解压失败"
      fi
      rm -f /data/local/tmp/jc-update.zip
    else
      echo "✅ 已是最新版本 v${CUR:-?}"
    fi
    ;;
  status|*)
    if pgrep -f "junkclean daemon" >/dev/null 2>&1; then
      echo "🟢 运行中 (127.0.0.1:46780)"
      echo "WebUI: 浏览器打开 http://127.0.0.1:46780 或使用模块 WebUI 入口"
    else
      echo "⚪ 未运行"
      echo "用法: sh action.sh start|stop|status"
    fi
    ;;
esac
