// JunkClean v4.1.0 — Phase 2/3 功能扩展：大文件/空文件/缩略图/应用级/重复/整理/回收站/定时/监控/统计
package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	sdcard    = "/storage/emulated/0"
	trashDir  = "/sdcard/.junkclean_trash"
	statsFile = stateDir + "/stats.json"
	ruleDir   = stateDir + "/rules"
)

var sdcardRoots = []string{sdcard}
var screenLock sync.Mutex

func registerFeatures(mux *http.ServeMux) {
	mux.HandleFunc("/api/big", corsMiddleware(apiBig))
	mux.HandleFunc("/api/empty", corsMiddleware(apiEmpty))
	mux.HandleFunc("/api/thumb", corsMiddleware(apiThumb))
	mux.HandleFunc("/api/cleanapp", corsMiddleware(apiCleanApp))
	mux.HandleFunc("/api/duplicate", corsMiddleware(apiDuplicate))
	mux.HandleFunc("/api/classify", corsMiddleware(apiClassify))
	mux.HandleFunc("/api/trash", corsMiddleware(apiTrash))
	mux.HandleFunc("/api/schedule", corsMiddleware(apiSchedule))
	mux.HandleFunc("/api/monitor", corsMiddleware(apiMonitor))
	mux.HandleFunc("/api/fstrim", corsMiddleware(apiFstrim))
	mux.HandleFunc("/api/stats", corsMiddleware(apiStats))
	mux.HandleFunc("/api/rules", corsMiddleware(apiRules))
	mux.HandleFunc("/api/delete", corsMiddleware(apiDelete))
}

// ---------- 通用：遍历 sdcard 收集文件 ----------

type fileHit struct {
	path string
	size int64
}

// walkSdcard 遍历 sdcard 下文件，跳过系统/应用私有目录
func walkSdcard(fn func(h fileHit)) {
	skipDirs := map[string]bool{
		"Android/data": true, ".junkclean_trash": true,
		"Android/obb": true, "Android": true,
	}
	var walk func(string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				if skipDirs[e.Name()] || strings.HasPrefix(e.Name(), ".") {
					continue
				}
				walk(filepath.Join(dir, e.Name()))
				continue
			}
			info, err := e.Info()
			if err != nil || e.Type()&fs.ModeSymlink != 0 {
				continue
			}
			fn(fileHit{path: filepath.Join(dir, e.Name()), size: info.Size()})
		}
	}
	for _, root := range sdcardRoots {
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			walk(root)
		}
	}
}

// ---------- 大文件 ----------

func scanBig(minSize int64, ext string) []JunkItem {
	var items []JunkItem
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	walkSdcard(func(h fileHit) {
		if h.size < minSize {
			return
		}
		if ext != "" && !strings.HasSuffix(strings.ToLower(h.path), "."+ext) {
			return
		}
		items = append(items, JunkItem{ID: "big:" + h.path, Path: h.path,
			Name: filepath.Base(h.path), Size: h.size, Count: 1})
	})
	sort.Slice(items, func(a, b int) bool { return items[a].Size > items[b].Size })
	if len(items) > 50 {
		items = items[:50]
	}
	return items
}

func apiBig(w http.ResponseWriter, r *http.Request) {
	minSize := int64(50 << 20) // 默认 50MB
	ext := ""
	if v := r.URL.Query().Get("min"); v != "" {
		fmt.Sscanf(v, "%d", &minSize)
	}
	ext = r.URL.Query().Get("ext")
	writeJSON(w, 200, map[string]any{"items": scanBig(minSize, ext)})
}

// ---------- 空文件 ----------

func scanEmpty() []JunkItem {
	var items []JunkItem
	walkSdcard(func(h fileHit) {
		if h.size == 0 {
			items = append(items, JunkItem{ID: "empty:" + h.path, Path: h.path,
				Name: filepath.Base(h.path), Size: 0, Count: 1})
		}
	})
	if len(items) > 200 {
		items = items[:200]
	}
	return items
}

func apiEmpty(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": scanEmpty()})
}

// ---------- 缩略图 ----------

func scanThumb() []JunkItem {
	var items []JunkItem
	dirs := []string{
		"/sdcard/Pictures/.thumbnails", "/sdcard/DCIM/.thumbnails",
		"/sdcard/MIUI/thumbnails",
	}
	for _, d := range dirs {
		sz, n := dirSize(d)
		if sz > 0 {
			items = append(items, JunkItem{ID: "thumb:" + d, Path: d,
				Name: filepath.Base(filepath.Dir(d)) + "/.thumbnails", Size: sz, Count: n})
		}
	}
	sort.Slice(items, func(a, b int) bool { return items[a].Size > items[b].Size })
	return items
}

