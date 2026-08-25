#!/system/bin/sh
# JunkClean customize.sh - 安装阶段：冲突检测 + 运行时目录 + 权限
MODPATH=${MODPATH:-$(dirname "$0")}
ADR=/data/adb/junk-cleaner

ui_print "• JunkClean 安装流程开始"

# ========== 冲突检测：同类别 GC/Trim/清理模块 ==========
# 发现即 abort 停止安装（KSU/AP/Magisk 通用 abort()）
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
  for cf in /data/adb/modules/*/module.prop; do
    [ -f "$cf" ] || continue
    mid=$(grep -m1 '^id=' "$cf" 2>/dev/null | cut -d= -f2 | tr 'A-Z' 'a-z')
    [ -n "$mid" ] || continue
    for bad in $CONFLICT_IDS; do
      if [ "$mid" = "$bad" ]; then
        abort "!! 检测到同类别冲突模块: $mid
(GC/Trim/清理类模块并存可能导致冲突、异常耗电或数据不一致)
请先在模块管理器中禁用/卸载 [$mid]，再重新安装 JunkClean。"
      fi
    done
  done
  ui_print "✓ 未检测到冲突模块"
fi

# ========== 运行时数据目录（模块外，升级保留） ==========
mkdir -p "$ADR/rules"
[ -f "$ADR/config.conf" ] || cat > "$ADR/config.conf" <<'CFG'
# JunkClean 主配置 (K=V)
# cat 开关 (1=启用 0=禁用)
cat_cache=1
cat_junk=1
cat_apk=1
cat_zip=1
cat_empty=1
cat_social=1
cat_sqlite=1
# 磁盘维护
cat_fstrim=1
cat_duplicate=1
cat_rescan=1
# 定时 (cleand 热重载)
task_interval_h=12
# AI (OpenAI 兼容)
ai_base=
ai_key=
ai_model=
CFG
chmod 600 "$ADR/config.conf"
[ -f "$ADR/rules/whitelist.list" ] || cp "$MODPATH/rules/whitelist.list" "$ADR/rules/whitelist.list" 2>/dev/null || true
# 复制默认规则（仅当运行时缺失，不覆盖用户改动）
for rf in cache junk apk social whitelist classify bind; do
  [ -f "$ADR/rules/$rf.list" ] || cp "$MODPATH/rules/$rf.list" "$ADR/rules/$rf.list" 2>/dev/null || true
done
# 安装时给全部文件 777 权限（用户明确要求，防启动失败）
chmod -R 777 "$MODPATH" 2>/dev/null
# 反馈权限设置结果
if [ -x "$MODPATH/bin/cleand" ] && [ -x "$MODPATH/bin/curl" ]; then
  ui_print "✓ 全部文件 777 权限设置成功"
else
  ui_print "✗ 权限设置失败！请手动执行: chmod -R 777 $MODPATH"
fi
ui_print "✓ 运行时目录就绪: $ADR"
ui_print "• JunkClean 安装完成，重启生效"
