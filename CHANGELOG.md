# Changelog

## v2.0.6 (2026-08-25)
- 🗜 压缩包后缀全量覆盖（24种：zip/7z/rar/tar/gz/xz/bz2/tgz/tbz2/zst/lz4/cab/arj/iso/img 等）+ 安装包补全(xapk/apks/apex)
- 📁 classify 分类后缀全量补全（图片18/视频14/音乐13/文档16/压缩包29）
- 🔴 修复多分类清理失效：cats 逗号→空格（for c in $cats 按空白拆，此前 zip,apk 只跑第一个）
- 🔀 do_clean 支持文件级规则（*.zip→find父目录 -name）+ set -f 防 glob 展开

## v2.0.5 (2026-08-25)
- 🔀 清理规则每条路径独立双开关：子目录 recurse（默认关）/ 完整性 no-integrity（默认开）
- 🔍 完整性检测：跳过 .part/.crdownload/.tmp/.partial/.downloading/.!q/.aria2
- ⚠️ 清理清单分类卡片：每条规则路径一行 + 双开关（点击切换写回）+ 不存在红色标注
- 🆕 cleand /api/check 路径存在性端点
- 🐛 修复：while read 管道计数丢失（改临时文件读行）、RFLAGS 前导|导致标志解析错

## v2.0.4 (2026-08-25)
- 🗜 压缩包独立分类（zip/rar/7z/tar/gz/xz 单独分类，与安装包拆分）
- 📁 文件分类重构：@src 指定源目录（默认 /sdcard，可自定义）+ @dest 指定目标根（默认 /sdcard/下载）
- 分类规则每行可自定义目标子目录（相对 @dest 或绝对路径）
- 修复 /子目录 被当绝对路径的 bug

## v2.0.3 (2026-08-25)
- ⏰ 补齐定时任务 UI（v2.0 缺口）：任务列表/开关/删除 + 添加（间隔或每日）+ 后端 tasks API 已通
- 📁 大文件清单项：scan.big 接入清单，勾选走 delbig（白名单保护）
- 修复 renderBig 局部变量 bug

## v2.0.2 (2026-08-25)
- 🔴 修复大目录 scan 统计错误：规则 glob 展开数万路径超 ARG_MAX → count=0 → 改目录级（去尾部 /*）find/du 内部遍历
- ✅ 43 万文件 scan 2.6s，统计完全正确（此前 5000 文件即 3.9s 且大目录 count=0）

## v2.0.1 (2026-08-25)
- ⚡ 性能优化：do_scan find 合并（一次 find 替代逐路径 for 循环），5000 文件 3.9s→0.156s（25×）
- du 已合并（v1.6.3 恢复），fork 数从 2×路径数 减到 2

## v2.0.0 (2026-08-25)
- 🎉 全新 WebUI 重构：原生 JS 单文件 + 线性流程 + 水波纹 + insets.css + 双通道 api（ksu.exec/fetch）
- 零框架零依赖（弃 Alpine，根治所有事件绑定/模板渲染问题）
- 线性流程：首页 → 扫描 → 清单 → 全屏确认 → 清理 → 完成（替代仪表盘）
- 新交互：全屏确认弹层替代小弹窗、大触控区卡片根治点不了、水波纹原生事件委托
- 后端保留（cleand/cleaner.sh 不变，功能 20+ 全部保留）

## v1.6.3 (2026-08-25)
- 🛡️ timer daily 定时任务防抖（同一 HH:MM 不重复触发）
- ponytail: 规则文件空格分隔限制标注（含空格自定义规则不精确，内置规则无空格不受影响）

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
