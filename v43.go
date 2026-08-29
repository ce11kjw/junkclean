package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// v4.3.0 对齐 APP：定时目录清理 / 保护路径 / 规则库
// 端点：/api/cleandirs  /api/protected  /api/rulelib

const SDROOT = "/sdcard"

// startScheduleLoop 后台定时循环：按 cfg.ScheduleMin 触发目录清理
func startScheduleLoop() {
	go func() {
		for {
			time.Sleep(60 * time.Second)   // 每分钟检查一次
			cfgLock.RLock()
			on, min := cfg.ScheduleOn, cfg.ScheduleMin
			cfgLock.RUnlock()
			if !on {
				continue
			}
			if min < 5 {
				min = 360
			}
			if time.Since(lastSchedRun) < time.Duration(min)*time.Minute {
				continue
			}
			lastSchedRun = time.Now()
			freed, count := runCleanDirs()
			if count > 0 {
				logLine(fmt.Sprintf("定时清理完成: %d 项, 释放 %s", count, fmtSize(freed)))
			}
		}
	}()
}

var lastSchedRun time.Time

func registerV43(mux *http.ServeMux) {
	startScheduleLoop()
	mux.HandleFunc("/api/cleandirs", corsMiddleware(apiCleanDirs))
	mux.HandleFunc("/api/protected", corsMiddleware(apiProtected))
	mux.HandleFunc("/api/rulelib", corsMiddleware(apiRuleLib))
}

// ---------- 保护路径匹配（支持通配符 *） ----------

// matchProtRule 判断 target 是否命中规则（* 匹配任意字符，含 /）
func matchProtRule(target, rule string) bool {
	if rule == "" {
		return false
	}
	if strings.Contains(rule, "*") {
		// 通配符转正则
		parts := strings.Split(rule, "*")
		for i := range parts {
			parts[i] = regexp.QuoteMeta(parts[i])
		}
		rx := ".*" + strings.Join(parts, ".*") + ".*"
		ok, err := regexp.MatchString(rx, target)
		return err == nil && ok
	}
	if target == rule {
		return true
	}
	return strings.Contains(target, "/"+rule+"/") || strings.HasSuffix(target, "/"+rule)
}

// isProtectedPath 检查路径是否受保护（读 cfg.Protected，为空则用内置默认）
func isProtectedPath(p string) bool {
	cfgLock.RLock()
	list := cfg.Protected
	cfgLock.RUnlock()
	if len(list) == 0 {
		list = defaultProtected
	}
	for _, r := range list {
		if matchProtRule(p, r) {
			return true
		}
	}
	return false
}

// 出厂默认保护路径
var defaultProtected = []string{
	"DCIM", "Pictures", "Movies", "Music", "Download", "Documents",
	"tencent/MicroMsg", "MIUI/backup", "backups", "Backup",
	"Android/media", "WhatsApp", "Telegram",
	".keys", "keystore", ".ssh",
}

// ---------- /api/cleandirs：定时清理目录 ----------

