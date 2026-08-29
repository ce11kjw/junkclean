// JunkClean v3.0 - Android garbage cleaner (Go single binary, embedded WebUI)
package main

import (
	"archive/zip"
	"os/signal"
	"context"
	"crypto/sha256"
	"bytes"
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed webroot
var webFS embed.FS

const (
	ver      = "4.3.5"
	verCode  = 435
	port     = "46780"
	stateDir = "/data/adb/junkclean"
	logFile  = stateDir + "/junkclean.log"
	stateF   = stateDir + "/state.json"
	confF    = stateDir + "/config.json"
)

// ----- models -----

type JunkFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type JunkItem struct {
	ID        string     `json:"id"`
	Path      string     `json:"path"`
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	Count     int        `json:"count"`
	Files     []JunkFile `json:"files,omitempty"`
	Recurse   bool       `json:"recurse"`   // 子目录递归（默认关）
	Integrity bool       `json:"integrity"` // 完整性检测（默认开）
}

type Category struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Icon    string     `json:"icon"`
	Desc    string     `json:"desc"`
	Careful bool       `json:"careful"`
	Items   []JunkItem `json:"items"`
	Total   int64      `json:"total"`
}

type Config struct {
	Whitelist     []string        `json:"whitelist"`
	Cats          map[string]bool `json:"cats"`
	AIEndpoint    string          `json:"aiEndpoint"`
	AIKey         string          `json:"aiKey"`
	AIModel       string          `json:"aiModel"`
	AutoClean     bool            `json:"autoClean"`
	AutoTrashDays int             `json:"autoTrashDays"`
	LastTrim      string          `json:"lastTrim"`
	BGUrl         string          `json:"bgUrl"`
	ScanRoot      string          `json:"scanRoot"`
	Theme         string          `json:"theme"`
	Accent        string          `json:"accent"`

	// v4.3.0 对齐 APP：定时目录清理 / 保护路径 / 规则库
	CleanDirs   []string `json:"cleanDirs"`   // 每项 "路径|1(删整个)/0(只清内容)"
	Protected   []string `json:"protected"`   // 保护路径（支持通配符 *）
	ScheduleOn  bool     `json:"scheduleOn"`  // 定时清理开关
	ScheduleMin int      `json:"scheduleMin"` // 间隔（分钟）
}

type ScanState struct {
	mu         sync.Mutex
	running    bool
	stage      string
	curPath    string
	found      int64
	bytes      int64
	categories []Category
	total      int64
	done       bool
	lastScan   time.Time
}

var (
	scanSt     = &ScanState{}
	cfg        = Config{}
	cfgLock    sync.RWMutex
	whitelistM = map[string]bool{}
)

// ----- scan rules -----

type scanRule struct {
	id      string
	name    string
	icon    string
	desc    string
	careful bool
	dirs    []string
	subDirs []string
}

var rules = []scanRule{
	{"cache", "应用缓存", "📦", "各应用 cache / code_cache，清理后自动重建", false, nil,
		[]string{"cache", "code_cache"}},
	{"webview", "WebView 缓存", "🌐", "WebView 渲染缓存，可安全清理", false, nil,
		[]string{"app_webview"}},
	{"logs", "系统日志", "📋", "崩溃、ANR、dropbox 等诊断日志", false,
		[]string{"/data/tombstones", "/data/anr", "/data/system/dropbox",
			"/data/log", "/data/logcat", "/data/system/package_cache"}, nil},
	{"temp", "临时文件", "🗑️", "系统临时目录", false,
		[]string{"/data/local/tmp", "/cache", "/data/cache"}, nil},
	{"remnant", "应用残留", "🧩",
		"no_backup 与数据库日志文件（谨慎，可能影响部分应用状态）", true, nil,
		[]string{"no_backup"}},
}

var pkgRoots = []string{"/data/data", "/data/user_de/0"}
var forbidden = []string{"/system", "/vendor", "/product", "/apex", "/data/adb", "/data/app"}

func isForbidden(p string) bool {
	for _, f := range forbidden {
		if p == f || strings.HasPrefix(p, f+"/") {
			return true
		}
	}
	return false
}

// ----- helpers -----

