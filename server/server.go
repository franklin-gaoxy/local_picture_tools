package server

import (
	"bwrs/databases"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

/*
This is the function that implements all interfaces of gin
*/

type Session struct {
	UserID   int64
	Username string
}

var sessionStore = struct {
	mu sync.Mutex
	m  map[string]Session
}{
	m: make(map[string]Session),
}

type RequestRecord struct {
	Time   string
	Method string
	Path   string
	IP     string
	Params map[string]string
}

var metrics = struct {
	mu       sync.Mutex
	start    time.Time
	counters map[string]int
	total    int
	records  []RequestRecord
}{
	start:    time.Now(),
	counters: make(map[string]int),
	records:  make([]RequestRecord, 0, 200),
}

func TrackMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if !(path == "/api/server/save" || path == "/api/server/duplicate") {
			return
		}
		method := c.Request.Method
		ip := c.ClientIP()
		params := map[string]string{}
		for k, v := range c.Request.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
		_ = c.Request.ParseForm()
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
		metrics.mu.Lock()
		metrics.total++
		metrics.counters[fmt.Sprintf("%s %s", method, path)]++
		rec := RequestRecord{
			Time:   start.Format("2006-01-02 15:04:05"),
			Method: method,
			Path:   path,
			IP:     ip,
			Params: params,
		}
		metrics.records = append(metrics.records, rec)
		if len(metrics.records) > 200 {
			metrics.records = metrics.records[len(metrics.records)-200:]
		}
		metrics.mu.Unlock()
	}
}

func initEnvironment(str string) {
	klog.Info(str)
}

func Test(c *gin.Context, database databases.Databases) {
	klog.Info(c.Request.RequestURI, database)
}