func apiCleanDirs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string `json:"action"` // list | add | remove | clear | run | toggle | interval
		Path   string `json:"path"`
		Index  int    `json:"index"`
		On     bool   `json:"on"`
		Min    int    `json:"min"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	switch req.Action {
	case "list":
		cfgLock.RLock()
		dirs, on, min := cfg.CleanDirs, cfg.ScheduleOn, cfg.ScheduleMin
		cfgLock.RUnlock()
		if min == 0 {
			min = 360
		}
		writeJSON(w, 200, map[string]any{
			"dirs": dirs, "scheduleOn": on, "scheduleMin": min,
		})

	case "add":
		p := strings.TrimSpace(req.Path)
		if p == "" {
			writeJSON(w, 400, map[string]string{"error": "路径为空"})
			return
		}
		// 尾斜杠规则：带 / = 只清内容(0)；不带 = 删整个目录(1)
		delItself := !strings.HasSuffix(p, "/")
		clean := strings.TrimRight(p, "/")
		if !filepath.IsAbs(clean) {
			clean = filepath.Join(SDROOT, clean)
		}
		st, err := os.Stat(clean)
		if err != nil || !st.IsDir() {
			writeJSON(w, 400, map[string]string{"error": "目录不存在：" + clean})
			return
		}
		flag := "0"
		if delItself {
			flag = "1"
		}
		line := clean + "|" + flag
		cfgLock.Lock()
		// 去重
		found := false
		for i, d := range cfg.CleanDirs {
			if strings.HasPrefix(d, clean+"|") {
				cfg.CleanDirs[i] = line
				found = true
				break
			}
		}
		if !found {
			cfg.CleanDirs = append(cfg.CleanDirs, line)
		}
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]any{"status": "ok", "delItself": delItself})

	case "remove":
		cfgLock.Lock()
		if req.Index >= 0 && req.Index < len(cfg.CleanDirs) {
			cfg.CleanDirs = append(cfg.CleanDirs[:req.Index], cfg.CleanDirs[req.Index+1:]...)
		}
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]string{"status": "ok"})

	case "clear":
		cfgLock.Lock()
		cfg.CleanDirs = nil
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]string{"status": "ok"})

	case "toggle":
		cfgLock.Lock()
		cfg.ScheduleOn = req.On
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]any{"status": "ok", "on": req.On})

	case "interval":
		if req.Min < 5 {
			req.Min = 5
		}
		cfgLock.Lock()
		cfg.ScheduleMin = req.Min
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]any{"status": "ok", "min": req.Min})

	case "run":
		freed, count := runCleanDirs()
		writeJSON(w, 200, map[string]any{
			"status": "ok", "freed": freed, "count": count,
		})

	default:
		writeJSON(w, 400, map[string]string{"error": "unknown action"})
	}
}

// runCleanDirs 执行已保存目录清理，返回释放字节数与项数
func runCleanDirs() (int64, int) {
	cfgLock.RLock()
	dirs := append([]string{}, cfg.CleanDirs...)
	cfgLock.RUnlock()

	var freed int64
	count := 0
	var survivors []string

	for _, line := range dirs {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		p := parts[0]
		delItself := len(parts) > 1 && parts[1] == "1"

		if isProtectedPath(p) {
			logLine(fmt.Sprintf("定时清理跳过受保护路径: %s", p))
			survivors = append(survivors, line)
			continue
		}
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			continue
		}

		if delItself {
			sz, _ := dirSize(p)
			if err := os.RemoveAll(p); err == nil {
				freed += sz
				count++
				logLine(fmt.Sprintf("定时清理删除目录: %s (%d bytes)", p, sz))
				// 删掉的目录从列表移除
				continue
			}
			survivors = append(survivors, line)
		} else {
			entries, err := os.ReadDir(p)
			if err != nil {
				survivors = append(survivors, line)
				continue
			}
			for _, e := range entries {
				child := filepath.Join(p, e.Name())
				if isProtectedPath(child) {
					continue
				}
				sz := int64(0)
				if e.IsDir() {
					sz, _ = dirSize(child)
				} else if fi, err := e.Info(); err == nil {
					sz = fi.Size()
				}
				if err := os.RemoveAll(child); err == nil {
					freed += sz
					count++
				}
			}
			logLine(fmt.Sprintf("定时清理清空目录内容: %s", p))
			survivors = append(survivors, line)
		}
	}

	cfgLock.Lock()
	cfg.CleanDirs = survivors
	cfgLock.Unlock()
	saveConfig()
	return freed, count
}

// ---------- /api/protected：保护路径管理 ----------

func apiProtected(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string `json:"action"` // list | add | remove | reset
		Path   string `json:"path"`
		Index  int    `json:"index"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	switch req.Action {
	case "list":
		cfgLock.RLock()
		list := cfg.Protected
		cfgLock.RUnlock()
		if len(list) == 0 {
			// 首次：写入默认
			cfgLock.Lock()
			cfg.Protected = append([]string{}, defaultProtected...)
			list = cfg.Protected
			cfgLock.Unlock()
			saveConfig()
		}
		writeJSON(w, 200, map[string]any{"protected": list, "defaults": len(defaultProtected)})

	case "add":
		p := strings.TrimSpace(req.Path)
		if p == "" {
			writeJSON(w, 400, map[string]string{"error": "路径为空"})
			return
		}
		cfgLock.Lock()
		exists := false
		for _, x := range cfg.Protected {
			if x == p {
				exists = true
				break
			}
		}
		if !exists {
			cfg.Protected = append(cfg.Protected, p)
		}
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]string{"status": "ok"})

	case "remove":
		cfgLock.Lock()
		if req.Index >= 0 && req.Index < len(cfg.Protected) {
			cfg.Protected = append(cfg.Protected[:req.Index], cfg.Protected[req.Index+1:]...)
		}
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]string{"status": "ok"})

	case "reset":
		cfgLock.Lock()
		cfg.Protected = append([]string{}, defaultProtected...)
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]any{"status": "ok", "count": len(defaultProtected)})

	default:
		writeJSON(w, 400, map[string]string{"error": "unknown action"})
	}
}

