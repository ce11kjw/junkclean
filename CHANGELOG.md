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
