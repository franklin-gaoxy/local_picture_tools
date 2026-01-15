package databases

import (
	"bwrs/tools"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"k8s.io/klog"
)

type Mysql struct {
	db     *sql.DB
	dbName string
	host   string
	port   int
}

func NewMysql() *Mysql {
	return &Mysql{}
}

func (m *Mysql) Init(config tools.ServiceConfig) {
	user := config.Database.Description.Username
	pass := config.Database.Description.Password
	m.host = config.Database.Host
	m.port = int(config.Database.Port)
	m.dbName = config.Database.Path

	dsnServer := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true&charset=utf8mb4&multiStatements=true", user, pass, m.host, m.port)
	dbServer, err := sql.Open("mysql", dsnServer)
	if err != nil {
		klog.Fatal(err)
	}
	if err = dbServer.Ping(); err != nil {
		klog.Fatal(err)
	}
	_, err = dbServer.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", m.dbName))
	if err != nil {
		klog.Fatal(err)
	}
	_ = dbServer.Close()

	dsnDB := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&multiStatements=true", user, pass, m.host, m.port, m.dbName)
	m.db, err = sql.Open("mysql", dsnDB)
	if err != nil {
		klog.Fatal(err)
	}
	if err = m.db.Ping(); err != nil {
		klog.Fatal(err)
	}
}

