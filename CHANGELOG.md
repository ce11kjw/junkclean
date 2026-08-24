# Changelog

#
## v1.4.1 (2026-08-24)
- 🐛 修复 GSAP errors=replace 损坏 Alpine 启动代码导致 UI 乱码
- 🔧 移除 GSAP，恢复 Alpine 原生 x-transition


## v1.4.0 (2026-08-24)
- 🎨 frontend-design 文案校准（技术术语→用户语言）+ 空状态行动邀请
- 🎬 GSAP 卡片 stagger 入场（tab 切换错峰上浮 + reduced-motion 尊重）
- 📝 daemon.log 完整生命周期日志（cleand 启动/bind 失败/子进程退出码/WebUI 双切换）
- 🏷️ ponytail 标注 7 处（简化点天花板+升级路径）
- 🐛 死代码清理（taskid/lastline/b/r）

# Changelog

## v1.4.2 (2026-08-24)
- 🔧 build.sh 版本硬约束检查（拒绝打包如果版本未变且有新 commit）
- 🔧 bump.sh 一键 patch 版本 bump 脚本


## v1.4.1 (2026-08-24)
- 🐛 修复 GSAP errors=replace 损坏 Alpine 启动代码导致 UI 乱码
- 🔧 移除 GSAP，恢复 Alpine 原生 x-transition


## v1.3.0 (2026-08-24)
- 🎨 **Premium UI 重设计**：Ethereal Glass 玻璃拟态 / 浮岛 Header+Tab / Asymmetrical Bento / 入口动效 / Button-in-Button / 噪点纹理
- 🐛 **全量审计修复**：do_duplicate tab分隔、do_classify 子shell计数、force 系统目录保护、runner fork 检查、strncpy 补0、api_config 写失败处理等 10+ 项
- 🔧 **curl 完全静态**（-all-static），修复 Android 无法运行问题
- 🔌 **端口改为 46780**
- 📝 README 致谢修正（RikkaW & Xingchen）+ 规则路径修正
- 🛠️ build-curl.sh 固化编译流程

# Changelog


## v1.4.1 (2026-08-24)
- 🐛 修复 GSAP errors=replace 损坏 Alpine 启动代码导致 UI 乱码
- 🔧 移除 GSAP，恢复 Alpine 原生 x-transition


## v1.2.0 (2026-08-20)
- ⚡ do_duplicate 并行化：xargs -P5 md5sum，5 线程同时计算哈希
- ⚡ sqlite_opt 排除系统 DB：/system/|/data/system/|/data/misc/|/data/virtual/ 路径跳过
- 🎨 UI 全面打磨（阴影/指示条/动效/聚焦）
- 🔄 updateJson 远程更新
- 🛡️ SIGTERM/SIGINT 优雅退出 + 子进程 180s 超时
- 🔒 AI 请求 30s 频率限制（超频 429）
- 📝 致谢区改为普通卡片移至设置页底部

## v1.0.0 (2026-08-19)
- 首发
