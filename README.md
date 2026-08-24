# JunkClean 🧹 v1.5.0

智能垃圾清理 + 存储维护模块（KernelSU / APatch / Magisk 通用）
本地优先 · 零数据采集 · 删除全程人工确认 · 开源致谢

## ✨ 功能总览

| 页签 | 功能 |
|---|---|
| 🩺 体检 | 分类垃圾统计 / 大文件 Top20（只列不删）/ 红黄绿健康度 / 推荐项一键清 / AI 深度建议 |
| 🧹 清理 | 应用缓存 / 系统垃圾 / 安装包压缩包 / 空文件夹 / 社交专项 / SQLite 优化（WAL+VACUUM） |
| 🗂 整理 | 文件分类（跳过未下载完文件） / 重复文件归档（移至 Duplicates/，只移不删） |
| 🔧 维护 | fstrim / F2FS 智能 GC（脏段阈值 + 灭屏充电） / 目录重定向（bind mount） / 媒体库刷新 |
| ⚙️ 设置 | 定时任务（多任务 / 热重载） / AI 配置（自填 URL·Key·Model） / 规则编辑器 / 白名单 / 中英双语 / 功能致谢 |

## 🚀 v1.2.0 更新

- ⚡ **do_duplicate 并行化**：`find \| xargs -P5 md5sum`，5 线程同时计算大文件哈希，提速 ~5×
- ⚡ **sqlite_opt 排除系统 DB**：跳过 `/system/` `/data/system/` `/data/misc/` `/data/virtual/` 路径，避免 VACUUM 卡系统进程
- 🎨 **UI 全面打磨**：卡片阴影增强、Tab 底部指示条、按钮动效反馈、输入框焦点高亮
- 🔄 **远程更新**：`updateJson` 配好，KSU 管理器可直接检测新版本
- 🛡️ **健壮性**：cleand 捕获 SIGTERM/SIGINT 优雅退出；子进程 180s 超时强杀兜底
- 🔒 **安全加固**：AI 请求 30s 频率限制（超频返回 429）；delbig 路径注入守卫

## 🛡️ 安全设计

- 扫描结果 = 只读清单；删除默认全不勾选 + 二次确认弹窗
- 🔴 红线数据（聊天媒体 / 下载文件）默认保留，勾选需确认
- 白名单优先级最高；定时自动清理受白名单保护（人工勾选可 force）
- 所有删除写入日志；SQLite VACUUM 前检查磁盘空间 ≥1GB
- 安装时检测同类别模块冲突并中止安装

## 📦 安装

1. 下载 Release 中的 `JunkClean-v1.2.0.zip`
2. KSU / APatch 管理器 → 刷入模块（免 recovery，原生格式；也含 META-INF 兼容 Magisk）
3. 首次安装会检测 fstrim / GC / 清理类冲突模块，发现即中止提示
4. 重启后 cleand 守护自启（127.0.0.1:46780）

## 🖥 使用

- **WebUI**：浏览器打开 `http://127.0.0.1:46780`（KSU 管理器模块 WebUI 入口亦可，若 CSP 受限请用浏览器）
- **快捷动作**：管理器 Action 按钮 = 立即全量清理（受白名单保护）
- **终端命令**：`su -c /data/adb/modules/junkclean/bin/junkclean <cmd>`
  - 可用命令：`clean` / `scan` / `classify` / `duplicate` / `fstrim` / `rescan` / `ai` / `status`
- **配置/规则路径**（升级保留）：
  - `/data/adb/junk-cleaner/config.conf`（权限 0600，含 AI Key）
  - `/data/adb/junk-cleaner/rules/{cache,junk,apk,social,whitelist,classify,bind}.list`
  - `/data/adb/junk-cleaner/scan.json`（体检结果）
  - `/data/adb/junk-cleaner/cleaner.log`（256KB 循环日志）
  - `/data/adb/junk-cleaner/tasks.conf`（定时任务，格式 `enable=1,every=12h,cats=cache,social`，热重载）

## 🤖 AI 深度建议

设置页填写 OpenAI 兼容端点（如 `https://api.xxx.com/v1` + key + model，也支持本地 Ollama `http://127.0.0.1:11434/v1`）。
仅发送**聚合统计**（分类大小/大文件列表/重复项数），不上传任何文件内容。

## ⚠️ 注意事项

- 清理后若 App 登录态失效，通常因缓存被误删，可在白名单排除对应路径
- 目录重定向为开机 bind mount：Android 12+ FUSE 设备有效；已运行 App 可能需重启生效；失败仅记日志
- 勿与其他 GC / Trim / 清理类模块并用（安装时已检测）
- SQLite VACUUM 需磁盘空闲 ≥ 1GB，否则跳过
- API Key 明文存 config.conf（0600），仅发往你填写的地址

## 📚 功能致谢

灵感 / 思路来源，按功能逐条标注（未直接引用代码）：

- 清理 / 整理 / 重定向 / 自动化架构 → **ClearBox** · FLYCOM-E（GPL-3.0）· [github.com/FLYCOM-E/ClearBox](https://github.com/FLYCOM-E/ClearBox)；F2FS GC 方案 → 酷安 @Amktiao；App → Kr-Script 项目（helloklf）
- 文件分类 / 下载完整性 / 重复文件归档 → **Sortify** · xCaptaiN09（MIT）· [github.com/xCaptaiN09/Sortify](https://github.com/xCaptaiN09/Sortify)
- fstrim / F2FS-GC 智能感知（脏段阈值 / 灭屏充电） → **F2FS-Optimizer** · Coolapk-Code9527（酷安 乄代号9527，MIT）
- 清理规则与定时思路（多源） → Clear-Optimization（DEMONNICA，Apache-2.0）· SmartClear（S123123sd，Apache-2.0）· Cache Cleaner（taamarin）· 小玖清理（Axiaosanjiu）· BasicCleaner（WeirdMidas）
- 整理·清理·重定向三位一体 → **苏柚 SUU** · 回忆溢出工作组 OOM-WG（@梦璃酱 发起 / @白彩恋 维护，[github.com/OOM-WG/ShiroSU-Utils](https://github.com/OOM-WG/ShiroSU-Utils)）；分类规则贡献 → @GunRain（安音咲汀）/ 酷安 @Luxus_ / 酷安 @爱生活的土豆子
- per-app 存储隔离参考 → **存储空间隔离（Storage Redirect）** · RikkaW & Xingchen（RikkaApps / He Hanbo，GPL-3.0）· [sr.rikka.app](https://sr.rikka.app/)
- 感谢所有在酷安 / XDA / GitHub 贡献开源清理工具的同好们 🙏

## 📄 License

MIT License — 可自行修改分发，保留本声明即可。
