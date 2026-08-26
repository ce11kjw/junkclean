#!/system/bin/sh
# JunkClean 卸载清理
pkill -f "junkclean daemon" 2>/dev/null
# 默认保留 /data/adb/junkclean（含白名单/配置）；如需彻底清除删此行
# rm -rf /data/adb/junkclean
echo "JunkClean 已卸载"
