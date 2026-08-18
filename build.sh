#!/bin/bash
# JunkClean 打包脚本（KSU/AP 免 META-INF 格式）
set -e
cd "$(dirname "$0")"
VER=$(grep '^version=' module.prop | cut -d= -f2)
OUT="JunkClean-${VER}.zip"
rm -f "$OUT"
# 等 curl 编译产物就位（手动运行前确保 bin/curl 是 arm64）
zip -r -9 "$OUT" \
  module.prop customize.sh service.sh action.sh uninstall.sh binder.sh cleaner.sh \
  webroot/ bin/ rules/ \
  -x 'webroot/_alpine.js' -x 'bin/curl.old' -x '*/.DS_Store'
echo "=== $OUT ==="
unzip -l "$OUT" | tail -3
