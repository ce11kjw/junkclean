#!/bin/sh
# bump.sh - patch 版本一键 bump（v1.4.2 → v1.4.3）
# 用户硬规则：动一次代码就更新版本号
set -e
cur_v=$(grep '^version=' module.prop | cut -d= -f2 | sed 's/^v//')
cur_code=$(grep '^versionCode=' module.prop | cut -d= -f2)
new_v=$(echo "$cur_v" | awk -F. '{print $1"."$2"."$3+1}')
new_code=$((cur_code + 1))
sed -i "s/^version=.*/version=v${new_v}/" module.prop
sed -i "s/^versionCode=.*/versionCode=${new_code}/" module.prop
sed -i "s|^# JunkClean 🧹 v.*|# JunkClean 🧹 v${new_v}|" README.md
# update.json 用 python 原子重建（保证合法 JSON）
python3 - "$new_v" "$new_code" << 'PY'
import json, sys
v, code = sys.argv[1], int(sys.argv[2])
d = {
  "version": v,
  "versionCode": code,
  "zipUrl": f"https://github.com/ce11kjw/junkclean/releases/download/{v}/JunkClean-{v}.zip",
  "changelog": "https://raw.githubusercontent.com/ce11kjw/junkclean/main/CHANGELOG.md"
}
json.dump(d, open('update.json','w'), indent=2)
print(f"✓ update.json rebuilt: {v} / {code}")
PY
echo "✓ bumped: v${cur_v} (${cur_code}) → v${new_v} (${new_code})"