func apiThumb(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"items": scanThumb()})
}

// ---------- 应用级清理 ----------

func cleanApp(pkg string) (int64, []string) {
	var freed int64
	var errs []string
	for _, root := range pkgRoots {
		base := filepath.Join(root, pkg)
		for _, sub := range []string{"cache", "code_cache", "app_webview"} {
			p := filepath.Join(base, sub)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				sz, _ := dirSize(p)
				if err := os.RemoveAll(p); err == nil {
					freed += sz
				} else {
					errs = append(errs, p)
				}
				logLine(fmt.Sprintf("cleanapp %s %s (%s)", pkg, sub, fmtSize(sz)))
			}
		}
	}
	return freed, errs
}

func apiCleanApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Package string `json:"package"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Package == "" || strings.Contains(req.Package, "/") || strings.Contains(req.Package, "..") {
		writeJSON(w, 400, map[string]string{"error": "invalid package"})
		return
	}
	freed, errs := cleanApp(req.Package)
	writeJSON(w, 200, map[string]any{"freed": freed, "errors": errs})
}

// ---------- 重复文件（md5 检测） ----------

func fileMD5(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := md5.New()
	buf := make([]byte, 1<<20)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// findDuplicates 遍历 sdcard 找 md5 相同文件组（跳过 <1MB，加速）
func findDuplicates() []map[string]any {
	bySize := map[int64][]string{}
	walkSdcard(func(h fileHit) {
		if h.size >= 1<<20 { // 只查 ≥1MB
			bySize[h.size] = append(bySize[h.size], h.path)
		}
	})
	byMD5 := map[string][]string{}
	for _, paths := range bySize {
		if len(paths) < 2 {
			continue
		}
		for _, p := range paths {
			sum := fileMD5(p)
			if sum != "" {
				byMD5[sum] = append(byMD5[sum], p)
			}
		}
	}
	var groups []map[string]any
	for _, paths := range byMD5 {
		if len(paths) < 2 {
			continue
		}
		sort.Strings(paths)
		groups = append(groups, map[string]any{
			"files": paths, "size": func() int64 {
				info, _ := os.Stat(paths[0])
				if info != nil {
					return info.Size()
				}
				return 0
			}(),
		})
	}
	sort.Slice(groups, func(a, b int) bool {
		sa, _ := groups[a]["size"].(int64)
		sb, _ := groups[b]["size"].(int64)
		return sa > sb
	})
	if len(groups) > 20 {
		groups = groups[:20]
	}
	return groups
}

func apiDuplicate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string   `json:"action"` // preview | delete | archive
		Keep   []string `json:"keep"`   // 保留的文件（其余删除）
	}
	json.NewDecoder(r.Body).Decode(&req)
	switch req.Action {
	case "preview":
		writeJSON(w, 200, map[string]any{"groups": findDuplicates()})
	case "delete":
		groups := findDuplicates()
		var freed int64
		var deleted int
		keep := map[string]bool{}
		for _, k := range req.Keep {
			keep[k] = true
		}
		for _, g := range groups {
			files, _ := g["files"].([]string)
			sz, _ := g["size"].(int64)
			for _, f := range files {
				if keep[f] {
					continue
				}
				if err := os.Remove(f); err == nil {
					freed += sz
					deleted++
				}
			}
		}
		logLine(fmt.Sprintf("duplicate delete: %d files, %s", deleted, fmtSize(freed)))
		writeJSON(w, 200, map[string]any{"freed": freed, "deleted": deleted})
	case "archive":
		os.MkdirAll("/sdcard/下载/Duplicates", 0755)
		var moved int
		for _, g := range findDuplicates() {
			files, _ := g["files"].([]string)
			for i, f := range files {
				if i == 0 {
					continue // 保留第一份
				}
				dest := filepath.Join("/sdcard/下载/Duplicates", filepath.Base(f))
				if _, err := os.Stat(dest); err == nil {
					dest = fmt.Sprintf("%s.%d%s", dest[:len(dest)-4], time.Now().Unix()%1000, filepath.Ext(dest))
				}
				if err := os.Rename(f, dest); err == nil {
					moved++
				}
			}
		}
		writeJSON(w, 200, map[string]any{"moved": moved})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: preview|delete|archive"})
	}
}

// ---------- 整理中心 classify ----------

type ClassifyRule struct {
	Src     string            `json:"src"`
	Dest    string            `json:"dest"`
	Exclude string            `json:"exclude"`
	Map     map[string]string `json:"map"` // 文件名模式 → 目标子目录
}

var defaultClassifyRules = []ClassifyRule{
	{Src: sdcardRoots[0]+"/Download", Dest: sdcardRoots[0]+"/下载", Map: map[string]string{
		"*.jpg": "图片", "*.jpeg": "图片", "*.png": "图片", "*.gif": "图片",
		"*.mp4": "视频", "*.mkv": "视频", "*.mov": "视频",
		"*.mp3": "音乐", "*.flac": "音乐", "*.wav": "音乐",
		"*.zip": "压缩包", "*.rar": "压缩包", "*.7z": "压缩包",
		"*.apk": "安装包", "*.xapk": "安装包",
		"*.pdf": "文档", "*.doc*": "文档", "*.txt": "文档",
	}},
}

// loadRules / saveRules 用 JSON 存于 stateDir/classify.json
func loadClassifyRules() []ClassifyRule {
	data, err := os.ReadFile(stateDir + "/classify.json")
	if err != nil {
		return defaultClassifyRules
	}
	var rules []ClassifyRule
	if json.Unmarshal(data, &rules) != nil || len(rules) == 0 {
		return defaultClassifyRules
	}
	return rules
}

func saveClassifyRules(rules []ClassifyRule) {
	os.WriteFile(stateDir+"/classify.json", mustJSON(rules), 0600)
}

// doClassify 执行整理：匹配 @src 下文件 → 按 @map 移到 @dest/<子目录>
func doClassify(rules []ClassifyRule, dryRun bool) ([]string, int64) {
	var moves []string
	var total int64
	for _, rule := range rules {
		if rule.Src == "" || rule.Dest == "" {
			continue
		}
		var exts []string
		var dirs []string
		for pat, sub := range rule.Map {
			_ = sub
			if strings.Contains(pat, "*") {
				exts = append(exts, strings.ToLower(strings.TrimPrefix(pat, "*")))
			}
		}
		_ = dirs
		entries, err := os.ReadDir(rule.Src)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if rule.Exclude != "" && matchGlob(rule.Exclude, name) {
				continue
			}
			// 找匹配的 map 条目
			for pat, sub := range rule.Map {
				if matchGlob(pat, name) {
					dest := filepath.Join(rule.Dest, sub)
					os.MkdirAll(dest, 0755)
					srcPath := filepath.Join(rule.Src, name)
					dstPath := filepath.Join(dest, name)
					if dryRun {
						moves = append(moves, srcPath+" → "+dstPath)
					} else if err := os.Rename(srcPath, dstPath); err == nil {
						info, _ := os.Stat(dstPath)
						if info != nil {
							total += info.Size()
						}
						moves = append(moves, srcPath+" → "+dstPath)
					}
					break
				}
			}
		}
	}
	return moves, total
}

// matchGlob 简易 glob：支持 * 通配（* 在任意位置）
func matchGlob(pattern, name string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == name
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(name, parts[0]) && strings.HasSuffix(name, parts[1])
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}

func apiClassify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string         `json:"action"` // preview | run | save | get
		Rules  []ClassifyRule `json:"rules"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	switch req.Action {
	case "get":
		writeJSON(w, 200, map[string]any{"rules": loadClassifyRules()})
	case "save":
		saveClassifyRules(req.Rules)
		writeJSON(w, 200, map[string]string{"status": "ok"})
	case "preview", "run":
		rules := req.Rules
		if len(rules) == 0 {
			rules = loadClassifyRules()
		}
		moves, total := doClassify(rules, req.Action == "preview")
		if req.Action == "run" {
			logLine(fmt.Sprintf("classify: %d files, %s", len(moves), fmtSize(total)))
		}
		writeJSON(w, 200, map[string]any{"moves": moves, "count": len(moves), "total": total})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: get|save|preview|run"})
	}
}