// ---------- /api/rulelib：清理规则库（对齐 APP RuleEngine） ----------

type CleanRule struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Risk         string   `json:"risk"`       // low / mid / high
	TargetType   string   `json:"targetType"` // file / dir / any
	PathContains []string `json:"pathContains"`
	NameStarts   []string `json:"nameStarts"`
	NameEnds     []string `json:"nameEnds"`
	Exclusions   []string `json:"exclusions"`
	Regex        []string `json:"regex"`
	MinSize      int64    `json:"minSize"`
	MaxSize      int64    `json:"maxSize"`
	MinAgeDays   int64    `json:"minAgeDays"`
	Enabled      bool     `json:"enabled"`
}

func ruleLibPath() string { return filepath.Join(SDROOT, ".junkclean", "rules.json") }

var defaultRules = []CleanRule{
	{ID: "log_files", Label: "日志文件", Risk: "low", TargetType: "file",
		NameEnds: []string{".log", ".log.1", ".logcat"}, Exclusions: []string{"backup", "important"}, Enabled: true},
	{ID: "temp_files", Label: "临时文件", Risk: "low", TargetType: "file",
		NameEnds: []string{".tmp", ".temp", ".part", ".crdownload", ".download", ".bak", ".old"}, Enabled: true},
	{ID: "thumb_cache", Label: "缩略图缓存", Risk: "low", TargetType: "dir",
		PathContains: []string{"/.thumbnails", "/.face"}, Enabled: true},
	{ID: "tencent_log", Label: "腾讯日志", Risk: "low",
		PathContains: []string{"/tencent/msflogs", "/tencent/qalsdklogs", "/tencent/imsdklogs",
			"/tencent/wns", "/tencent/tbs_live_log", "/tencent/beacon"}, Enabled: true},
	{ID: "mac_files", Label: "Mac 残留", Risk: "low", TargetType: "file",
		NameEnds: []string{".ds_store"}, NameStarts: []string{"._"}, Enabled: true},
	{ID: "windows_files", Label: "Windows 残留", Risk: "low", TargetType: "file",
		NameEnds: []string{"thumbs.db", "desktop.ini"}, Enabled: true},
	{ID: "lostdir", Label: "LOST.DIR", Risk: "low", TargetType: "dir",
		PathContains: []string{"/lost.dir"}, Enabled: true},
	{ID: "ad_files", Label: "广告缓存", Risk: "mid",
		PathContains: []string{"/.adcache", "/adcache", "/gdt_plugin", "/adnet"}, Enabled: true},
	{ID: "analytics", Label: "统计埋点", Risk: "mid",
		PathContains: []string{"/bugly", "/umeng", "/.mta/", "/mobclick"}, Enabled: true},
	{ID: "crash_dump", Label: "崩溃转储", Risk: "low", TargetType: "file",
		NameEnds: []string{".dmp", ".hprof"}, PathContains: []string{"crash", "tombstone"}, Enabled: true},
}

func loadRuleLib() []CleanRule {
	b, err := os.ReadFile(ruleLibPath())
	if err != nil {
		writeRuleLib(defaultRules)
		return defaultRules
	}
	var wrap struct{ Rules []CleanRule `json:"rules"` }
	if json.Unmarshal(b, &wrap) != nil || len(wrap.Rules) == 0 {
		return defaultRules
	}
	return wrap.Rules
}