func Login(c *gin.Context, database databases.Databases) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "missing credentials"})
		return
	}
	u, err := database.GetUserByUsername(username)
	if err != nil || u == nil {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}
	if u.Password != password {
		c.JSON(401, gin.H{"error": "invalid credentials"})
		return
	}
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := hex.EncodeToString(buf)
	_ = database.SetUserToken(u.Id, token)
	sessionStore.mu.Lock()
	sessionStore.m[token] = Session{UserID: u.Id, Username: u.Username}
	sessionStore.mu.Unlock()
	c.SetCookie("session_token", token, 3600, "/", "", false, true)
	c.JSON(200, gin.H{"ok": true})
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil || token == "" {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}
		sessionStore.mu.Lock()
		_, ok := sessionStore.m[token]
		sessionStore.mu.Unlock()
		if !ok {
			c.Redirect(302, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func AuthRequiredAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil || token == "" {
			c.JSON(401, gin.H{"error": "unauthenticated"})
			c.Abort()
			return
		}
		sessionStore.mu.Lock()
		_, ok := sessionStore.m[token]
		sessionStore.mu.Unlock()
		if !ok {
			c.JSON(401, gin.H{"error": "unauthenticated"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func ButtonsList(c *gin.Context, database databases.Databases) {
	list, err := database.ListButtons()
	if err != nil {
		c.JSON(500, gin.H{"error": "list failed"})
		return
	}
	c.JSON(200, gin.H{"items": list})
}

func ButtonsAdd(c *gin.Context, database databases.Databases) {
	name := c.PostForm("name")
	url := c.PostForm("url")
	typ := c.PostForm("type")
	if name == "" || url == "" || typ == "" {
		c.JSON(400, gin.H{"error": "missing fields"})
		return
	}
	if err := database.AddButton(name, url, typ); err != nil {
		c.JSON(500, gin.H{"error": "add failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func ButtonsDelete(c *gin.Context, database databases.Databases) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id <= 0 {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if err := database.DeleteButton(id); err != nil {
		c.JSON(500, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func Dashboard(c *gin.Context, database databases.Databases) {
	meta, _ := database.DatabaseMeta()
	tables, _ := database.ListTables()
	proto := c.Request.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.Request.Host
	remote := c.ClientIP()
	path := c.Request.URL.Path
	metrics.mu.Lock()
	uptime := int(time.Since(metrics.start).Seconds())
	counters := make([]gin.H, 0, len(metrics.counters))
	for k, v := range metrics.counters {
		if strings.HasSuffix(k, " /api/server/save") || strings.HasSuffix(k, " /api/server/duplicate") {
			counters = append(counters, gin.H{"name": k, "count": v})
		}
	}
	recent := make([]gin.H, 0, len(metrics.records))
	for _, r := range metrics.records {
		if r.Path == "/api/server/save" || r.Path == "/api/server/duplicate" {
			recent = append(recent, gin.H{
				"time":   r.Time,
				"method": r.Method,
				"path":   r.Path,
				"ip":     r.IP,
				"params": r.Params,
			})
		}
	}
	total := metrics.total
	metrics.mu.Unlock()
	c.JSON(200, gin.H{
		"database": gin.H{
			"type":   meta.Type,
			"name":   meta.Name,
			"host":   meta.Host,
			"port":   meta.Port,
			"tables": tables,
		},
		"status": gin.H{
			"startTime":     metrics.start.Format("2006-01-02 15:04:05"),
			"uptimeSec":     uptime,
			"totalRequests": total,
		},
		"request": gin.H{
			"proto":  proto,
			"host":   host,
			"remote": remote,
			"path":   path,
		},
		"interfaces": counters,
		"requests":   recent,
	})
}

func LocalList(c *gin.Context, database databases.Databases) {
	typ := c.PostForm("type")
	root := c.PostForm("path")
	if typ != "local" || root == "" {
		c.JSON(400, gin.H{"error": "invalid params"})
		return
	}
	var items []databases.LocalEntry
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		t := "file"
		if d.IsDir() {
			t = "dir"
		}
		items = append(items, databases.LocalEntry{
			Path:  p,
			Type:  t,
			Size:  info.Size(),
			Mtime: info.ModTime().Unix(),
		})
		return nil
	})
	if err != nil {
		c.JSON(500, gin.H{"error": "scan failed"})
		return
	}
	c.JSON(200, gin.H{"items": items})
}

func LocalIndex(c *gin.Context, database databases.Databases) {
	tableBase := c.PostForm("table")
	displayName := c.PostForm("display")
	desc := c.PostForm("desc")
	root := c.PostForm("path")
	if tableBase == "" || displayName == "" || root == "" {
		c.JSON(400, gin.H{"error": "missing fields"})
		return
	}
	for _, ch := range tableBase {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			c.JSON(400, gin.H{"error": "invalid table name"})
			return
		}
	}
	table := "local_index_" + tableBase
	_ = database.CreateLocalIndexTable(table)
	var items []databases.LocalEntry
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		t := "file"
		if d.IsDir() {
			t = "dir"
		}
		items = append(items, databases.LocalEntry{
			Path:  p,
			Type:  t,
			Size:  info.Size(),
			Mtime: info.ModTime().Unix(),
		})
		return nil
	})
	if err := database.SaveLocalIndexEntries(table, items); err != nil {
		c.JSON(500, gin.H{"error": "save failed"})
		return
	}
	if err := database.UpsertLocalIndexBinding(table, displayName, desc); err != nil {
		c.JSON(500, gin.H{"error": "bind failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true, "table": table, "count": len(items)})
}

func LocalFile(c *gin.Context, database databases.Databases) {
	path := c.Query("path")
	if path == "" {
		c.JSON(400, gin.H{"error": "missing path"})
		return
	}
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	ext := strings.ToLower(filepath.Ext(path))
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}
	c.Header("Content-Type", ct)
	c.File(path)
}

func TagsSearch(c *gin.Context, database databases.Databases) {
	q := c.Query("q")
	list, _ := database.SearchTags(q)
	c.JSON(200, gin.H{"items": list})
}

func TagsAdd(c *gin.Context, database databases.Databases) {
	name := c.PostForm("name")
	if name == "" {
		c.JSON(400, gin.H{"error": "missing name"})
		return
	}
	if err := database.AddTag(name); err != nil {
		c.JSON(500, gin.H{"error": "add failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func TagsAll(c *gin.Context, database databases.Databases) {
	list, _ := database.ListAllTags()
	c.JSON(200, gin.H{"items": list})
}

func FavoriteSave(c *gin.Context, database databases.Databases) {
	path := c.PostForm("path")
	orig := c.PostForm("original")
	fav := c.PostForm("favorite")
	desc := c.PostForm("desc")
	tagsStr := c.PostForm("tags")
	if path == "" || orig == "" || fav == "" {
		c.JSON(400, gin.H{"error": "missing fields"})
		return
	}
	if err := database.UpsertFavorite(path, orig, fav, desc); err != nil {
		c.JSON(500, gin.H{"error": "save favorite failed"})
		return
	}
	var tags []string
	for _, s := range strings.Split(tagsStr, ",") {
		ss := strings.TrimSpace(s)
		if ss != "" {
			tags = append(tags, ss)
		}
	}
	if err := database.SetDirectoryTags(path, tags); err != nil {
		c.JSON(500, gin.H{"error": "set tags failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}

func FavoritesList(c *gin.Context, database databases.Databases) {
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if s := c.Query("pageSize"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			pageSize = v
		}
	}
	q := c.Query("q")
	tagsStr := c.Query("tags")
	var tags []string
	if tagsStr != "" {
		for _, s := range strings.Split(tagsStr, ",") {
			ss := strings.TrimSpace(s)
			if ss != "" {
				tags = append(tags, ss)
			}
		}
	}
	items, total, err := database.ListFavorites(q, page, pageSize, tags)
	if err != nil {
		c.JSON(500, gin.H{"error": "list failed"})
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "pageSize": pageSize})
}
func IndexesList(c *gin.Context, database databases.Databases) {
	tables, _ := database.ListTables()
	var bindings map[string]databases.LocalIndexBinding
	bindings = make(map[string]databases.LocalIndexBinding)
	if bl, err := database.ListLocalIndexBindings(); err == nil {
		for _, b := range bl {
			bindings[b.TableName] = b
		}
	}
	var items []gin.H
	for _, t := range tables {
		if strings.HasPrefix(t, "local_index_") {
			if b, ok := bindings[t]; ok {
				items = append(items, gin.H{
					"TableName":   b.TableName,
					"DisplayName": b.DisplayName,
					"Description": b.Description,
					"CreatedAt":   b.CreatedAt,
				})
			} else {
				items = append(items, gin.H{
					"TableName":   t,
					"DisplayName": t,
					"Description": "",
					"CreatedAt":   "",
				})
			}
		}
	}
	c.JSON(200, gin.H{"items": items})
}

func IndexFiles(c *gin.Context, database databases.Databases) {
	table := c.Query("table")
	offStr := c.Query("offset")
	limStr := c.Query("limit")
	offset, _ := strconv.Atoi(offStr)
	limit, _ := strconv.Atoi(limStr)
	if table == "" {
		c.JSON(400, gin.H{"error": "missing table"})
		return
	}
	items, total, err := database.ListLocalIndexEntries(table, offset, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "list files failed"})
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "offset": offset, "limit": limit})
}

func IndexSearch(c *gin.Context, database databases.Databases) {
	table := c.Query("table")
	q := c.Query("q")
	offStr := c.Query("offset")
	limStr := c.Query("limit")
	offset, _ := strconv.Atoi(offStr)
	limit, _ := strconv.Atoi(limStr)
	if table == "" {
		c.JSON(400, gin.H{"error": "missing table"})
		return
	}
	items, total, err := database.SearchLocalIndexEntries(table, q, offset, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "search failed"})
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total, "offset": offset, "limit": limit, "q": q})
}

func IndexInfo(c *gin.Context, database databases.Databases) {
	table := c.Query("table")
	if table == "" {
		c.JSON(400, gin.H{"error": "missing table"})
		return
	}
	binding, _ := database.GetLocalIndexBinding(table)
	if binding == nil {
		binding = &databases.LocalIndexBinding{
			TableName:   table,
			DisplayName: table,
			Description: "",
			CreatedAt:   "",
		}
	}
	count, _ := database.CountLocalIndexEntries(table)
	c.JSON(200, gin.H{"binding": binding, "count": count})
}

func ServerDuplicate(c *gin.Context, database databases.Databases) {
	ask := c.Query("ask")
	if ask == "" {
		c.JSON(400, gin.H{"status": "false", "error": "missing ask"})
		return
	}
	ok, err := database.CheckAsk(ask)
	if err != nil || !ok {
		c.JSON(403, gin.H{"status": "false", "error": "invalid ask"})
		return
	}
	name := c.Query("name")
	if name == "" {
		c.JSON(400, gin.H{"status": "false", "error": "missing name"})
		return
	}
	database.EnsureDownloadSaveTable()
	row, err := database.GetDownloadSaveByName(name)
	if err != nil {
		c.JSON(500, gin.H{"status": "false"})
		return
	}
	if row != nil {
		c.JSON(200, gin.H{
			"status": "true",
			"data": gin.H{
				"id":            row.Id,
				"group":         row.Group,
				"name":          row.Name,
				"desc":          row.Desc,
				"local_address": row.LocalAddress,
			},
		})
		return
	}
	c.JSON(200, gin.H{"status": "false"})
}

func ServerSave(c *gin.Context, database databases.Databases) {
	ask := c.Query("ask")
	if ask == "" {
		c.JSON(400, gin.H{"status": "false", "error": "missing ask"})
		return
	}
	ok, err := database.CheckAsk(ask)
	if err != nil || !ok {
		c.JSON(403, gin.H{"status": "false", "error": "invalid ask"})
		return
	}
	name := c.Query("name")
	group := c.Query("group")
	desc := c.Query("desc")
	local := c.Query("local_address")
	if name == "" {
		c.JSON(400, gin.H{"status": "false", "error": "missing name"})
		return
	}
	database.EnsureDownloadSaveTable()
	exist, err := database.GetDownloadSaveByName(name)
	if err != nil {
		c.JSON(500, gin.H{"status": "false"})
		return
	}
	if exist != nil {
		c.JSON(200, gin.H{"status": "false", "data": gin.H{
			"id":            exist.Id,
			"group":         exist.Group,
			"name":          exist.Name,
			"desc":          exist.Desc,
			"local_address": exist.LocalAddress,
		}})
		return
	}
	if _, err := database.InsertDownloadSave(name, group, desc, local); err != nil {
		c.JSON(200, gin.H{"status": "false"})
		return
	}
	c.JSON(200, gin.H{"status": "true"})
}

func AskList(c *gin.Context, database databases.Databases) {
	items, err := database.ListAsk()
	if err != nil {
		c.JSON(500, gin.H{"error": "list failed"})
		return
	}
	c.JSON(200, gin.H{"items": items})
}

func AskCreate(c *gin.Context, database databases.Databases) {
	key, err := database.CreateAsk()
	if err != nil {
		c.JSON(500, gin.H{"error": "create failed"})
		return
	}
	c.JSON(200, gin.H{"id": key.Id, "ask": key.Ask})
}

func AskDelete(c *gin.Context, database databases.Databases) {
	idStr := c.Param("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id <= 0 {
		c.JSON(400, gin.H{"error": "invalid id"})
		return
	}
	if err := database.DeleteAsk(id); err != nil {
		c.JSON(500, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
