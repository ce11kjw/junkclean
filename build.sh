#!/bin/bash
# JunkClean 打包脚本（KSU/AP 免 META-INF 格式 + Magisk 兼容）
set -e
cd "$(dirname "$0")"
VER=$(grep '^version=' module.prop | cut -d= -f2)
OUT="JunkClean-${VER}.zip"

# 检查 bin/curl 是否就位（不入 git，服务器重建后需 build-curl.sh）
if [ ! -f bin/curl ] || ! file bin/curl 2>/dev/null | grep -q aarch64; then
  echo "⚠️ bin/curl 缺失或非 ARM64，先执行 ./build-curl.sh"
  exit 1
fi
if [ ! -f bin/cleand ]; then
  echo "⚠️ bin/cleand 缺失，先交叉编译: aarch64-linux-gnu-gcc -O2 -static -o bin/cleand cleand.c -pthread"
  exit 1
fi

rm -f "$OUT"
zip -r -9 "$OUT" \
  module.prop customize.sh service.sh action.sh uninstall.sh binder.sh cleaner.sh \
  webroot/ bin/ rules/ META-INF/ update.json CHANGELOG.md README.md .gitignore \
  -x 'webroot/_alpine.js' -x 'bin/curl.old' -x '*/.DS_Store' -x '*.bak'
echo "=== $OUT ==="
unzip -l "$OUT" | tail -3