func writeRuleLib(rules []CleanRule) error {
	os.MkdirAll(filepath.Dir(ruleLibPath()), 0755)
	b, err := json.MarshalIndent(map[string]any{"version": 1, "rules": rules}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ruleLibPath(), b, 0644)
}

func apiRuleLib(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string `json:"action"` // list | reset | scan | toggle
		ID     string `json:"id"`
		On     bool   `json:"on"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	switch req.Action {
	case "list":
		writeJSON(w, 200, map[string]any{
			"rules": loadRuleLib(), "path": ruleLibPath(),
		})

	case "reset":
		if err := writeRuleLib(defaultRules); err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"status": "ok", "count": len(defaultRules)})

	case "toggle":
		rules := loadRuleLib()
		for i := range rules {
			if rules[i].ID == req.ID {
				rules[i].Enabled = req.On
			}
		}
		writeRuleLib(rules)
		writeJSON(w, 200, map[string]string{"status": "ok"})

	case "scan":
		hits := scanByRules()
		writeJSON(w, 200, map[string]any{"items": hits, "count": len(hits)})

	default:
		writeJSON(w, 400, map[string]string{"error": "unknown action"})
	}
}

// scanByRules 按规则库扫描（匹配顺序按代价短路，regex 最后）
func scanByRules() []map[string]any {
	rules := loadRuleLib()
	var out []map[string]any

	walkSdcard(func(h fileHit) {
		if len(out) > 2000 {
			return
		}
		if isProtectedPath(h.path) {
			return
		}
		name := strings.ToLower(filepath.Base(h.path))
		low := strings.ToLower(h.path)

		for _, r := range rules {
			if !r.Enabled {
				continue
			}
			if !ruleMatch(r, h, name, low) {
				continue
			}
			out = append(out, map[string]any{
				"path": h.path, "size": h.size,
				"label": r.Label, "risk": r.Risk,
			})
			return
		}
	})
	return out
}

func ruleMatch(r CleanRule, h fileHit, name, lowPath string) bool {
	// 1. targetType（最便宜）
	if r.TargetType == "file" || r.TargetType == "dir" {
		st, err := os.Stat(h.path)
		if err != nil {
			return false
		}
		if r.TargetType == "file" && st.IsDir() {
			return false
		}
		if r.TargetType == "dir" && !st.IsDir() {
			return false
		}
	}
	// 2. size
	if r.MinSize > 0 && h.size < r.MinSize {
		return false
	}
	if r.MaxSize > 0 && h.size > r.MaxSize {
		return false
	}
	// 3. age
	if r.MinAgeDays > 0 {
		if time.Since(h.mod) < time.Duration(r.MinAgeDays)*24*time.Hour {
			return false
		}
	}
	// 4. pathContains（OR）
	if len(r.PathContains) > 0 {
		hit := false
		for _, p := range r.PathContains {
			if strings.Contains(lowPath, strings.ToLower(p)) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	// 5. nameStarts
	if len(r.NameStarts) > 0 {
		hit := false
		for _, p := range r.NameStarts {
			if strings.HasPrefix(name, strings.ToLower(p)) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	// 6. nameEnds
	if len(r.NameEnds) > 0 {
		hit := false
		for _, p := range r.NameEnds {
			if strings.HasSuffix(name, strings.ToLower(p)) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	// 7. exclusions
	for _, ex := range r.Exclusions {
		if strings.Contains(lowPath, strings.ToLower(ex)) {
			return false
		}
	}
	// 8. regex（最贵，最后）
	if len(r.Regex) > 0 {
		hit := false
		for _, rx := range r.Regex {
			if ok, err := regexp.MatchString(rx, h.path); err == nil && ok {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	// 无任何条件 → 不匹配（防全盘误删）
	if len(r.PathContains) == 0 && len(r.NameStarts) == 0 &&
		len(r.NameEnds) == 0 && len(r.Regex) == 0 {
		return false
	}
	return true
}
