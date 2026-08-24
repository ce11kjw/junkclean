#!/bin/bash
# JunkClean 打包脚本 + 自动版本管理（改代码即升版本）
set -e
# 版本一致性检查：BUMP_VERSION_CHECK
# 用户硬规则：动一次代码就更新版本号
last_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
if [ -n "$last_tag" ]; then
  last_code=$(git show "$last_tag:module.prop" 2>/dev/null | grep "^versionCode=" | cut -d= -f2)
  cur_code=$(grep "^versionCode=" module.prop | cut -d= -f2)
  if [ "$last_code" = "$cur_code" ]; then
    nnew=$(git rev-list --count "$last_tag..HEAD" 2>/dev/null || echo 0)
    if [ "$nnew" -gt 0 ]; then
      echo "⚠️ 硬规则: 有 $nnew 次新 commit 但版本号未变（上次 $last_tag）"
      echo "   请运行 ./bump.sh 升版本号后再打包"
      exit 1
    fi
  fi
fi
cd "$(dirname "$0")"

# ---- 自动版本 bump：代码有新提交但版本未变 → patch+1 ----
auto_bump() {
  LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
  CUR_VER=$(grep '^version=' module.prop | cut -d= -f2)
  CUR_CODE=$(grep '^versionCode=' module.prop | cut -d= -f2)
  COMMITS_SINCE=$(git rev-list --count "${LAST_TAG}..HEAD" 2>/dev/null || echo 0)

  if [ "$COMMITS_SINCE" -gt 0 ] && [ "$CUR_VER" = "$LAST_TAG" ]; then
    NEW_CODE=$((CUR_CODE + 1))
    # vX.Y.Z → Z+1
    NEW_VER="v${CUR_VER#v}"
    MAJOR="${NEW_VER%%.*}"; REST="${NEW_VER#*.}"
    MINOR="${REST%%.*}"; PATCH="${REST##*.}"
    NEW_VER="v${MAJOR}.${MINOR}.$((PATCH + 1))"

    # 同步 4 处版本号
    sed -i "s/^version=.*/version=${NEW_VER}/; s/^versionCode=.*/versionCode=${NEW_CODE}/" module.prop
    sed -i "s/\"version\": \".*\"/\"version\": \"${NEW_VER}\"/; s/\"versionCode\": [0-9]*/\"versionCode\": ${NEW_CODE}/" update.json
    sed -i "s|/v[0-9.]*/JunkClean-v[0-9.]*|/${NEW_VER}/JunkClean-${NEW_VER}|" update.json
    sed -i "s/^# JunkClean.*/# JunkClean 🧹 ${NEW_VER}/" README.md
    echo "✅ 代码有改动，自动升版本: ${CUR_VER} → ${NEW_VER} (${CUR_CODE}→${NEW_CODE})"
  fi
}

# ---- 二进制就位检查 ----
if [ ! -f bin/curl ] || ! readelf -h bin/curl 2>/dev/null | grep -q AArch64; then
  echo "⚠️ bin/curl 缺失或非 ARM64，先执行 ./build-curl.sh"; exit 1
fi
if [ ! -f bin/cleand ] || ! readelf -h bin/cleand 2>/dev/null | grep -q AArch64; then
  echo "⚠️ bin/cleand 缺失，先交叉编译"; exit 1
fi

auto_bump

VER=$(grep '^version=' module.prop | cut -d= -f2)
OUT="JunkClean-${VER}.zip"
rm -f "$OUT"
zip -r -9 "$OUT" \
  module.prop customize.sh service.sh action.sh uninstall.sh binder.sh cleaner.sh \
  webroot/ bin/ rules/ META-INF/ update.json CHANGELOG.md README.md .gitignore \
  -x 'webroot/_alpine.js' -x 'bin/curl.old' -x '*/.DS_Store' -x '*.bak'
echo "=== $OUT ==="
unzip -l "$OUT" | tail -2
