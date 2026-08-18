#!/system/bin/sh
# JunkClean uninstall.sh - 停止守护 + 保留运行时数据（规则/配置不丢）
pkill -f 'bin/cleand' 2>/dev/null
ui_print "• JunkClean 已卸载（配置保留于 /data/adb/junk-cleaner，可手动删除）"
