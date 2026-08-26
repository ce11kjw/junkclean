# JunkClean 🧹 v3.3.4

智能垃圾清理 + 存储维护模块（KernelSU / APatch / Magisk 通用）
本地优先 · 零数据采集 · 删除全程人工确认

## ✨ 功能总览

| 模块 | 功能 |
|---|---|
| 🧹 **12 类清理** | 应用缓存/系统垃圾/安装包/压缩包/缩略图/崩溃日志/临时文件/卸载残留/空文件夹/空文件/社交专项/SQLite |
| 📱 **应用级清理** | 扫描后按应用单独清理缓存 |
| 📁 **大文件** | Top 列表 + 阈值可调 + 筛选 + 归档/删除 |
| 📑 **重复文件** | md5 跨目录 + 预览（保留最大/指定保留）+ 归档/删除 |
| 🗂 **分类整理** | @src/@dest + @map 自定义规则 + @exclude 排除 + 预览 |
| 📡 **自动监控** | 30s 轮询新文件 → 自动 classify + 媒体库刷新（API 开关/目录管理）|
| 🛡 **安全** | 白名单/回收站/风险分级/系统保护/规则空检测/路径双开关 |
| 🧠 **AI 助手** | 多轮对话 + 一键采纳 |
| ⏰ **定时任务** | 多任务 + 条件触发（充电/WiFi/空闲）|
| 📊 **数据** | 趋势图表/累计统计/前后对比 |
| ⚙️ **规则** | 模板库 + 导入导出 + 白名单 + 双开关（子目录/完整性）|
| 🎨 **UI** | 霓虹全息 HUD + 玻璃 + 底部导航三 Tab + 三主题 + 中英双语 |
| 🧪 **正则测试** | 内置正则测试工具（实时匹配 + 捕获组）|

## 🏗 架构

- module.prop：模块元数据
- customize.sh：安装脚本（备份规则 + 生成 sha）
- service.sh：启动 cleand
- cleaner.sh：核心引擎（清理/扫描/分类/去重/AI/exclude）
- cleand.c：守护进程（HTTP API @127.0.0.1:46780 + 定时器 + 监控线程）
- webroot/index.html：WebUI 骨架（17KB）
- webroot/style.css：WebUI 样式（17KB）
- webroot/app.js：WebUI 逻辑（30KB）
- rules/*.list：清理规则（recurse/no-integrity/high）
- bin/cleand：ARM64 静态守护进程（自动监控）
- bin/curl：静态 curl（KSU 桥）

## 🔌 API 端点（20 个）

status / scan / progress / clean / check / rules / config / classify(+preview) /
duplicate(+preview) / delbig / bigmove / cleanapp / ai / tasks / log / stats-history / **monitor**

## 📜 规则格式

```
# 清理规则：路径 [recurse] [no-integrity] [high]
/sdcard/Android/data/*/cache    recurse
/data/data/*/cache              no-integrity
/sdcard/Download/*.zip          high

# 分类整理（classify.list）：
@src=/sdcard                    # 源目录（默认 /sdcard）
@dest=/sdcard/下载               # 目标根（默认 下载）
@exclude=*.part                 # 排除模式（glob，支持多条）
@exclude=*.tmp
@map=IMG_* /截图照片             # 自定义模式规则（放最前优先）
jpg jpeg png /图片              # 扩展名 → 子目录
```

## 📡 自动监控配置

```
POST /api/monitor  {"on":1}             # 开启监控
POST /api/monitor  {"add":"/sdcard/Download"}  # 添加目录
POST /api/monitor  {"remove":"/sdcard/Download"} # 移除目录
GET  /api/monitor                      # 查询状态
```

监控线程每 30s 扫描新文件 → 自动 classify 整理 → 媒体库刷新（am broadcast）。

## 🧪 测试 / 🔧 构建 / 📦 发布

- 测试：`export JC_ADR=/tmp/jc-test; sh cleaner.sh "scan force"`
- 交叉编译：`aarch64-linux-gnu-gcc -O2 -static -o bin/cleand cleand.c -pthread`
- 构建：`./bump.sh && bash build.sh`
- 发布：GitHub Release + update.json（KSU 在线更新）