func dirSize(root string) (int64, int) {
	var sz int64
	var n int
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		sz += info.Size()
		n++
		return nil
	})
	return sz, n
}

func diskInfo(p string) map[string]any {
	var st syscall.Statfs_t
	if err := syscall.Statfs(p, &st); err != nil {
		return nil
	}
	total := st.Blocks * uint64(st.Bsize)
	free := st.Bavail * uint64(st.Bsize)
	used := total - free
	pct := float64(used) / float64(total) * 100
	return map[string]any{"total": total, "used": used, "free": free, "percent": pct}
}

func fmtSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ----- scan engine -----

func (s *ScanState) setPath(p string) {
	s.mu.Lock()
	s.curPath = p
	s.mu.Unlock()
}

func (s *ScanState) addFound(n int, b int64) {
	s.mu.Lock()
	s.found += int64(n)
	s.bytes += b
	s.mu.Unlock()
}

func forEachPkg(fn func(pkg string)) {
	for _, root := range pkgRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || whitelistM[e.Name()] {
				continue
			}
			fn(filepath.Join(root, e.Name()))
		}
	}
}

func (s *ScanState) scanSubDir(cat *Category, ruleID, sub string) {
	forEachPkg(func(pkg string) {
		p := filepath.Join(pkg, sub)
		s.setPath(p)
		if isForbidden(p) {
			return
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			sz, n := dirSize(p)
			if sz > 0 {
				cat.Items = append(cat.Items, JunkItem{
					ID: ruleID + ":" + p, Path: p,
					Name: filepath.Base(pkg), Size: sz, Count: n,
				})
				s.addFound(n, sz)
			}
		}
	})
}

func (s *ScanState) scanJournals(cat *Category) {
	forEachPkg(func(pkg string) {
		dbDir := filepath.Join(pkg, "databases")
		s.setPath(dbDir)
		var files []JunkFile
		var sz int64
		filepath.WalkDir(dbDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if strings.HasSuffix(name, ".log") || strings.HasSuffix(name, "-journal") ||
				strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-shm") {
				info, _ := d.Info()
				if info != nil {
					sz += info.Size()
					files = append(files, JunkFile{Path: p, Size: info.Size()})
				}
			}
			return nil
		})
		if sz > 0 {
			cat.Items = append(cat.Items, JunkItem{
				ID: "remnant:" + dbDir, Path: dbDir,
				Name: filepath.Base(pkg) + " DB logs", Size: sz,
				Count: len(files), Files: files,
			})
			s.addFound(len(files), sz)
		}
	})
}



func (s *ScanState) run() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running, s.stage, s.done = true, "walking", false
	s.categories, s.total, s.found, s.bytes = nil, 0, 0, 0
	s.mu.Unlock()

	var cats []Category
	var total int64
	cfgLock.RLock()
	catsOn := cfg.Cats
	cfgLock.RUnlock()
	for _, r := range rules {
		if catsOn != nil && !catsOn[r.id] {
			continue // 分类开关关闭
		}
		cat := Category{ID: r.id, Name: r.name, Icon: r.icon,
			Desc: r.desc, Careful: r.careful}
		for _, d := range r.dirs {
			s.setPath(d)
			if isForbidden(d) {
				continue
			}
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				sz, n := dirSize(d)
				if sz > 0 {
					cat.Items = append(cat.Items, JunkItem{
						ID: r.id + ":" + d, Path: d, Name: d,
						Size: sz, Count: n,
					})
					s.addFound(n, sz)
				}
			}
		}
		for _, sub := range r.subDirs {
			s.scanSubDir(&cat, r.id, sub)
		}
		if r.id == "remnant" {
			s.scanJournals(&cat)
		}
		for i := range cat.Items {
			cat.Total += cat.Items[i].Size
		}
		total += cat.Total
		if len(cat.Items) > 0 {
			sort.Slice(cat.Items, func(a, b int) bool {
				return cat.Items[a].Size > cat.Items[b].Size
			})
			cats = append(cats, cat)
		}
	}

	s.mu.Lock()
	s.categories, s.total = cats, total
	s.stage, s.done, s.running = "done", true, false
	s.lastScan = time.Now()
	s.mu.Unlock()
	saveState()
	logLine("scan done: "+fmt.Sprintf("%d categories, %s", len(cats), fmtSize(total)))
}

