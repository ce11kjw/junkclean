# Changelog

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
