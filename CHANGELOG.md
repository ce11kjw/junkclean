# Changelog

## v1.6.2 (2026-08-25)
- 🔴 前端严审修复：saveTasks 裸 $() ReferenceError（无 jQuery）→ 定时任务保存/开关失效 → 删死代码
- 🔴 saveAISettings 整文件覆盖 config.conf → 用户 cat_* 配置丢失 → 改为读现有+替换ai_行+回写
- ponytail: UI 用可选链 ?.（需 Android 10+ WebView）

## v1.6.1 (2026-08-25)
- 🔴 全功能测试发现修复：load_rules 的 #RED 行被 # 注释分支拦截 → 红线系统失效（REDR 空）→ #RED 优先匹配
- 🔴 #RED 中文标题行被误当红线路径 → 清理为非路径标题用 # 注释

## v1.6.0 (2026-08-25)
- 🔴 白名单机制完善：delbig 大文件删除 / classify 分类移动 / duplicate 重复归档 全部加白名单保护（此前绕过）
- 🔴 白名单大小写不敏感（/sdcard 大小写不敏感，微信 tencent vs Tencent 此前白名单失效）

## v1.5.9 (2026-08-25)
- 🔴 修复管理器 WebUI 适配层 2 坑：裸 curl 命令不存在（改模块完整路径）+ POST body 丢失（透传）
- 其他确认无坑项：单文件无外部资源/无混合内容/cleand 绑 127.0.0.1 root 可访问/日志轮转正常

## v1.5.8 (2026-08-25)
- 🔴 修复管理器 WebUI 按钮失效：APatch/KSU WebView 页面为 https://mui.kernelsu.org，fetch http://127.0.0.1 被混合内容拦截
- ✅ $f 加 JS 桥适配：window.ksu.exec() 执行 curl 绕过（浏览器场景仍走 fetch）

## v1.5.7 (2026-08-25)
- ✏️ 文案用户视角：'存储体检'→'垃圾扫描'（scan 按钮/页签），'开始体检'→'扫描垃圾'

## v1.5.6 (2026-08-25)
- 📚 规则库扩充（参考 Ghost Cleaner 等三框架清理模块）：junk 28 条（厂商日志/杂项）/ social 48+15 红线（微信QQ深度）/ cache 25 条（广告SDK通配）
- 🔴 修复红线规则格式 bug（#RED 当注释标题→聊天媒体误入普通可删规则）→ 全部改为 #RED /path 单行
- 规则分类安全：普通 0 聊天媒体，红线 15 条全为媒体类

## v1.5.5 (2026-08-25)
- 🌐 WebUI 适配 KSU/APatch 管理器内置入口（官方文档核实）：API 改绝对地址 http://127.0.0.1:46780 + cleand 加 CORS 头
- 📚 README 修正 WebUI 说明（此前误判'勿用管理器入口'）

## v1.5.4 (2026-08-25)
- 📚 root 管理器文档严审：修正 KSU/APatch 内置 WebUI 入口误导（API 端口不匹配）+ 新增兼容性矩阵（Magisk/KSU/APatch/Zygisk/mountify/SELinux）

## v1.5.3 (2026-08-25)
- 🐛 前端 poll 不更新 daemon 状态（页面打开时守护重启中→永远显示未启动）→ poll 同时拉 /api/status

## v1.5.2 (2026-08-25)
- 🐛 cleaner.sh 严审修复4项：duplicate xargs空格拆词→-print0/-0；scan find引号；白名单精确匹配；fstrim gc_urgent多分区

## v1.5.1 (2026-08-25)
- 📚 文档严审修复：README 全面更新（下载链接/功能表/版本史/日志路径）+ CHANGELOG 补 v1.4.8/v1.5.0 + git tag 补同步

## v1.5.0 (2026-08-25)
- 🎨 frontend-design 全面美化（纯CSS零依赖）：健康数字发光/卡片毛玻璃/按钮光晕/17个SVG图标/卡片hover上浮

## v1.4.8 (2026-08-25)
- 🔴 严审修复：clean 命令不执行（execl 单参 vs case 匹配）→ set -- 拆分；force body 支持；delbig .. 检查



## v1.4.7 (2026-08-24)
- 🔧 bump.sh 修复 update.json 缺 v 前缀（zipUrl 404）
- 🔧 CHANGELOG 整理（v1.4.2~v1.4.6 补全）

## v1.4.6 (2026-08-24)
- 🐛 修复 UI `{{}}` 模板插值不渲染，全部替换为 x-text

## v1.4.5 (2026-08-24)
- 🔧 安装时全部文件 777 权限 + ui_print 显示权限成功/失败反馈

## v1.4.4 (2026-08-24)
- 🔧 service.sh/action.sh 启动日志不依赖 cleand（启动失败也有日志）

## v1.4.3 (2026-08-24)
- 🐛 修复 update.json JSON 损坏 / CHANGELOG 重复 / build.sh 双重逻辑冲突

## v1.4.2 (2026-08-24)
- 🔧 build.sh 版本硬约束检查 + bump.sh 一键 patch bump

## v1.4.1 (2026-08-24)
- 🐛 移除 GSAP（errors=replace 损坏 Alpine 启动代码 → UI 乱码）

## v1.4.0 (2026-08-24)
- 🎨 frontend-design 文案校准 + GSAP 卡片 stagger + daemon.log 生命周期日志 + ponytail 标注 + 死代码清理 + set_perm bin/

## v1.3.0 (2026-08-24)
- 🎨 Premium UI 重设计（Ethereal Glass）+ 全量审计修复 10+ 项 + curl 完全静态 + 端口 46780

## v1.2.0 (2026-08-24)
- ⚡ 性能优化（do_duplicate xargs -P5 并行 / sqlite 排除系统 DB）+ UI 打磨 + updateJson + 健壮性 + AI 限频

## v1.1.0 (2026-08-24)
- 🔧 updateJson 远程更新 / UI 打磨 / 性能优化 / 健壮性

## v1.0.0 (2026-08-24)
- 🚀 首版：清理 6 类 / 体检 / AI 建议 / 文件分类 / 重复归档 / fstrim / 目录重定向 / 定时 / 多语言 / 日志