// ----- clean -----

func (s *ScanState) clean(ids []string) (int64, []string) {
	var freed int64
	var errs []string
	idx := map[string]JunkItem{}
	for _, c := range s.categories {
		for _, it := range c.Items {
			idx[it.ID] = it
		}
	}
	for _, id := range ids {
		it, ok := idx[id]
		if !ok {
			errs = append(errs, "unknown: "+id)
			continue
		}
		if len(it.Files) > 0 {
			for _, f := range it.Files {
				if err := os.Remove(f.Path); err == nil {
					freed += f.Size
				} else {
					errs = append(errs, f.Path)
				}
			}
		} else if err := os.RemoveAll(it.Path); err == nil {
			freed += it.Size
		} else {
			errs = append(errs, it.Path+": "+err.Error())
		}
		logLine("clean " + it.Path + " (" + fmtSize(it.Size) + ")")
	}
	return freed, errs
}

// ----- state / config / log -----

func saveState() {
	data, _ := json.Marshal(scanSt.categories)
	os.WriteFile(stateF, data, 0600)
}

func loadState() {
	data, err := os.ReadFile(stateF)
	if err != nil {
		return
	}
	var cats []Category
	if json.Unmarshal(data, &cats) != nil {
		return
	}
	scanSt.mu.Lock()
	scanSt.categories, scanSt.done = cats, true
	scanSt.total = 0
	for _, c := range cats {
		scanSt.total += c.Total
	}
	scanSt.mu.Unlock()
}

func loadConfig() {
	data, err := os.ReadFile(confF)
	if err == nil {
		json.Unmarshal(data, &cfg)
	}
	// 默认分类全开
	if cfg.Cats == nil {
		cfg.Cats = map[string]bool{"cache": true, "webview": true, "logs": true, "temp": true, "remnant": true}
	}
	cfgLock.Lock()
	whitelistM = map[string]bool{}
	for _, p := range cfg.Whitelist {
		whitelistM[p] = true
	}
	cfgLock.Unlock()
}

// atomicWrite 先写临时文件再 rename，防写入中断损坏
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func saveConfig() {
	atomicWrite(confF, mustJSON(cfg), 0600)
}

func mustJSON(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}

const maxLogBytes = 512 * 1024

func logLine(s string) {
	if st, err := os.Stat(logFile); err == nil && st.Size() > maxLogBytes {
		if data, err := os.ReadFile(logFile); err == nil {
			// 保留后半部分，从下一个换行开始
			half := data[len(data)/2:]
			if i := bytes.IndexByte(half, '\n'); i >= 0 {
				half = half[i+1:]
			}
			atomicWrite(logFile, half, 0644)
		}
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", time.Now().Format("2006-01-02 15:04:05"), s)
}

func tailLog(n int) []string {
	f, err := os.Open(logFile)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// ----- HTTP handlers -----

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// 预检请求放行（WebView 跨域 fetch）
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin == "" ||
			strings.HasPrefix(origin, "http://127.0.0.1") ||
			strings.HasPrefix(origin, "http://localhost") ||
			strings.HasPrefix(origin, "file://") ||
			strings.HasPrefix(origin, "ksu://") ||
			strings.HasPrefix(origin, "https://mui.kernelsu")
		if !allowed {
			writeJSON(w, 403, map[string]string{"error": "跨域来源不被允许"})
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(204)
			return
		}
		next(w, r)
	}
}

func apiStatus(w http.ResponseWriter, r *http.Request) {
	scanSt.mu.Lock()
	resp := map[string]any{
		"version": ver,
		"root":    os.Geteuid() == 0,
		"disk":    diskInfo("/data"),
		"scan": map[string]any{
			"running": scanSt.running,
			"stage":   scanSt.stage,
			"curPath": scanSt.curPath,
			"found":   scanSt.found,
			"bytes":   scanSt.bytes,
			"done":    scanSt.done,
			"total":   scanSt.total,
		},
	}
	scanSt.mu.Unlock()
	writeJSON(w, 200, resp)
}