func (m *Mysql) EnsureUserLoginTable() {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS userlogin (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(255) UNIQUE NOT NULL,
  password VARCHAR(255) NOT NULL,
  token VARCHAR(255) DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
}

func (m *Mysql) GetUserByUsername(username string) (*UserLogin, error) {
	row := m.db.QueryRow("SELECT id, username, password, token FROM userlogin WHERE username = ? LIMIT 1", username)
	var u UserLogin
	err := row.Scan(&u.Id, &u.Username, &u.Password, &u.Token)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (m *Mysql) SetUserToken(id int64, token string) error {
	_, err := m.db.Exec("UPDATE userlogin SET token = ? WHERE id = ?", token, id)
	return err
}

func (m *Mysql) UpsertUser(username string, password string) error {
	_, err := m.db.Exec(`INSERT INTO userlogin (username, password) VALUES (?, ?)
ON DUPLICATE KEY UPDATE password = VALUES(password)`, username, password)
	return err
}

func (m *Mysql) EnsureButtonsTable() {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS buttons (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  url VARCHAR(1024) NOT NULL,
  type VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
}

func (m *Mysql) ListButtons() ([]Button, error) {
	rows, err := m.db.Query("SELECT id, name, url, type FROM buttons ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Button
	for rows.Next() {
		var b Button
		if err := rows.Scan(&b.Id, &b.Name, &b.Url, &b.Type); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, nil
}

func (m *Mysql) AddButton(name string, url string, typ string) error {
	_, err := m.db.Exec("INSERT INTO buttons (name, url, type) VALUES (?, ?, ?)", name, url, typ)
	return err
}

func (m *Mysql) DeleteButton(id int64) error {
	_, err := m.db.Exec("DELETE FROM buttons WHERE id = ?", id)
	return err
}

func (m *Mysql) DatabaseMeta() (DatabaseMeta, error) {
	return DatabaseMeta{
		Type: "mysql",
		Name: m.dbName,
		Host: m.host,
		Port: m.port,
	}, nil
}

func (m *Mysql) ListTables() ([]string, error) {
	rows, err := m.db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (m *Mysql) EnsureLocalIndexBindingTable() {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS local_index_bindings (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  table_name VARCHAR(255) UNIQUE NOT NULL,
  display_name VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
}

func (m *Mysql) CreateLocalIndexTable(table string) error {
	for _, ch := range table {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return fmt.Errorf("invalid table")
		}
	}
	_, err := m.db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  path TEXT NOT NULL,
  type VARCHAR(16) NOT NULL,
  size BIGINT NOT NULL,
  mtime BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`, table))
	return err
}

func (m *Mysql) SaveLocalIndexEntries(table string, entries []LocalEntry) error {
	for _, ch := range table {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return fmt.Errorf("invalid table")
		}
	}
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(fmt.Sprintf("INSERT INTO %s (path, type, size, mtime) VALUES (?, ?, ?, ?)", table))
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, e := range entries {
		if _, err := stmt.Exec(e.Path, e.Type, e.Size, e.Mtime); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
	}
	_ = stmt.Close()
	return tx.Commit()
}

func (m *Mysql) UpsertLocalIndexBinding(table string, displayName string, description string) error {
	m.EnsureLocalIndexBindingTable()
	_, err := m.db.Exec(`INSERT INTO local_index_bindings (table_name, display_name, description)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE display_name = VALUES(display_name), description = VALUES(description)`, table, displayName, description)
	return err
}

func (m *Mysql) ListLocalIndexBindings() ([]LocalIndexBinding, error) {
	m.EnsureLocalIndexBindingTable()
	rows, err := m.db.Query("SELECT table_name, display_name, description, DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') FROM local_index_bindings ORDER BY created_at DESC, table_name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []LocalIndexBinding
	for rows.Next() {
		var b LocalIndexBinding
		if err := rows.Scan(&b.TableName, &b.DisplayName, &b.Description, &b.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, nil
}

func (m *Mysql) GetLocalIndexBinding(table string) (*LocalIndexBinding, error) {
	m.EnsureLocalIndexBindingTable()
	for _, ch := range table {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return nil, fmt.Errorf("invalid table")
		}
	}
	row := m.db.QueryRow("SELECT table_name, display_name, description, DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') FROM local_index_bindings WHERE table_name = ? LIMIT 1", table)
	var b LocalIndexBinding
	if err := row.Scan(&b.TableName, &b.DisplayName, &b.Description, &b.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (m *Mysql) ListLocalIndexEntries(table string, offset int, limit int) ([]LocalEntry, int, error) {
	for _, ch := range table {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return nil, 0, fmt.Errorf("invalid table")
		}
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	row := m.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	_ = row.Scan(&total)
	query := fmt.Sprintf("SELECT path, type, size, mtime FROM %s ORDER BY id ASC LIMIT ? OFFSET ?", table)
	rows, err := m.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []LocalEntry
	for rows.Next() {
		var e LocalEntry
		if err := rows.Scan(&e.Path, &e.Type, &e.Size, &e.Mtime); err != nil {
			return nil, 0, err
		}
		list = append(list, e)
	}
	return list, total, nil
}

func (m *Mysql) CountLocalIndexEntries(table string) (int, error) {
	for _, ch := range table {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return 0, fmt.Errorf("invalid table")
		}
	}
	var total int
	row := m.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (m *Mysql) SearchLocalIndexEntries(table string, q string, offset int, limit int) ([]LocalEntry, int, error) {
	for _, ch := range table {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_') {
			return nil, 0, fmt.Errorf("invalid table")
		}
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	row := m.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE path LIKE ?", table), "%"+q+"%")
	_ = row.Scan(&total)
	query := fmt.Sprintf("SELECT path, type, size, mtime FROM %s WHERE path LIKE ? ORDER BY id ASC LIMIT ? OFFSET ?", table)
	rows, err := m.db.Query(query, "%"+q+"%", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []LocalEntry
	for rows.Next() {
		var e LocalEntry
		if err := rows.Scan(&e.Path, &e.Type, &e.Size, &e.Mtime); err != nil {
			return nil, 0, err
		}
		list = append(list, e)
	}
	return list, total, nil
}

func (m *Mysql) EnsureTagsTables() {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS tags (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) UNIQUE NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
	_, err = m.db.Exec(`CREATE TABLE IF NOT EXISTS favorites (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  dir_path TEXT NOT NULL,
  dir_hash CHAR(64) UNIQUE NOT NULL,
  original_name VARCHAR(255) NOT NULL,
  favorite_name VARCHAR(255) NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
	_, err = m.db.Exec(`CREATE TABLE IF NOT EXISTS dir_tag_map (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  dir_hash CHAR(64) NOT NULL,
  tag_id BIGINT NOT NULL,
  UNIQUE KEY uniq_dir_tag (dir_hash, tag_id),
  INDEX idx_tag (tag_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
	// Ensure favorites.dir_hash exists and backfill
	var cnt int
	row := m.db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = 'favorites' AND column_name = 'dir_hash'", m.dbName)
	_ = row.Scan(&cnt)
	if cnt == 0 {
		if _, err := m.db.Exec("ALTER TABLE favorites ADD COLUMN dir_hash CHAR(64)"); err != nil {
			klog.Fatal(err)
		}
	}
	if _, err := m.db.Exec("UPDATE favorites SET dir_hash = SHA2(dir_path, 256) WHERE dir_hash IS NULL OR dir_hash = ''"); err != nil {
		klog.Fatal(err)
	}
	// Ensure unique index on favorites.dir_hash
	row = m.db.QueryRow("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = ? AND table_name = 'favorites' AND index_name = 'uniq_dir_hash'", m.dbName)
	cnt = 0
	_ = row.Scan(&cnt)
	if cnt == 0 {
		if _, err := m.db.Exec("ALTER TABLE favorites ADD UNIQUE KEY uniq_dir_hash (dir_hash)"); err != nil {
			klog.Fatal(err)
		}
	}
	// Ensure dir_tag_map.dir_hash exists and unique index on (dir_hash, tag_id)
	row = m.db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = 'dir_tag_map' AND column_name = 'dir_hash'", m.dbName)
	cnt = 0
	_ = row.Scan(&cnt)
	if cnt == 0 {
		if _, err := m.db.Exec("ALTER TABLE dir_tag_map ADD COLUMN dir_hash CHAR(64)"); err != nil {
			klog.Fatal(err)
		}
		// Backfill dir_hash from dir_path if present
		_, _ = m.db.Exec("UPDATE dir_tag_map SET dir_hash = SHA2(dir_path, 256) WHERE dir_hash IS NULL OR dir_hash = ''")
	}
	row = m.db.QueryRow("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = ? AND table_name = 'dir_tag_map' AND index_name = 'uniq_dir_tag'", m.dbName)
	cnt = 0
	_ = row.Scan(&cnt)
	if cnt == 0 {
		if _, err := m.db.Exec("ALTER TABLE dir_tag_map ADD UNIQUE KEY uniq_dir_tag (dir_hash, tag_id)"); err != nil {
			klog.Fatal(err)
		}
	}
}

func (m *Mysql) SearchTags(q string) ([]Tag, error) {
	rows, err := m.db.Query("SELECT id, name FROM tags WHERE name LIKE ? ORDER BY name ASC LIMIT 20", "%"+q+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.Id, &t.Name); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (m *Mysql) ListAllTags() ([]Tag, error) {
	rows, err := m.db.Query("SELECT id, name FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.Id, &t.Name); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (m *Mysql) AddTag(name string) error {
	_, err := m.db.Exec("INSERT IGNORE INTO tags (name) VALUES (?)", name)
	return err
}

func (m *Mysql) UpsertFavorite(dirPath string, originalName string, favoriteName string, description string) error {
	h := sha256.Sum256([]byte(dirPath))
	dirHash := hex.EncodeToString(h[:])
	_, err := m.db.Exec(`INSERT INTO favorites (dir_path, dir_hash, original_name, favorite_name, description)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE favorite_name = VALUES(favorite_name), description = VALUES(description)`,
		dirPath, dirHash, originalName, favoriteName, description)
	return err
}

func (m *Mysql) SetDirectoryTags(dirPath string, tags []string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(dirPath))
	dirHash := hex.EncodeToString(h[:])
	for _, name := range tags {
		if name == "" {
			continue
		}
		if _, err := tx.Exec("INSERT IGNORE INTO tags (name) VALUES (?)", strings.TrimSpace(name)); err != nil {
			_ = tx.Rollback()
			return err
		}
		var tagID int64
		row := tx.QueryRow("SELECT id FROM tags WHERE name = ? LIMIT 1", strings.TrimSpace(name))
		if err := row.Scan(&tagID); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec("INSERT IGNORE INTO dir_tag_map (dir_hash, tag_id) VALUES (?, ?)", dirHash, tagID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (m *Mysql) ListFavorites(q string, page int, pageSize int, tags []string) ([]Favorite, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	where := "1=1"
	args := []interface{}{}
	if q != "" {
		where += " AND (favorite_name LIKE ? OR original_name LIKE ? OR description LIKE ? OR dir_path LIKE ?)"
		pat := "%" + q + "%"
		args = append(args, pat, pat, pat, pat)
	}
	tagFilter := ""
	if len(tags) > 0 {
		place := make([]string, 0, len(tags))
		for range tags {
			place = append(place, "?")
		}
		tagFilter = fmt.Sprintf(` AND dir_hash IN (
  SELECT dt.dir_hash FROM dir_tag_map dt
  JOIN tags t ON dt.tag_id = t.id
  WHERE t.name IN (%s)
  GROUP BY dt.dir_hash
  HAVING COUNT(DISTINCT t.name) = ?
)`, strings.Join(place, ","))
		args = append(args, toAnySlice(tags)...)
		args = append(args, len(tags))
	}
	var total int
	countSQL := "SELECT COUNT(*) FROM favorites WHERE " + where + tagFilter
	row := m.db.QueryRow(countSQL, args...)
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}
	query := "SELECT dir_path, dir_hash, original_name, favorite_name, description, DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') FROM favorites WHERE " + where + tagFilter + " ORDER BY created_at DESC, favorite_name ASC LIMIT ? OFFSET ?"
	argsQ := append(args, pageSize, offset)
	rows, err := m.db.Query(query, argsQ...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.DirPath, &f.DirHash, &f.OriginalName, &f.FavoriteName, &f.Description, &f.CreatedAt); err != nil {
			return nil, 0, err
		}
		trows, terr := m.db.Query("SELECT t.name FROM dir_tag_map dt JOIN tags t ON dt.tag_id = t.id WHERE dt.dir_hash = ? ORDER BY t.name ASC", f.DirHash)
		if terr == nil {
			var tags []string
			for trows.Next() {
				var name string
				if err := trows.Scan(&name); err == nil {
					tags = append(tags, name)
				}
			}
			_ = trows.Close()
			f.Tags = tags
		}
		list = append(list, f)
	}
	return list, total, nil
}

func toAnySlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func (m *Mysql) EnsureDownloadSaveTable() {
	_, err := m.db.Exec("CREATE TABLE IF NOT EXISTS downloadsave (" +
		"  id BIGINT PRIMARY KEY AUTO_INCREMENT," +
		"  `group` VARCHAR(255)," +
		"  name VARCHAR(255) NOT NULL," +
		"  `desc` TEXT," +
		"  local_address TEXT," +
		"  UNIQUE KEY uniq_name (name)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	if err != nil {
		klog.Fatal(err)
	}
	// Ensure local_address column exists (for backfill upgrades)
	var cnt int
	row := m.db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = 'downloadsave' AND column_name = 'local_address'", m.dbName)
	_ = row.Scan(&cnt)
	if cnt == 0 {
		if _, err := m.db.Exec("ALTER TABLE downloadsave ADD COLUMN local_address TEXT"); err != nil {
			klog.Fatal(err)
		}
	}
}

func (m *Mysql) GetDownloadSaveByName(name string) (*DownloadSave, error) {
	row := m.db.QueryRow("SELECT id, `group`, name, `desc`, local_address FROM downloadsave WHERE name = ? LIMIT 1", name)
	var d DownloadSave
	err := row.Scan(&d.Id, &d.Group, &d.Name, &d.Desc, &d.LocalAddress)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (m *Mysql) InsertDownloadSave(name string, group string, desc string, localAddress string) (int64, error) {
	res, err := m.db.Exec("INSERT INTO downloadsave (`group`, name, `desc`, local_address) VALUES (?, ?, ?, ?)", group, name, desc, localAddress)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (m *Mysql) EnsureAskTable() {
	_, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS ask_keys (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  ask VARCHAR(32) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	if err != nil {
		klog.Fatal(err)
	}
}

func (m *Mysql) CreateAsk() (*AskKey, error) {
	m.EnsureAskTable()
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, 20)
	for i := 0; i < 20; i++ {
		buf[i] = letters[randInt(len(letters))]
	}
	ask := string(buf)
	res, err := m.db.Exec("INSERT INTO ask_keys (ask) VALUES (?)", ask)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &AskKey{Id: id, Ask: ask}, nil
}

func (m *Mysql) ListAsk() ([]AskKey, error) {
	rows, err := m.db.Query("SELECT id, ask FROM ask_keys ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AskKey
	for rows.Next() {
		var a AskKey
		if err := rows.Scan(&a.Id, &a.Ask); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}

func (m *Mysql) CheckAsk(token string) (bool, error) {
	row := m.db.QueryRow("SELECT COUNT(*) FROM ask_keys WHERE ask = ? LIMIT 1", token)
	var cnt int
	if err := row.Scan(&cnt); err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func randInt(n int) int {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return int(b[0]) % n
}

func (m *Mysql) DeleteAsk(id int64) error {
	_, err := m.db.Exec("DELETE FROM ask_keys WHERE id = ?", id)
	return err
}
