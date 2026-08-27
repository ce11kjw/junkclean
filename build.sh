#!/bin/bash
# JunkClean 一键构建发布：版本递增 → 编译 → 打包 → 推送 GitHub
set -e
cd "$(dirname "$0")"

# 读当前版本
VER=$(grep '^version=' module.prop | cut -d= -f2)
VC=$(grep '^versionCode=' module.prop | cut -d= -f2)

# 递增（patch +1, versionCode +1）
BASE=${VER%.*}
PATCH=$((${VER##*.} + 1))
VER="${BASE}.${PATCH}"
VC=$((VC + 1))

# 同步 main.go 内嵌版本
sed -i "s/ver      = \"[^\"]*\"/ver      = \"$VER\"/; s/verCode  = [0-9]*/verCode  = $VC/" main.go
sed -i "s/^version=.*/version=$VER/; s/^versionCode=.*/versionCode=$VC/" module.prop

# 编译
echo "==> 编译 v$VER"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o system/bin/junkclean .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/junkclean-x64 .

# 打包
rm -f JunkClean-*.zip
ZIP="JunkClean-v${VER}.zip"
zip -r "$ZIP" module.prop system META-INF scripts customize.sh action.sh service.sh uninstall.sh webroot icon >/dev/null

# update.json（OTA）
SHA256=$(sha256sum "$ZIP" | cut -d" " -f1)
cat > update.json <<EOF
{
  "version": "$VER",
  "versionCode": $VC,
  "zipUrl": "https://raw.githubusercontent.com/ce11kjw/junkclean/main/$ZIP",
  "sha256": "$SHA256",
  "changelog": "${1:-v$VER 自动构建更新}"
}
EOF

# 追加 CHANGELOG（$1 为可选更新说明）
DESC="${1:-自动构建更新}"
cat > /tmp/jc-entry.md <<EOF
## v$VER ($(date +%Y-%m-%d))
- $DESC

EOF
python3 -c "
c = open('CHANGELOG.md').read()
entry = open('/tmp/jc-entry.md').read()
idx = c.find('## v')
c = c[:idx] + entry + c[idx:] if idx >= 0 else '# Changelog\n\n' + entry + c
open('CHANGELOG.md','w').write(c)
"
rm -f /tmp/jc-entry.md

# 提交推送
git add -A
git commit -m "release v$VER (versionCode $VC)" >/dev/null
git push origin main

echo "✅ v$VER (vc=$VC) -> $ZIP"