func apiScan(w http.ResponseWriter, r *http.Request) {
	scanSt.mu.Lock()
	running := scanSt.running
	cached := scanSt.done && time.Since(scanSt.lastScan) < 60*time.Second
	scanSt.mu.Unlock()
	if running {
		writeJSON(w, 409, map[string]string{"error": "scan already running"})
		return
	}
	if cached && r.URL.Query().Get("force") != "1" {
		writeJSON(w, 200, map[string]string{"status": "cached"}) // 60s 内用缓存
		return
	}
	go scanSt.run()
	writeJSON(w, 200, map[string]string{"status": "started"})
}

func apiResult(w http.ResponseWriter, r *http.Request) {
	scanSt.mu.Lock()
	defer scanSt.mu.Unlock()
	writeJSON(w, 200, map[string]any{
		"categories": scanSt.categories,
		"total":      scanSt.total,
		"done":       scanSt.done,
	})
}

func apiClean(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad body"})
		return
	}
	scanSt.mu.Lock()
	if !scanSt.done {
		scanSt.mu.Unlock()
		writeJSON(w, 409, map[string]string{"error": "scan not done"})
		return
	}
	before := diskInfo("/data")
	freed, errs := scanSt.clean(req.IDs)
	scanSt.done, scanSt.categories = false, nil
	scanSt.mu.Unlock()
	saveState()
	after := diskInfo("/data")
	recordStat(len(req.IDs))
	logLine(fmt.Sprintf("clean done: freed %s", fmtSize(freed)))
	writeJSON(w, 200, map[string]any{"freed": freed, "errors": errs, "before": before, "after": after})
}

func apiConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfgLock.RLock()
		c := cfg
		cfgLock.RUnlock()
		writeJSON(w, 200, c)
	case http.MethodPost:
		var nc Config
		if err := json.NewDecoder(r.Body).Decode(&nc); err != nil {
			writeJSON(w, 400, map[string]string{"error": "bad body"})
			return
		}
		cfgLock.Lock()
		cfg = nc
		whitelistM = map[string]bool{}
		for _, p := range nc.Whitelist {
			whitelistM[p] = true
		}
		cfgLock.Unlock()
		saveConfig()
		writeJSON(w, 200, map[string]string{"status": "ok"})
	default:
		writeJSON(w, 405, map[string]string{"error": "GET/POST"})
	}
}

func apiLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"logs": tailLog(300)})
}

// ----- 更新信息（自动获取最新 zipUrl，用户无需填） -----