// ---------- 回收站 ----------

type TrashItem struct {
	Path string `json:"path"`
	Orig string `json:"orig"`
	Size int64  `json:"size"`
	Time int64  `json:"time"`
}

func trashMeta() string { return trashDir + "/.meta.json" }

func loadTrashMeta() []TrashItem {
	data, _ := os.ReadFile(trashMeta())
	var items []TrashItem
	json.Unmarshal(data, &items)
	return items
}

func saveTrashMeta(items []TrashItem) {
	os.WriteFile(trashMeta(), mustJSON(items), 0600)
}

// moveToTrash 移入回收站（记录原始路径）
func moveToTrash(paths []string) (int64, []string) {
	os.MkdirAll(trashDir, 0755)
	items := loadTrashMeta()
	var freed int64
	var errs []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			errs = append(errs, p)
			continue
		}
		dest := filepath.Join(trashDir, fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(p)))
		if err := os.Rename(p, dest); err != nil {
			errs = append(errs, p)
			continue
		}
		items = append(items, TrashItem{Path: dest, Orig: p, Size: info.Size(), Time: time.Now().Unix()})
		freed += info.Size()
	}
	saveTrashMeta(items)
	return freed, errs
}

func apiTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string   `json:"action"` // list | restore | empty | delete
		Paths  []string `json:"paths"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	switch req.Action {
	case "list":
		writeJSON(w, 200, map[string]any{"items": loadTrashMeta()})
	case "restore":
		items := loadTrashMeta()
		var rest int
		var kept []TrashItem
		del := map[string]bool{}
		for _, p := range req.Paths {
			del[p] = true
		}
		for _, it := range items {
			if del[it.Path] {
				os.MkdirAll(filepath.Dir(it.Orig), 0755)
				if os.Rename(it.Path, it.Orig) == nil {
					rest++
					continue
				}
			}
			kept = append(kept, it)
		}
		saveTrashMeta(kept)
		writeJSON(w, 200, map[string]any{"restored": rest})
	case "delete":
		items := loadTrashMeta()
		var del int
		var kept []TrashItem
		delSet := map[string]bool{}
		for _, p := range req.Paths {
			delSet[p] = true
		}
		for _, it := range items {
			if delSet[it.Path] {
				if os.RemoveAll(it.Path) == nil {
					del++
				}
				continue
			}
			kept = append(kept, it)
		}
		saveTrashMeta(kept)
		writeJSON(w, 200, map[string]any{"deleted": del})
	case "empty":
		os.RemoveAll(trashDir)
		os.MkdirAll(trashDir, 0755)
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: list|restore|delete|empty"})
	}
}

// ---------- 通用删除（仅限 sdcard 内路径，可选回收站） ----------

func apiDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Paths   []string `json:"paths"`
		ToTrash bool     `json:"toTrash"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	root := sdcardRoots[0]
	var freed int64
	var errs []string
	var toTrash []string
	for _, p := range req.Paths {
		if !strings.HasPrefix(p, root+"/") && p != root {
			errs = append(errs, "拒绝: "+p+"（不在 sdcard 内）")
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			errs = append(errs, p)
			continue
		}
		if req.ToTrash {
			toTrash = append(toTrash, p)
		} else if err := os.RemoveAll(p); err == nil {
			freed += info.Size()
		} else {
			errs = append(errs, p)
		}
	}
	if len(toTrash) > 0 {
		f, _ := moveToTrash(toTrash)
		freed += f
	}
	logLine(fmt.Sprintf("delete: %d paths, %s", len(req.Paths), fmtSize(freed)))
	writeJSON(w, 200, map[string]any{"freed": freed, "errors": errs})
}

