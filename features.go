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
	mux.HandleFunc("/api/apps", corsMiddleware(apiApps))
	mux.HandleFunc("/api/rollback", corsMiddleware(apiRollback))
	mux.HandleFunc("/api/delete", corsMiddleware(apiDelete))
}

// ---------- 通用：遍历 sdcard 收集文件 ----------

type fileHit struct {
	path string
	size int64
	mod  time.Time
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
			fn(fileHit{path: filepath.Join(dir, e.Name()), size: info.Size(), mod: info.ModTime()})
		}
	}
	roots := append([]string{}, sdcardRoots...)
	if cfg.ScanRoot != "" && cfg.ScanRoot != sdcardRoots[0] {
		roots = append(roots, cfg.ScanRoot)
	}
	for _, root := range roots {
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			walk(root)
		}
	}
}

// ---------- 大文件 ----------

func scanBig(minSize int64, ext string, minDays int) []JunkItem {
	var items []JunkItem
	var exts []string
	for _, e := range strings.Split(ext, ",") {
		e = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(e, ".")))
		if e != "" {
			exts = append(exts, e)
		}
	}
	walkSdcard(func(h fileHit) {
		if h.size < minSize {
			return
		}
		if len(exts) > 0 {
			lower := strings.ToLower(h.path)
			match := false
			for _, e := range exts {
				if strings.HasSuffix(lower, "."+e) {
					match = true
					break
				}
			}
			if !match {
				return
			}
		}
		if minDays > 0 && time.Since(h.mod) < time.Duration(minDays)*24*time.Hour {
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
	minSize := int64(50 << 20)
	ext := ""
	days := 0
	if v := r.URL.Query().Get("min"); v != "" {
		fmt.Sscanf(v, "%d", &minSize)
	}
	ext = r.URL.Query().Get("ext")
	fmt.Sscanf(r.URL.Query().Get("days"), "%d", &days)
	writeJSON(w, 200, map[string]any{"items": scanBig(minSize, ext, days)})
}

// ---------- 空文件 ----------

func scanEmpty() []JunkItem {
	var items []JunkItem
	// 空目录（跳过隐藏/系统目录）
	root := sdcardRoots[0]
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || p == root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		entries, _ := os.ReadDir(p)
		if len(entries) == 0 {
			items = append(items, JunkItem{ID: "emptydir:" + p, Path: p,
				Name: filepath.Base(p), Size: 0, Count: 0})
		}
		return nil
	})
	// 空文件（0 字节）
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
	Src       string            `json:"src"`
	Dest      string            `json:"dest"`
	Exclude   string            `json:"exclude"`
	Map       map[string]string `json:"map"`
	Recurse   bool              `json:"recurse"`
	Integrity bool              `json:"integrity"`
	MinSize   int64             `json:"minSize"`
	MinDays   int               `json:"minDays"`
	Rename    string            `json:"rename"`
}

var incompleteSuffix = []string{".part", ".crdownload", ".tmp", ".partial", ".downloading", ".!q", ".aria2"}

func isIncomplete(name string) bool {
	lower := strings.ToLower(name)
	for _, s := range incompleteSuffix {
		if strings.HasSuffix(lower, s) {
			return true
		}
	}
	return false
}

func uniqueDest(dir, name, strategy string) string {
	if strategy == "add" {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		ext := filepath.Ext(name)
		for i := 1; ; i++ {
			cand := filepath.Join(dir, fmt.Sprintf("%s.%d%s", base, i, ext))
			if _, err := os.Stat(cand); err != nil {
				return cand
			}
		}
	} else if strategy == "skip" {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return ""
		}
	}
	return filepath.Join(dir, name)
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
		if (!strings.HasPrefix(p, root+"/") && p != root) || strings.Contains(p, "..") {
			errs = append(errs, "拒绝: "+p+"（路径不合法）")
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
	ID         string `json:"id"`
	Name       string `json:"name"`
	Every      int    `json:"every"`
	Hour       int    `json:"hour"`
	OnlyCharge bool   `json:"onlyCharge"`
	OnlyWifi   bool   `json:"onlyWifi"`
	LastRun    string `json:"lastRun"`
	LastResult string `json:"lastResult"`
	LastUnix   int64  `json:"lastUnix"`
	Action     string `json:"action"`
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
func runAction(action string) {
	autoRun()
	if action == "both" || action == "classify" {
		classifyAuto()
	}
}

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
		for _, t := range tasks {
			if t.Name == req.Task.Name {
				writeJSON(w, 400, map[string]string{"error": "同名任务已存在"})
				return
			}
		}
		if req.Task.Action == "" {
			req.Task.Action = "both"
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

func classifyAuto() {
	rules := loadClassifyRules()
	if len(rules) == 0 {
		return
	}
	moves, _ := doClassify(rules, false)
	if len(moves) > 0 {
		logLine(fmt.Sprintf("classifyAuto: %d files", len(moves)))
	}
}



// ---------- 监控（轮询自动整理；ponytail: 轮询替代 inotify，效率敏感时换 fsnotify） ----------

var monitorOn bool
var monitorStop chan bool

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
	case "pause":
		monitorPaused = true
		writeJSON(w, 200, map[string]any{"status": "paused"})
	case "resume":
		monitorPaused = false
		writeJSON(w, 200, map[string]any{"status": "on"})
	case "status":
		st := "off"
		if monitorOn {
			st = "on"
		}
		if monitorPaused {
			st = "paused"
		}
		writeJSON(w, 200, map[string]any{"status": st, "count": monitorCount, "last": monitorLast})
	default:
		writeJSON(w, 400, map[string]string{"error": "action: start|stop|status"})
	}
}

