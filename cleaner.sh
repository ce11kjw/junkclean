#!/system/bin/sh
# JunkClean cleaner.sh - 核心引擎：清理/体检/分类/归档/维护/状态
# 协议: PROG <pct> <msg> → cleand 捕获转 WebUI 进度条; 标准输出即结果
#############################################
_self=$(readlink -f "$0" 2>/dev/null || echo "$0")
ADR=${JC_ADR:-/data/adb/junk-cleaner}
CFG="$ADR/config.conf"
LOG="$ADR/cleaner.log"
SCAN="$ADR/scan.json"
RULES="$ADR/rules"
[ -d "$RULES" ] || RULES="$(dirname "$_self")/rules"
export PATH=/sbin:/system/bin:/system/xbin:$PATH

log() { # log <level> <msg>
  # ponytail: 数字参数 guard 是防御性（实际 level 恒为字符串）。天花板：静默吞错。
  # 升级路径：若引入数字级别，改为显式级别过滤而非静默跳过。
  [ -z "${1##[0-9]*}" ] && return 0 # guard
  echo "$(date '+%F %T') [$1] $2" >> "$LOG"
  [ "$(wc -c < "$LOG" 2>/dev/null || echo 0)" -gt 262144 ] && {
    tail -c 131072 "$LOG" > "$LOG.tmp" && mv "$LOG.tmp" "$LOG"
  }
}
prog() { echo "PROG $1 $2"; } # prog <pct> <msg>
human() { # human <kb> -> "1.2G"
  b=$1
  if   [ "$b" -ge 1048576 ]; then awk -v x=$b 'BEGIN{printf "%.1fG", x/1048576}'
  elif [ "$b" -ge 1024 ];   then awk -v x=$b 'BEGIN{printf "%.1fM", x/1024}'
  else echo "${b}K"; fi
}
get_cfg() { # get_cfg <key> <default>
  v=$(sed -n "s/^$1=//p" "$CFG" 2>/dev/null | tail -1)
  [ -n "$v" ] && echo "$v" || echo "$2"
}
bj() { # 白名单拦截: bj <path>; 0=放行 1=拦截（/sdcard 大小写不敏感，统一转小写比较）
  p=$1; [ -n "$p" ] || return 1
  p=$(printf '%s' "$p" | tr 'A-Z' 'a-z')
  while IFS= read -r w; do
    case "$w" in \#*|"") continue;; esac
    w=$(printf '%s' "$w" | tr 'A-Z' 'a-z')
    case "$p" in "$w"|"$w"/*) return 1;; esac
  done < "$RULES/whitelist.list"
  return 0
}
# ponytail: 规则文件用空格分隔路径，含空格路径（如 "我的 目录"）会被拆错。
# 天花板：用户自定义含空格规则时统计/删除不精确。
# 升级路径：load_rules 改换行分隔 + 所有 for p in $RUL 调用点改 while read（改动大，内置规则无空格暂缓）。
load_rules() { # load_rules <file> -> RULES_LINES (norm+red separated)
  RUL='' REDR=''
  [ -f "$1" ] || return 0
  while IFS= read -r l; do
    case "$l" in '#RED '*) REDR="$REDR ${l#\#RED }";; \#*|"") continue;; *) RUL="$RUL $l";; esac
  done < "$1"
}
do_clean() { # do_clean <cats_csv> [force]
  FORCE=0; [ "$2" = "force" ] && FORCE=1
  [ "$1" = "all" ] && cats="cache junk apk zip empty social sqlite" || cats="$1"
  n_del=0; n_skip=0; freed=0; total_jobs=0; job=0
  for c in $cats; do total_jobs=$((total_jobs+1)); done
  for c in $cats; do
    job=$((job+1))
    case "$c" in
      cache)   [ "$(get_cfg cat_cache 1)" = "1" ] || { prog $((job*100/total_jobs)) "跳过 缓存(已关)"; continue; }; f="$RULES/cache.list"; nm="应用缓存";;
      junk)    [ "$(get_cfg cat_junk 1)" = "1" ] || continue; f="$RULES/junk.list"; nm="系统垃圾";;
      apk)     [ "$(get_cfg cat_apk 1)" = "1" ] || continue; f="$RULES/apk.list"; nm="安装包";;
      zip)     [ "$(get_cfg cat_zip 1)" = "1" ] || continue; f="$RULES/zip.list"; nm="压缩包";;
      social)  [ "$(get_cfg cat_social 1)" = "1" ] || continue; f="$RULES/social.list"; nm="社交专项";;
      empty)   [ "$(get_cfg cat_empty 1)" = "1" ] || continue; f=""; nm="空文件夹";;
      sqlite)  [ "$(get_cfg cat_sqlite 1)" = "1" ] || continue; f=""; nm="SQLite优化";;
      temp)    [ "$(get_cfg cat_temp 1)" = "1" ] || continue; f=""; nm="临时文件";;
      uninst)  [ "$(get_cfg cat_uninst 0)" = "1" ] || continue; f=""; nm="卸载残留";;
      *) continue;;
    esac
    prog $((job*100/total_jobs)) "$nm 清理中…"
    if [ "$c" = "empty" ]; then
      # 空文件夹（仅 /sdcard 且白名单保护）
      for d in /sdcard/* /sdcard/Android/data/*/cache; do
        [ -d "$d" ] || continue
        if [ "$FORCE" != "1" ]; then
          bj "$d" || { n_skip=$((n_skip+1)); continue; }
        fi
        find "$d" -type d -empty -delete 2>/dev/null
      done
      log INFO "empty-clean done"
      continue
    fi
    if [ "$c" = "sqlite" ]; then
      sqlite_opt; continue
    fi
    if [ "$c" = "temp" ]; then
      # 递归临时文件（白名单保护；日志类保守处理）
      find /sdcard -type f \( -name "*.tmp" -o -name "*.bak" -o -name "*.old" -o -name "*.temp" -o -name "*.log" \) ! -path "/sdcard/Android/obb/*" 2>/dev/null | while IFS= read -r tf; do
        [ -e "$tf" ] || continue
        if [ "$FORCE" != "1" ]; then
          bj "$tf" || { echo x >> "$ADR/.temp.skip"; continue; }
        fi
        sz=$(du -sk "$tf" 2>/dev/null | awk '{print $1}')
        rm -f "$tf" 2>/dev/null && { n_del=$((n_del+1)); freed=$((freed+sz)); log INFO "del-temp $tf"; }
      done
      rm -f "$ADR/.temp.skip"
      prog $((job*100/total_jobs)) "临时文件清理完成"
      continue
    fi
    if [ "$c" = "uninst" ]; then
      # 卸载应用残留：/data/data 下已卸载包目录（与 pm list 对比）
      pm list packages 2>/dev/null | sed 's/^package://' > "$ADR/.pkgs"
      for dd in /data/data/*/; do
        [ -d "$dd" ] || continue
        pkg=$(basename "$dd")
        grep -qx "$pkg" "$ADR/.pkgs" 2>/dev/null && continue
        # 确认不是系统关键目录
        case "$pkg" in com.android.*|com.google.android.*) continue;; esac
        sz=$(du -sk "$dd" 2>/dev/null | awk '{print $1}')
        rm -rf "$dd" 2>/dev/null && { n_del=$((n_del+1)); freed=$((freed+sz)); log INFO "del-uninst $dd"; }
      done
      rm -f "$ADR/.pkgs"
      prog $((job*100/total_jobs)) "卸载残留清理完成"
      continue
    fi
    load_rules "$f"
    # APK 保留期：仅删 N 天前的安装包（防误删刚下载）
    if [ "$c" = "apk" ]; then
      keep=$(get_cfg apk_keep_days 7)
      [ "$keep" -gt 0 ] 2>/dev/null || keep=7
      find /sdcard/Download /sdcard/tmp -type f \( -name "*.apk" -o -name "*.xapk" \) -mtime +"$keep" 2>/dev/null | while IFS= read -r ap; do
        [ -e "$ap" ] || continue
        sz=$(du -sk "$ap" 2>/dev/null | awk '{print $1}')
        rm -f "$ap" 2>/dev/null && { n_del=$((n_del+1)); freed=$((freed+sz)); log INFO "del-old-apk $ap"; }
      done
      # 非 apk 的压缩包仍走规则（不过期）
    fi
    # 红线项默认不删（social 的聊天媒体）; cat_junk 等开关为1时普通规则全清
    for p in $RUL; do
      case "$p" in /*);; *) continue;; esac
      for rp in $p; do
        [ -e "$rp" ] || continue
        # force 模式跳过白名单，但绝不允许删除根/关键系统目录
        case "$rp" in
          /|//|/system/*|/data/*|/cache/*|/vendor/*|/product/*|/apex/*) n_skip=$((n_skip+1)); continue;;
        esac
        if [ "$FORCE" != "1" ]; then
          bj "$rp" || { n_skip=$((n_skip+1)); continue; }
        fi
        sz=$(du -sk "$rp" 2>/dev/null | awk '{print $1}')
        rm -rf "$rp" 2>/dev/null && { n_del=$((n_del+1)); freed=$((freed+sz)); log INFO "del $rp"; }
      done
    done
  done
  prog 100 "完成: 删除$n_del项 释放$(human $freed)"
  # 累计统计（SmartClear/SweepX 模式：管理器 description 可见）
  if [ -f "$ADR/stats.total" ]; then
    read -r tdel tfreed < "$ADR/stats.total" 2>/dev/null
  fi
  tdel=$((tdel + n_del)); tfreed=$((tfreed + freed))
  echo "$tdel $tfreed" > "$ADR/stats.total"
  # 更新 module.prop description（动态展示累计清理）
  MP="$JC_MOD/module.prop"
  if [ -f "$MP" ] && [ -n "$JC_MOD" ]; then
    desc=$(sed -n 's/^description=//p' "$MP" 2>/dev/null | sed 's/ · 累计.*//')
    sed -i "s|^description=.*|description=${desc} · 累计清理 ${tdel} 项/释放 $(human $tfreed)|" "$MP" 2>/dev/null
  fi
  echo "{\"deleted\":$n_del,\"skipped\":$n_skip,\"freed_kb\":$freed}"
}
# ponytail: shell 引擎的天花板是高频操作性能（每次操作 fork 子进程）。
# 升级路径：核心引擎 C 化（目录遍历/统计/删除/md5），shell 保留 fstrim/sqlite 编排。
# 用户 2026-08-24 明确：保持现状，暂缓 C 化。
sqlite_opt() {
  # WAL-VACUUM 安全序列；磁盘<1GB 或 无 sqlite3 则跳过
  sq=$(command -v sqlite3) || sq=/system/bin/sqlite3
  [ -x "$sq" ] || { log WARN "sqlite3 不可用，跳过"; return; }
  free_kb=$(df -k /data | awk 'NR==2{print $4}')
  [ "$free_kb" -lt 1048576 ] && { log WARN "磁盘<1GB 跳过 VACUUM"; return; }
  # 排除系统 DB（避免影响系统进程）
  excl="/system/|/data/system/|/data/misc/|/data/virtual/"
  n=0; tot=0
  for db in /data/data/*/databases/*.db; do
    [ -f "$db" ] || continue
    tot=$((tot+1))
    bj "$db" || continue
    echo "$db" | grep -qE "$excl" && continue
    "$sq" "$db" "PRAGMA wal_checkpoint(TRUNCATE);" >/dev/null 2>&1
    "$sq" "$db" "VACUUM;" >/dev/null 2>&1
    "$sq" "$db" "PRAGMA wal_checkpoint(TRUNCATE); PRAGMA optimize;" >/dev/null 2>&1
    n=$((n+1))
  done
  log INFO "sqlite-opt $n/$tot dbs"
}

do_scan() { # 体检：规则分类统计 + 大文件 Top20（只统计不删）
  prog 5 "开始体检…"
  free_kb=$(df -k /sdcard 2>/dev/null | awk 'NR==2{print $4}'); [ -z "$free_kb" ] && free_kb=0
  total_kb=0; sc=''
  i=0; cats="cache junk apk social temp"
  for c in $cats; do
    i=$((i+1))
    f=$RULES/$c.list; [ -f "$f" ] || f=""; load_rules "$f"
    n=0; sz=0
    # 性能+正确性：规则转目录级（去尾部 /*），find/du 对目录内部遍历
    # 避免 glob 展开数万路径超 ARG_MAX（此前大目录 count=0）
    dirs=$(echo "$RUL" | tr ' ' '\n' | sed 's|/\*$||' | grep '^/')
    sz=$(du -sk $dirs 2>/dev/null | awk 'END{print $1}'); [ -z "$sz" ] && sz=0
    n=$(find $dirs -type f 2>/dev/null | wc -l)
    total_kb=$((total_kb+sz))
    sc="$sc\"$c\":{\"count\":$n,\"kb\":\"$sz\"},"
    prog $((5+i*15)) "统计中 $c ($(human $sz))"
  done
  # 大文件 Top20
  big=''
  # busybox find +du 排序（避免进程替换）
  find /sdcard -type f -size +100M 2>/dev/null | while IFS= read -r pf; do
    bs=$(du -sk "$pf" 2>/dev/null | awk '{print $1}')
    echo "$bs|$pf"
  done | sort -rn | head -20 > "$SCAN.tmp"
  while IFS='|' read -r bs pf; do
    [ -n "$bs" ] || continue
    big="$big{\"p\":\"$pf\",\"kb\":\"$bs\"},"
  done < "$SCAN.tmp"; rm -f "$SCAN.tmp"
  big=${big%,}
  red=''
  # 红线（聊天媒体）仅提示不统计删除
  load_rules "$RULES/social.list"
  for p in $REDR; do red="$red\"$p\","; done
  red=${red%,}
  health=green
  [ "$free_kb" -lt 3145728 ] && health=yellow   # <3GB
  [ "$free_kb" -lt 1048576 ] && health=red      # <1GB
  sc=${sc%,}
  printf '{"ts":"%s","free_kb":"%s","health":"%s","cats":{%s},"big":[%s],"redlines":[%s]}\n' \
    "$(date '+%F %T')" "$free_kb" "$health" "$sc" "$big" "$red" > "$SCAN"
  prog 100 "体检完成，可释放约 $(human $total_kb)"
  cat "$SCAN"
}
do_classify() { # 文件分类（@src 源目录 / @dest 目标根，跳过下载中文件，重名加序号）
  f="$RULES/classify.list"; [ -f "$f" ] || { log WARN "无 classify.list"; exit 0; }
  src=/sdcard; destbase=/sdcard/下载
  moved=0
  while IFS= read -r l; do
    case "$l" in \#*|"") continue;; esac
    case "$l" in
      @src=*)  src="${l#@src=}";  continue;;
      @dest=*) destbase="${l#@dest=}"; continue;;
    esac
    dest=$(echo "$l" | awk '{print $NF}')
    exts=$(echo "$l" | sed 's/ *[^ ]*$//')
    case "$dest" in
      /sdcard/*|/data/*|/storage/*|/mnt/*) ;;          # 绝对路径保留
      /*) dest="$destbase$dest";;                       # /子目录 → 目标根下
      *)  dest="$destbase/$dest";; esac                  # 相对路径 → 目标根下
    [ -d "$dest" ] || mkdir -p "$dest" 2>/dev/null || continue
    for ext in $exts; do
      # 用临时文件计数（避免管道子 shell 变量丢失）
      find "$src" -maxdepth 4 -type f -name "*.$ext" ! -path "$dest/*" 2>/dev/null | while IFS= read -r sf; do
        base=$(basename "$sf")
        case "$base" in
          *.part|*.crdownload|*.partial|*.tmp|*.downloading|*.!q|*.aria2) continue;;
        esac
        tgt="$dest/$base"
        [ -e "$tgt" ] && tgt="$dest/${base%.*}_$(date +%s).${ext}"
        bj "$sf" || continue
        mv -f "$sf" "$tgt" 2>/dev/null && echo x >> "$ADR/.classify.cnt"
      done
    done
    prog 50 "分类完成 $dest"
  done < "$f"
  [ -f "$ADR/.classify.cnt" ] && { moved=$(wc -l < "$ADR/.classify.cnt"); rm -f "$ADR/.classify.cnt"; }
  log INFO "classify moved=$moved"
  echo "{\"moved\":$moved}"
}
do_duplicate() { # 重复文件归档（>10M 哈希分组 → Duplicates，只移不删）
  dest=/sdcard/Duplicates; mkdir -p "$dest" 2>/dev/null
  [ -d "$dest" ] || { log WARN "无法创建 Duplicates"; exit 1; }
  tmp="$ADR/.dup.tmp"; : > "$tmp"
  prog 10 "比对重复文件中…"
  find /sdcard -type f -size +10M ! -path "$dest/*" -print0 2>/dev/null | \
    xargs -0 -P5 -I{} md5sum {} 2>/dev/null | \
    while read -r h sf; do [ -n "$h" ] || continue
      echo "$h|$sf" >> "$tmp"
    done
  prog 60 "发现重复项，归档中…"
  # 分组合并（awk 保持顺序）
  awk -F'|' '{n[$1]++; if(n[$1]==2) print; else if(n[$1]>2) print}' "$tmp" > "$tmp.dup"
  # 上述输出每行 h|sf；逐条移动（保留第一个副本）
  awk -F'|' '!seen[$1]++ {keep[$1]=$2; next} {print $2}' "$tmp" > "$tmp.mv"
  mv_n=0
  while IFS= read -r sf; do
    [ -n "$sf" ] || continue
    bj "$sf" || continue
    mv -f "$sf" "$dest/" 2>/dev/null && { mv_n=$((mv_n+1)); log INFO "dup->  $sf"; }
  done < "$tmp.mv"
  rm -f "$tmp" "$tmp.dup" "$tmp.mv"
  log INFO "duplicate moved=$mv_n"
  echo "{\"moved\":$mv_n}"
}
do_fstrim() { # 磁盘维护：drop_caches 清 RAM + EXT4 fstrim / F2FS 智能 GC
  sync
  # 释放页缓存/dentry/inode（drop_caches=3），写入不落盘不丢失数据
  echo 3 > /proc/sys/vm/drop_caches 2>/dev/null
  log INFO "drop_caches done"
  scr=0
  # 亮屏检测（无 backlight 节点视为亮屏可执行）
  if ls /sys/class/backlight/*/brightness >/dev/null 2>&1; then
    for bl in /sys/class/backlight/*/brightness; do
      [ "$(cat $bl 2>/dev/null)" -gt 0 ] && scr=1
    done
  fi
  [ "$(get_cfg fstrim_screenoff 1)" = "1" ] && [ "$scr" = "1" ] && { prog 100 "屏幕点亮，跳过磁盘维护"; echo '{"skipped":"screen-on"}'; exit 0; }
  chg=0
  [ -f /sys/class/power_supply/battery/status ] && case "$(cat /sys/class/power_supply/battery/status)" in
    Charging|Full) chg=1;; esac
  [ "$(get_cfg fstrim_charge_only 0)" = "1" ] && [ "$chg" = "0" ] && { prog 100 "未充电，跳过"; exit 0; }
  j=0; tot=8
  for mp in $(mount | awk '$3 ~ /^\/(system|data|cache|persist|vendor|product|system_ext)$/ {print $3}'); do
    j=$((j+1)); prog $((j*100/tot)) "维护中 $mp"
    fs=$(mount | awk -v m="$mp" '$3==m {print $5}')
    case "$fs" in
      f2fs)
        dev=$(mount | awk -v m="$mp" '$3==m {print $1}')
        dirty=$(cat /sys/fs/f2fs/*/dirty_segments 2>/dev/null | paste -sd+ | bc 2>/dev/null)
        [ -z "$dirty" ] && dirty=0
        if [ "$dirty" -gt "${F2FS_DIRTY_MIN:-200}" ] 2>/dev/null; then
          for gc in /sys/fs/f2fs/*/gc_urgent; do echo 1 > "$gc" 2>/dev/null; done
          sleep 2
          for gc in /sys/fs/f2fs/*/gc_urgent; do echo 0 > "$gc" 2>/dev/null; done
          log INFO "f2fs-gc $mp dirty=$dirty"
        else
          log INFO "f2fs healthy $mp dirty=$dirty (skip)"
        fi;;
      ext4|ext2)
        fstrim -v "$mp" 2>/dev/null >> "$ADR/fstrim.log"
        log INFO "fstrim $mp";;
    esac
  done
  prog 100 "磁盘维护完成"
  echo '{"done":1}'
}
do_rescan() { # 媒体库刷新（清后死条目清理）
  cmd=$(command -v cmd) 
  if [ -n "$cmd" ]; then
    "$cmd" content call --uri content://media/ --method scan_volume --arg external_primary >/dev/null 2>&1
  fi
  log INFO "media rescan"
  echo '{"rescan":1}'
}
do_ai() { # AI 分析：拼聚合数据 → 自填端点（cleaner 只输出，cleand 编排请求）
  [ -f "$SCAN" ] || do_scan > /dev/null
  cat "$SCAN"
}
do_verify() { # 规则完整性：输出每条规则文件的条数+SHA256
  [ -d "$RULES" ] || RULES="$(dirname "$_self")/rules"
  for f in "$RULES"/*.list; do
    [ -f "$f" ] || continue
    n=$(grep -c '^[^#]' "$f" 2>/dev/null)
    h=$(sha256sum "$f" 2>/dev/null | cut -d' ' -f1)
    echo "$(basename "$f") | 条数=$n | sha256=$h"
  done
}
do_status() { # cleand/WebUI 状态
  fk=$(df -k /sdcard 2>/dev/null | awk 'NR==2{print $4}'); [ -z "$fk" ] && fk=0
  h=green; [ "$fk" -lt 3145728 ] && h=yellow; [ "$fk" -lt 1048576 ] && h=red
  aut=$(sed -n 's/^author=//p' "$(dirname "$_self")/module.prop" 2>/dev/null)
  ver=$(sed -n 's/^version=//p' "$(dirname "$_self")/module.prop" 2>/dev/null)
  echo "{\"free_kb\":\"$fk\",\"health\":\"$h\",\"author\":\"$aut\",\"version\":\"$ver\",\"daemon\":1}"
}
[ "${JC_LIB:-0}" = "1" ] && return 0
# 入口参数拆分：cleand 传整个命令行串（"clean cache,apk force" → $1=clean $2=cache,apk $3=force）
set -- $1
case "$1" in
  clean)     do_clean "${2:-all}" "$3" ;;
  scan)      do_scan ;;
  classify)  do_classify ;;
  duplicate) do_duplicate ;;
  fstrim)    do_fstrim ;;
  rescan)    do_rescan ;;
  ai)        do_ai ;;
  status)    do_status ;;
  verify)    do_verify ;;
  *) echo "usage: $0 {clean|scan|classify|duplicate|fstrim|rescan|ai|status}"; exit 1;;
esac