func apiUpdateinfo(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://raw.githubusercontent.com/ce11kjw/junkclean/main/update.json")
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "网络错误: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	var up struct {
		Version     string `json:"version"`
		VersionCode int    `json:"versionCode"`
		ZipURL      string `json:"zipUrl"`
		SHA256      string `json:"sha256"`
		Changelog   string `json:"changelog"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&up); err != nil {
		writeJSON(w, 502, map[string]string{"error": "update.json 解析失败"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"current": ver, "currentCode": verCode,
		"remote": up.Version, "remoteCode": up.VersionCode,
		"zipUrl": up.ZipURL, "changelog": up.Changelog, "sha256": up.SHA256,
		"hasUpdate": up.VersionCode > verCode,
	})
}

// ----- 热更新：下载 zip → 覆盖模块文件 → 重启 daemon（不重启手机） -----

func unzipTo(zipPath, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		p := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(p, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(p), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(p)
		if err != nil {
			rc.Close()
			return err
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
		if f.Mode()&0111 != 0 {
			os.Chmod(p, 0755)
		}
	}
	return nil
}

func copyTree(src, dst string) {
	filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(src, p)
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		os.WriteFile(target, data, 0755)
		return nil
	})
}

func apiHotupdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	var req struct {
		URL    string `json:"url"`
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	modDir := "/data/adb/modules/junkclean"
	tmpZip := "/data/local/tmp/jc-hot.zip"
	downloaded := false
	tmpDir := "/data/local/tmp/jc-hot"

	if req.URL != "" {
		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Get(req.URL)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "下载失败: " + err.Error()})
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			writeJSON(w, 502, map[string]string{"error": fmt.Sprintf("下载失败: HTTP %d", resp.StatusCode)})
			return
		}
		f, err := os.Create(tmpZip)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		io.Copy(f, resp.Body)
		f.Close()
		downloaded = true
	} else if req.Path != "" {
		tmpZip = req.Path
	} else {
		writeJSON(w, 400, map[string]string{"error": "需要 url 或 path"})
		return
	}

	// 解压
	os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)
	if req.SHA256 != "" {
		data, err := os.ReadFile(tmpZip)
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": "读取 zip 失败"})
			return
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(data))
		if !strings.EqualFold(sum, req.SHA256) {
			if downloaded {
				os.Remove(tmpZip)
			}
			writeJSON(w, 400, map[string]string{"error": "SHA256 校验失败，已丢弃"})
			return
		}
		logLine("hotupdate: sha256 校验通过")
	}
	if err := unzipTo(tmpZip, tmpDir); err != nil {
		writeJSON(w, 400, map[string]string{"error": "解压失败: " + err.Error()})
		return
	}

	// 校验
	if _, err := os.Stat(tmpDir + "/module.prop"); err != nil {
		writeJSON(w, 400, map[string]string{"error": "不是有效模块（缺 module.prop）"})
		return
	}
	newBin := tmpDir + "/system/bin/junkclean"
	if _, err := os.Stat(newBin); err != nil {
		writeJSON(w, 400, map[string]string{"error": "zip 缺少二进制 system/bin/junkclean"})
		return
	}

	// 备份旧二进制
	if data, err := os.ReadFile(modDir + "/system/bin/junkclean"); err == nil {
		os.WriteFile("/data/adb/junkclean/junkclean.bak", data, 0755)
	}

	// 覆盖模块目录（运行时数据 /data/adb/junkclean 不受影响）
	copyTree(tmpDir, modDir)

	// 启动新 daemon（setsid 独立，等待旧进程释放端口）
	os.MkdirAll("/data/adb/junkclean", 0700)
	shPath := "/system/bin/sh"
	if _, err := os.Stat(shPath); err != nil {
		shPath = "/bin/sh"
	}
	cmd := exec.Command(shPath, "-c",
		"setsid "+modDir+"/system/bin/junkclean daemon >> /data/adb/junkclean/daemon.log 2>&1 &")
	if err := cmd.Start(); err != nil {
		logLine("hotupdate: 启动新 daemon 失败: " + err.Error())
		writeJSON(w, 500, map[string]string{"error": "启动失败: " + err.Error()})
		return
	}

	logLine("hotupdate: 新版本已应用，重启 daemon")
	writeJSON(w, 200, map[string]string{"status": "ok", "msg": "热更新完成，daemon 重启中"})

	// 响应发送后退出旧进程（释放端口）
	go func() {
		time.Sleep(600 * time.Millisecond)
		os.Exit(0)
	}()
}

// ----- 一键全清（安全分类） -----

func apiCleanall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	scanSt.mu.Lock()
	if !scanSt.done {
		scanSt.mu.Unlock()
		writeJSON(w, 409, map[string]string{"error": "请先扫描"})
		return
	}
	var ids []string
	for _, c := range scanSt.categories {
		if c.Careful {
			continue
		}
		for _, it := range c.Items {
			ids = append(ids, it.ID)
		}
	}
	before := diskInfo("/data")
	freed, errs := scanSt.clean(ids)
	scanSt.done, scanSt.categories = false, nil
	scanSt.mu.Unlock()
	saveState()
	after := diskInfo("/data")
	recordStat(len(ids))
	logLine(fmt.Sprintf("cleanall: %d items, freed %s", len(ids), fmtSize(freed)))
	writeJSON(w, 200, map[string]any{"freed": freed, "items": len(ids), "errors": errs, "before": before, "after": after})
}

// ----- 路径存在性检查 -----

func apiCheck(w http.ResponseWriter, r *http.Request) {
	var req struct{ Paths []string `json:"paths"` }
	json.NewDecoder(r.Body).Decode(&req)
	result := map[string]bool{}
	for _, p := range req.Paths {
		_, err := os.Stat(p)
		result[p] = err == nil
	}
	writeJSON(w, 200, map[string]any{"exist": result})
}

// ----- AI analysis (optional) -----

func apiAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "POST only"})
		return
	}
	cfgLock.RLock()
	c := cfg
	cfgLock.RUnlock()
	if c.AIEndpoint == "" || c.AIKey == "" {
		writeJSON(w, 400, map[string]string{"error": "AI not configured"})
		return
	}
	var req struct {
		Summary string `json:"summary"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	prompt := "你是存储清理专家。分析以下垃圾清理扫描结果摘要，给出简短的清理建议（中文，100字内），标注哪些必须清、哪些建议保留：\n" + req.Summary
	body, _ := json.Marshal(map[string]any{
		"model": orDefault(c.AIModel, "gpt-4o-mini"),
		"messages": []map[string]string{
			{"role": "system", "content": "你是一名专业的 Android 存储清理顾问。"},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 300,
	})
	httpReq, err := http.NewRequest("POST", c.AIEndpoint+"/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.AIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	text := ""
	if len(out.Choices) > 0 {
		text = out.Choices[0].Message.Content
	}
	writeJSON(w, 200, map[string]any{"advice": text})
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// ----- CLI -----

func cliScan() int {
	scanSt.run()
	scanSt.mu.Lock()
	defer scanSt.mu.Unlock()
	b, _ := json.MarshalIndent(scanSt.categories, "", "  ")
	fmt.Println(string(b))
	fmt.Printf("TOTAL: %s\n", fmtSize(scanSt.total))
	return 0
}

func cliClean(ids []string) int {
	if len(ids) == 0 {
		fmt.Println("usage: junkclean clean <id1,id2,...>")
		return 1
	}
	loadState()
	scanSt.mu.Lock()
	freed, errs := scanSt.clean(ids)
	scanSt.done = false
	scanSt.mu.Unlock()
	fmt.Printf("freed %s\n", fmtSize(freed))
	for _, e := range errs {
		fmt.Println("ERR", e)
	}
	return 0
}

// ----- main -----

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scan":
			os.Exit(cliScan())
		case "clean":
			os.Exit(cliClean(strings.Split(strings.Join(os.Args[2:], ","), ",")))
		case "daemon", "serve":
			// fallthrough to daemon
		default:
			fmt.Println("usage: junkclean [scan|clean <ids>|daemon]")
			os.Exit(1)
		}
	}
	os.MkdirAll(stateDir, 0700)
	loadConfig()
	webSub, err := fs.Sub(webFS, "webroot")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", http.FileServer(http.FS(webSub)))
	http.HandleFunc("/api/status", corsMiddleware(apiStatus))
	http.HandleFunc("/api/scan", corsMiddleware(apiScan))
	http.HandleFunc("/api/result", corsMiddleware(apiResult))
	http.HandleFunc("/api/clean", corsMiddleware(apiClean))
	http.HandleFunc("/api/config", corsMiddleware(apiConfig))
	http.HandleFunc("/api/logs", corsMiddleware(apiLogs))
	http.HandleFunc("/api/ai", corsMiddleware(apiAI))
	http.HandleFunc("/api/hotupdate", corsMiddleware(apiHotupdate))
	http.HandleFunc("/api/updateinfo", corsMiddleware(apiUpdateinfo))
	http.HandleFunc("/api/whitelist/add", corsMiddleware(apiWhitelistAdd))
	http.HandleFunc("/api/apkcheck", corsMiddleware(apiApkCheck))
	http.HandleFunc("/api/cleanall", corsMiddleware(apiCleanall))
	http.HandleFunc("/api/check", corsMiddleware(apiCheck))
	registerFeatures(http.DefaultServeMux)
	registerV43(http.DefaultServeMux)
	startScheduler()
	fmt.Printf("JunkClean v%s daemon on 127.0.0.1:%s (root=%v)\n", ver, port, os.Geteuid() == 0)
	// 端口被占时等待旧进程退出（热更新重启场景），最多 5s
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		for i := 0; i < 50 && err != nil; i++ {
			time.Sleep(100 * time.Millisecond)
			ln, err = net.Listen("tcp", "127.0.0.1:"+port)
		}
		if err != nil {
			log.Fatal("端口 " + port + " 被占用: " + err.Error())
		}
	}
	srv := &http.Server{}
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		logLine("daemon: 收到退出信号，保存状态")
		saveConfig()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Close()
		_ = ctx
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