// ---------- 统计 ----------

func recordStat(items int) {
	s := loadStats()
	s.Daily[time.Now().Format("2006-01-02")] += items
	s.Total["items"] += int64(items)
	os.WriteFile(statsFile, mustJSON(s), 0600)
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

// doClassify 增强版：递归/完整性/过滤/重名策略
func doClassify(rules []ClassifyRule, dryRun bool) ([]string, int64) {
	var moves []string
	var total int64
	for _, rule := range rules {
		if rule.Src == "" || rule.Dest == "" {
			continue
		}
		var process func(dir string)
		process = func(dir string) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return
			}
			for _, e := range entries {
				path := filepath.Join(dir, e.Name())
				if e.IsDir() {
					if rule.Recurse && !strings.HasPrefix(e.Name(), ".") {
						process(path)
					}
					continue
				}
				classifyFile(path, rule, &moves, &total, dryRun)
			}
		}
		process(rule.Src)
	}
	return moves, total
}

// classifyFile 单文件分类：排除/完整性/大小/时间/重名
func classifyFile(srcPath string, rule ClassifyRule, moves *[]string, total *int64, dryRun bool) {
	name := filepath.Base(srcPath)
	if rule.Exclude != "" && matchGlob(rule.Exclude, name) {
		return
	}
	if rule.Integrity && isIncomplete(name) {
		return
	}
	info, err := os.Stat(srcPath)
	if err != nil || info.IsDir() {
		return
	}
	if rule.MinSize > 0 && info.Size() < rule.MinSize {
		return
	}
	if rule.MinDays > 0 && time.Since(info.ModTime()) < time.Duration(rule.MinDays)*24*time.Hour {
		return
	}
	for pat, sub := range rule.Map {
		if !matchGlob(pat, name) {
			continue
		}
		dest := filepath.Join(rule.Dest, sub)
		os.MkdirAll(dest, 0755)
		dstPath := uniqueDest(dest, name, rule.Rename)
		if dstPath == "" {
			return // skip 且目标已存在
		}
		if dryRun {
			*moves = append(*moves, srcPath+" → "+dstPath)
		} else if err := os.Rename(srcPath, dstPath); err == nil {
			*total += info.Size()
			*moves = append(*moves, srcPath+" → "+dstPath)
		}
		return
	}
}

// ---------- 应用列表（各应用缓存大小） ----------

func scanApps() []map[string]any {
	seen := map[string]bool{}
	var apps []map[string]any
	for _, root := range pkgRoots {
		entries, _ := os.ReadDir(root)
		for _, e := range entries {
			if !e.IsDir() || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			var total int64
			for _, sub := range []string{"cache", "code_cache", "app_webview"} {
				sz, _ := dirSize(filepath.Join(root, e.Name(), sub))
				total += sz
			}
			if total > 0 {
				apps = append(apps, map[string]any{
					"package": e.Name(), "size": total,
				})
			}
		}
	}
	sort.Slice(apps, func(a, b int) bool {
		sa, _ := apps[a]["size"].(int64)
		sb, _ := apps[b]["size"].(int64)
		return sa > sb
	})
	if len(apps) > 100 {
		apps = apps[:100]
	}
	return apps
}

func apiApps(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"apps": scanApps()})
}


// ---------- fstrim 增强（多分区 + 记录上次结果） ----------

func apiFstrim(w http.ResponseWriter, r *http.Request) {
	var parts []string
	if v := r.URL.Query().Get("parts"); v != "" {
		parts = strings.Split(v, ",")
	} else {
		parts = []string{"/data"}
	}
	out := []string{}
	for _, p := range parts {
		o, err := exec.Command("fstrim", p).CombinedOutput()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error(), "output": string(o)})
			return
		}
		out = append(out, string(o))
	}
	cfgLock.Lock()
	cfg.LastTrim = time.Now().Format("2006-01-02 15:04")
	cfgLock.Unlock()
	saveConfig()
	logLine("fstrim " + strings.Join(parts, ",") + " done")
	writeJSON(w, 200, map[string]any{"status": "ok", "output": strings.Join(out, "\n")})
}

