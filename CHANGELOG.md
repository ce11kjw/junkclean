# Changelog

## v1.3.0 (2026-08-24)
- 🎨 **Premium UI 重设计**：Ethereal Glass 玻璃拟态 / 浮岛 Header+Tab / Asymmetrical Bento / 入口动效 / Button-in-Button / 噪点纹理
- 🐛 **全量审计修复**：do_duplicate tab分隔、do_classify 子shell计数、force 系统目录保护、runner fork 检查、strncpy 补0、api_config 写失败处理等 10+ 项
- 🔧 **curl 完全静态**（-all-static），修复 Android 无法运行问题
- 🔌 **端口改为 46780**
- 📝 README 致谢修正（RikkaW & Xingchen）+ 规则路径修正
- 🛠️ build-curl.sh 固化编译流程

# Changelog

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
