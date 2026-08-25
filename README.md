# JunkClean 🧹 v3.1

智能垃圾清理 + 存储维护模块（KernelSU / APatch / Magisk 通用）
本地优先 · 零数据采集 · 删除全程人工确认

## ✨ 功能总览

| 模块 | 功能 |
|---|---|
| 🧹 **12 类清理** | 应用缓存/系统垃圾/安装包/压缩包/缩略图/崩溃日志/临时文件/卸载残留/空文件夹/空文件/社交专项/SQLite |
| 📱 **应用级清理** | 扫描后按应用单独清理缓存 |
| 📁 **大文件** | Top 列表 + 阈值可调 + 筛选 + 归档/删除 |
| 📑 **重复文件** | md5 跨目录 + 预览（保留最大/指定保留）+ 归档/删除 |
| 🗂 **分类整理** | @src/@dest + @map 自定义规则 + 预览 |
| 🛡 **安全** | 白名单/回收站/风险分级/系统保护/规则空检测 |
| 🧠 **AI 助手** | 多轮对话 + 一键采纳 |
| ⏰ **定时任务** | 多任务 + 条件触发（充电/WiFi/空闲）|
| 📊 **数据** | 趋势图表/累计统计/前后对比 |
| ⚙️ **规则** | 模板库 + 导入导出 + 白名单 + 双开关 |
| 🎨 **UI** | 霓虹全息 HUD + 玻璃 + 底部导航三 Tab + 三主题 |

## 🏗 架构

- module.prop：模块元数据
- customize.sh：安装脚本（备份规则 + 生成 sha）
- service.sh：启动 cleand
- cleaner.sh：核心引擎（清理/扫描/分类/去重/AI）
- cleand.c：守护进程（HTTP API @127.0.0.1:46780 + 定时器）
- webroot/index.html：WebUI 单文件
- rules/*.list：清理规则（recurse/no-integrity/high）
- bin/curl：静态 curl（KSU 桥）

## 🔌 API 端点（19 个）

status / scan / progress / clean / check / rules / config / classify(+preview) /
duplicate(+preview) / delbig / bigmove / cleanapp / ai / tasks / log / stats-history

## 📜 规则格式

```
# 清理：路径 [recurse] [no-integrity] [high]
/sdcard/Android/data/*/cache    recurse
# 分类（classify.list）：
@src=/sdcard
@dest=/sdcard/下载
@map=IMG_* /截图照片
jpg jpeg png /图片
```

## 🧪 测试 / 🔧 构建 / 📦 发布

- 测试：`export JC_ADR=/tmp/jc-test; sh cleaner.sh "scan force"`
- 构建：`./bump.sh && bash build.sh`
- 发布：GitHub Release + update.json（KSU 在线更新）