// ---------- 统计增强（累计 + 分类） ----------

type StatData struct {
	Daily map[string]int   `json:"daily"`
	Total map[string]int64 `json:"total"`
	Cats  map[string]int   `json:"cats"`
}

func loadStats() StatData {
	var s StatData
	data, _ := os.ReadFile(statsFile)
	json.Unmarshal(data, &s)
	if s.Daily == nil {
		s.Daily = map[string]int{}
	}
	if s.Total == nil {
		s.Total = map[string]int64{}
	}
	if s.Cats == nil {
		s.Cats = map[string]int{}
	}
	return s
}

func apiStats(w http.ResponseWriter, r *http.Request) {
	s := loadStats()
	keys := []string{}
	for i := 6; i >= 0; i-- {
		keys = append(keys, time.Now().AddDate(0, 0, -i).Format("2006-01-02"))
	}
	days := []map[string]any{}
	for _, k := range keys {
		days = append(days, map[string]any{"date": k[5:], "items": s.Daily[k]})
	}
	var totalItems int
	var totalSize int64
	for _, v := range s.Total {
		totalSize += v
	}
	for _, v := range s.Daily {
		totalItems += v
	}
	writeJSON(w, 200, map[string]any{
		"days": days, "totalItems": totalItems, "totalSize": totalSize, "cats": s.Cats,
	})
}

func isCharging() bool {
	data, err := os.ReadFile("/sys/class/power_supply/battery/status")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "Charging") || strings.Contains(string(data), "Full")
}

func isWifi() bool {
	data, err := os.ReadFile("/sys/class/net/wlan0/operstate")
	return err == nil && strings.TrimSpace(string(data)) == "up"
}

func startScheduler() {
	taskTicker = time.NewTicker(time.Minute)
	go func() {
		for range taskTicker.C {
			now := time.Now()
			// 回收站自动清理
			if cfg.AutoTrashDays > 0 && now.Hour() == 0 && now.Minute() < 5 {
				go autoEmptyTrash()
			}
			for _, t := range loadTasks() {
				triggered := false
				if t.Every > 0 {
					if t.LastUnix == 0 || now.Unix()-t.LastUnix >= int64(t.Every)*3600 {
						triggered = true
					}
				} else if t.Hour == now.Hour() && now.Minute() < 5 {
					triggered = true
				}
				if !triggered {
					continue
				}
				if t.OnlyCharge && !isCharging() {
					continue
				}
				if t.OnlyWifi && !isWifi() {
					continue
				}
				go func(id, action string) {
					ts := time.Now().Format("2006-01-02 15:04")
					runAction(action)
					tasks := loadTasks()
					for i := range tasks {
						if tasks[i].ID == id {
							tasks[i].LastRun = ts
							tasks[i].LastResult = "ok"
							tasks[i].LastUnix = time.Now().Unix()
						}
					}
					saveTasks(tasks)
				}(t.ID, t.Action)
			}
		}
	}()
}

func autoEmptyTrash() {
	items := loadTrashMeta()
	var kept []TrashItem
	for _, it := range items {
		if time.Now().Unix()-it.Time > int64(cfg.AutoTrashDays)*86400 {
			os.RemoveAll(it.Path)
			continue
		}
		kept = append(kept, it)
	}
	saveTrashMeta(kept)
}

// ---------- 热更新回滚 ----------

func apiRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	bak := "/data/adb/junkclean/junkclean.bak"
	data, err := os.ReadFile(bak)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "无备份文件"})
		return
	}
	modDir := "/data/adb/modules/junkclean"
	os.MkdirAll(modDir+"/system/bin", 0755)
	os.WriteFile(modDir+"/system/bin/junkclean", data, 0755)
	shPath := "/system/bin/sh"
	if _, err := os.Stat(shPath); err != nil {
		shPath = "/bin/sh"
	}
	cmd := exec.Command(shPath, "-c",
		"setsid "+modDir+"/system/bin/junkclean daemon >> /data/adb/junkclean/daemon.log 2>&1 &")
	if err := cmd.Start(); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	logLine("rollback: 已恢复备份并重启")
	writeJSON(w, 200, map[string]string{"status": "ok", "msg": "已回滚，daemon 重启中"})
	go func() {
		time.Sleep(600 * time.Millisecond)
		os.Exit(0)
	}()
}

// ---------- 监控状态 ----------

var monitorCount int64
var monitorLast string
var monitorPaused bool

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
				if monitorPaused {
					continue
				}
				rules := loadClassifyRules()
				if len(rules) > 0 {
					moves, _ := doClassify(rules, false)
					if len(moves) > 0 {
						monitorCount += int64(len(moves))
						monitorLast = time.Now().Format("2006-01-02 15:04")
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
