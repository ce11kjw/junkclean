#!/bin/sh
# bump.sh - patch 版本一键 bump（v1.4.1 → v1.4.2）
# 用户硬规则：动一次代码就更新版本号
set -e
cur_v=$(grep '^version=' module.prop | cut -d= -f2 | sed 's/^v//')
cur_code=$(grep '^versionCode=' module.prop | cut -d= -f2)
new_v=$(echo "$cur_v" | awk -F. '{print $1"."$2"."$3+1}')
new_code=$((cur_code + 1))
sed -i "s/^version=.*/version=v${new_v}/" module.prop
sed -i "s/^versionCode=.*/versionCode=${new_code}/" module.prop
sed -i "s/\"version\": \".*\"/\"version\": \"v${new_v}\"/" update.json
sed -i "s/\"versionCode\": .*/\"versionCode\": ${new_code}/" update.json
sed -i "s|/v[0-9.]*\.zip\"|/v${new_v}.zip\"|" update.json
sed -i "s|JunkClean-v[0-9.]*\.zip|JunkClean-v${new_v}.zip|g" update.json
sed -i "s|^# JunkClean 🧹 v.*|# JunkClean 🧹 v${new_v}|" README.md
echo "✓ bumped: v${cur_v} (${cur_code}) → v${new_v} (${new_code})"
