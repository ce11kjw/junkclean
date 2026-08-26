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
ui_print "• JunkClean 安装完成，重启生效"
