#!/bin/bash
# JunkClean 打包脚本 + 自动版本管理（改代码即升版本）
set -e
cd "$(dirname "$0")"
# 版本兜底检查：有未提交改动但版本号与 HEAD 相同 → 拒绝打包（用户硬规则）
head_code=$(git show HEAD:module.prop 2>/dev/null | grep '^versionCode=' | cut -d= -f2)
cur_code=$(grep '^versionCode=' module.prop | cut -d= -f2)
if [ "$head_code" = "$cur_code" ] && ! git diff --quiet; then
  echo "⚠️ 硬规则: 有未提交改动但版本号未变，请运行 ./bump.sh"
  exit 1
fi

# ---- 自动版本 bump：代码有新提交但版本未变 → patch+1 ----

VER=$(grep '^version=' module.prop | cut -d= -f2)
OUT="JunkClean-${VER}.zip"
rm -f "$OUT"
zip -r -9 "$OUT" \
  module.prop customize.sh service.sh action.sh uninstall.sh binder.sh cleaner.sh \
  webroot/ bin/ rules/ META-INF/ update.json CHANGELOG.md README.md .gitignore \
  -x 'webroot/_alpine.js' -x 'bin/curl.old' -x '*/.DS_Store' -x '*.bak'
echo "=== $OUT ==="
unzip -l "$OUT" | tail -2
