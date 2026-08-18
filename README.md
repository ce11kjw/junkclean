# JunkClean 🧹 v1.0.0

智能垃圾清理 + 存储维护模块（KernelSU / APatch / Magisk 通用）
本地优先 · 零数据采集 · 删除全程人工确认 · 开源致谢

## ✨ 功能
| 页 | 功能 |
|---|---|
| 🩺 体检 | 分类垃圾统计 / 大文件 Top20(只列不删) / 红黄绿健康度 / 推荐项一键清 / AI 深度建议 |
| 🧹 清理 | 应用缓存 / 系统垃圾 / 安装包压缩包 / 空文件夹 / 社交专项 / SQLite 优化(WAL+VACUUM) |
| 🗂 整理 | 文件分类（跳过未下载完文件） / 重复文件归档(移至 Duplicates/ 不删) |
| 🔧 维护 | fstrim / F2FS 智能 GC / 目录重定向(bind) / 媒体库刷新 |
| ⚙️ 设置 | 定时任务(多任务/热重载) / AI(自填 url·key·model) / 规则编辑器 / 白名单 / 中英双语 / 致谢 |

**安全设计**
- 扫描结果 = 只读清单；删除默认全不勾选 + 二次确认弹窗
- 🔴 红线数据（聊天媒体/下载文件）默认保留，勾选需确认
- 白名单优先级最高；定时自动清理受白名单保护（人工勾选可 force）
- 所有删除写入日志；SQLite VACUUM 前检查磁盘空间
- 安装时检测同类别模块冲突并中止安装

## 📦 安装
1. KSU/AP 管理器 → 刷入 `JunkClean-v1.0.0.zip`（免 recovery，KSU/AP 原生格式）
2. 首次安装会检测 fstrim/GC/清理类冲突模块，发现即中止提示
3. 重启后 cleand 守护自启（127.0.0.1:8801）

## 🖥 使用
- **WebUI**：浏览器打开 `http://127.0.0.1:8801`（KSU 管理器模块 WebUI 入口亦可，若 CSP 受限请用浏览器）
- **快捷动作**：管理器 Action 按钮 = 立即全量清理（受白名单保护）
- **终端**：`su -c /data/adb/modules/junkclean/bin/junkclean <cmd>`
  `clean|scan|classify|duplicate|fstrim|rescan|ai|status`
- **配置/规则**（升级保留）：`/data/adb/junk-cleaner/{config.conf,rules/,scan.json,cleaner.log}`
- **定时任务格式**（tasks.conf，WebUI 内管理）：`enable=1,every=12h,cats=cache,social,sqlite`

## 🤖 AI 深度建议
设置页填 OpenAI 兼容端点（如 https://api.xxx.com/v1 + key + model，也支持本地 Ollama http://127.0.0.1:11434/v1）。
仅发送**聚合统计**（分类大小/剩余空间），文件路径不出设备；手动触发，15s 超时。

## ⚠️ 风险声明
- 删除不可恢复：所有删除需人工确认；建议先开体检看预估值
- 目录重定向为开机 bind mount：Android 12+ FUSE 设备有效；已运行 App 可能需重启生效；失败仅记日志，不影响系统
- 勿与其他 GC/Trim/清理类模块并用（安装时已检测）
- API Key 明文存于 config.conf(0600)，仅发往你填写的地址
- 模块删除无残留（配置保留于 /data/adb/junk-cleaner 可手动删）

## 📚 功能致谢（灵感/思路来源，按功能逐条标注）
- 清理/整理/重定向/自动化架构 → **ClearBox** · FLYCOM-E (GPL-3.0) · github.com/FLYCOM-E/ClearBox；F2FS GC 方案 → 酷安@Amktiao；App → Kr-Script 项目(helloklf)
- 文件分类/下载完整性/重复文件归档 → **Sortify** · xCaptaiN09 (MIT) · github.com/xCaptaiN09/Sortify
- fstrim/F2FS-GC 智能感知（脏段阈值/灭屏充电） → **F2FS-Optimizer** · Coolapk-Code9527（酷安 乄代号9527, MIT）
- 清理规则与定时思路（多源） → Clear-Optimization(DEMONNICA, Apache-2.0) · SmartClear(S123123sd, Apache-2.0) · Cache Cleaner(taamarin) · 小玖清理(Axiaosanjiu) · BasicCleaner(WeirdMidas)
- 整理·清理·重定向三位一体 → **苏柚 SUU** · 回忆溢出工作组 OOM-WG（@梦璃酱 发起/@白彩恋 维护, github.com/OOM-WG/ShiroSU-Utils）；分类规则贡献 → @GunRain(安音咲汀) / 酷安@Luxus_ / 酷安@爱生活的土豆子
- per-app 存储隔离参考 → **存储空间隔离** Storage Redirect · RikkaW & Xingchen (RikkaApps, moe.shizuku.redirectstorage)
- 守护进程+WebUI 架构 → 自家 **ChargeControl** (battd) 经验复用
- 文档参考：AOSP 存储文档 · KernelSU · APatch · SQLite 官方文档

## 👨‍💻 作者
**CE11KJW** · 本项目为原创实现，功能灵感来源如上逐一署名；请尊重各上游许可证。

## License
MIT（本项目代码）—— 致谢列表中的上游项目保留各自许可证。