// ---------- 定时任务 ----------

type Task struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Every int    `json:"every"` // 小时；0=每日
	Hour  int    `json:"hour"`  // 每日触发时间
}

var taskLock sync.Mutex
var taskTicker *time.Ticker

func loadTasks() []Task {
	data, _ := os.ReadFile(stateDir + "/tasks.json")
	var tasks []Task
	json.Unmarshal(data, &tasks)
	return tasks
}

func saveTasks(tasks []Task) {
	os.WriteFile(stateDir+"/tasks.json", mustJSON(tasks), 0600)
}

// startScheduler 启动定时扫描（daemon 启动时调用）
func startScheduler() {
	taskTicker = time.NewTicker(time.Minute)
	go func() {
		for range taskTicker.C {
			now := time.Now()
			for _, t := range loadTasks() {
				if t.Every > 0 {
					if now.Hour() == 0 && now.Minute() < 5 { // 每小时粒度粗略触发
						go autoRun()
					}
				} else if t.Hour == now.Hour() && now.Minute() < 5 {
					go autoRun()
				}
			}
		}
	}()
}

// autoRun 定时自动清理：扫描 + 清理安全分类
func autoRun() {
	if scanSt.running {
		return
	}
	scanSt.run()
	scanSt.mu.Lock()
	var ids []string
	for _, c := range scanSt.categories {
		if c.Careful {
			continue
		}
		for _, it := range c.Items {
			ids = append(ids, it.ID)
		}
	}
	scanSt.mu.Unlock()
	if len(ids) > 0 {
		scanSt.mu.Lock()
		scanSt.clean(ids)
		scanSt.done = false
		scanSt.mu.Unlock()
		recordStat(len(ids))
	}
}

func apiSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string `json:"action"` // list | add | remove
		Task   Task   `json:"task"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	tasks := loadTasks()
	switch req.Action {
	case "list":
		writeJSON(w, 200, map[string]any{"tasks": tasks})
	case "add":
		if req.Task.Name == "" {
			req.Task.Name = fmt.Sprintf("任务%d", len(tasks)+1)
		}
		req.Task.ID = fmt.Sprintf("t%d", time.Now().Unix())
		tasks = append(tasks, req.Task)
		saveTasks(tasks)
		writeJSON(w, 200, map[string]string{"status": "ok", "id": req.Task.ID})
	case "remove":
		var kept []Task
		for _, t := range tasks {
			if t.ID != req.Task.ID {
				kept = append(kept, t)
			}
		}
		saveTasks(kept)
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: list|add|remove"})
	}
}

// ---------- 监控（轮询自动整理；ponytail: 轮询替代 inotify，效率敏感时换 fsnotify） ----------

var monitorOn bool
var monitorStop chan bool

func monitorStart() bool {
	if monitorOn {
		return false
	}
	monitorOn = true
	monitorStop = make(chan bool)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				rules := loadClassifyRules()
				if len(rules) > 0 {
					moves, _ := doClassify(rules, false)
					if len(moves) > 0 {
						logLine(fmt.Sprintf("monitor: auto-classify %d files", len(moves)))
					}
				}
			case <-monitorStop:
				return
			}
		}
	}()
	return true
}

func apiMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	switch req.Action {
	case "start":
		monitorStart()
		writeJSON(w, 200, map[string]any{"status": "on"})
	case "stop":
		if monitorOn {
			close(monitorStop)
			monitorOn = false
		}
		writeJSON(w, 200, map[string]any{"status": "off"})
	case "status":
		writeJSON(w, 200, map[string]any{"status": map[bool]string{true: "on", false: "off"}[monitorOn]})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: start|stop|status"})
	}
}

// ---------- fstrim ----------

func apiFstrim(w http.ResponseWriter, r *http.Request) {
	out, err := exec.Command("fstrim", "/data").CombinedOutput()
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error(), "output": string(out)})
		return
	}
	logLine("fstrim /data done")
	writeJSON(w, 200, map[string]any{"status": "ok", "output": string(out)})
}

// ---------- 统计 ----------

func recordStat(items int) {
	data, _ := os.ReadFile(statsFile)
	stats := map[string]int{}
	json.Unmarshal(data, &stats)
	key := time.Now().Format("2006-01-02")
	stats[key] += items
	os.WriteFile(statsFile, mustJSON(stats), 0600)
}

func apiStats(w http.ResponseWriter, r *http.Request) {
	data, _ := os.ReadFile(statsFile)
	stats := map[string]int{}
	json.Unmarshal(data, &stats)
	// 最近 7 天
	keys := []string{}
	for i := 6; i >= 0; i-- {
		keys = append(keys, time.Now().AddDate(0, 0, -i).Format("2006-01-02"))
	}
	days := []map[string]any{}
	for _, k := range keys {
		days = append(days, map[string]any{"date": k[5:], "items": stats[k]})
	}
	writeJSON(w, 200, map[string]any{"days": days})
}

// ---------- 规则管理（分类规则增删，规则文件） ----------

func apiRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		Action string `json:"action"` // list | reset
	}
	json.NewDecoder(r.Body).Decode(&req)
	switch req.Action {
	case "list":
		writeJSON(w, 200, map[string]any{"rules": loadClassifyRules()})
	case "reset":
		saveClassifyRules(defaultClassifyRules)
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: list|reset"})
	}
}
