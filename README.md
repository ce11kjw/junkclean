# JunkClean 🧹 v1.6.2

智能垃圾清理 + 存储维护模块（KernelSU / APatch / Magisk 通用）
本地优先 · 零数据采集 · 删除全程人工确认 · 开源致谢

## ✨ 功能总览

| 页签 | 功能 |
|---|---|
| 🩺 体检 | 分类垃圾统计 / 大文件 Top20（只列不删）/ 红黄绿健康度 / 推荐项一键清 / AI 深度建议 |
| 🧹 清理 | 应用缓存 / 系统垃圾 / 安装包压缩包 / 空文件夹 / 社交专项 / 压缩数据库文件 |
| 🗂 整理 | 文件分类（跳过未下载完文件） / 重复文件归档（移至 Duplicates/，只移不删） |
| 🔧 维护 | fstrim / F2FS 智能 GC（脏段阈值 + 灭屏充电） / 目录重定向（bind mount） / 媒体库刷新 |
| ⚙️ 设置 | 定时任务（多任务 / 热重载） / AI 配置（自填 URL·Key·Model） / 规则编辑器 / 白名单 / 中英双语 / 功能致谢 |

## 🚀 版本更新

### v1.5.0 · 全面美化
- 🎨 健康度数字呼吸发光 / 卡片毛玻璃 / 按钮光晕扫过 / 17 个 SVG 图标替换 emoji / 卡片 hover 上浮
- 纯 CSS 零依赖，尊重 prefers-reduced-motion

### v1.4.8 · 严审修复
- 🔴 修复 `clean` 命令不执行（参数传递 bug，此前一键清理实际无效）
- force 走 body 支持 / delbig 路径穿越守卫

### v1.4.4 · 启动日志
- 守护启动失败也有日志（shell 写，不依赖 cleand）

### v1.4.0 · 日志+审计
- daemon.log 生命周期日志 + WebUI 双日志切换 / 文案用户语言 / 死代码清理

### v1.3.0 · Premium UI
- Ethereal Glass 玻璃拟态重设计 / 全量审计修复 / curl 完全静态 / 端口 46780

### v1.2.0 · 性能优化
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

## 🧩 兼容性（root 管理器）

| 管理器 | 支持 | 说明 |
|---|---|---|
| Magisk | ✅ v20.4+ | META-INF 官方安装器，原生格式通用 |
| KernelSU | ✅ | 原生模块格式（module.prop / customize.sh / service.sh / action.sh / updateJson）|
| APatch | ✅ | 原生格式 + action.sh（已真机验证）|
| Zygisk | — | 无需适配（不注入 app 进程，无 zygisk 目录）|
| mountify | — | 无需适配（不替换 system 文件）|
| SELinux | ✅ | 守护以 su 域运行，已真机验证端口/目录访问 |

- **WebUI**：浏览器 `http://127.0.0.1:46780` 或管理器内置 WebUI 入口均可（API 绝对地址 + CORS 已适配）
- 更新检测：KSU/APatch 管理器读 `module.prop` 的 `updateJson` 指向的 update.json

## 📦 安装

1. 下载 Release 中的 `JunkClean-v1.5.1.zip`
2. KSU / APatch 管理器 → 刷入模块（免 recovery，原生格式；也含 META-INF 兼容 Magisk）
3. 首次安装会检测 fstrim / GC / 清理类冲突模块，发现即中止提示
4. 重启后 cleand 守护自启（127.0.0.1:46780）

## 🖥 使用

- **WebUI**（两种方式皆可）：
  - 浏览器打开 `http://127.0.0.1:46780`
  - **KSU / APatch 管理器内置 WebUI 入口**（webroot 已适配：API 走绝对地址 + CORS）
- **快捷动作**：管理器 Action 按钮 = 立即全量清理（受白名单保护）
- **终端命令**：`su -c /data/adb/modules/junkclean/bin/junkclean <cmd>`
  - 可用命令：`clean` / `scan` / `classify` / `duplicate` / `fstrim` / `rescan` / `ai` / `status`
- **配置/规则路径**（升级保留）：
  - `/data/adb/junk-cleaner/config.conf`（权限 0600，含 AI Key）
  - `/data/adb/junk-cleaner/rules/{cache,junk,apk,social,whitelist,classify,bind}.list`
  - `/data/adb/junk-cleaner/scan.json`（体检结果）
  - `/data/adb/junk-cleaner/cleaner.log`（清理操作日志）
  - `/data/adb/junk-cleaner/daemon.log`（守护生命周期日志，启动失败也记录）
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
- 清理规则与定时思路（多源）：深度规则库参考 **Ghost Cleaner**（酷安 柒黑，WebUI 多线程清理模块）
- 清理规则与定时思路（多源） → Clear-Optimization（DEMONNICA，Apache-2.0）· SmartClear（S123123sd，Apache-2.0）· Cache Cleaner（taamarin）· 小玖清理（Axiaosanjiu）· BasicCleaner（WeirdMidas）
- 整理·清理·重定向三位一体 → **苏柚 SUU** · 回忆溢出工作组 OOM-WG（@梦璃酱 发起 / @白彩恋 维护，[github.com/OOM-WG/ShiroSU-Utils](https://github.com/OOM-WG/ShiroSU-Utils)）；分类规则贡献 → @GunRain（安音咲汀）/ 酷安 @Luxus_ / 酷安 @爱生活的土豆子
- per-app 存储隔离参考 → **存储空间隔离（Storage Redirect）** · RikkaW & Xingchen（RikkaApps / He Hanbo，GPL-3.0）· [sr.rikka.app](https://sr.rikka.app/)
- 感谢所有在酷安 / XDA / GitHub 贡献开源清理工具的同好们 🙏

## 📄 License

MIT License — 可自行修改分发，保留本声明即可。
