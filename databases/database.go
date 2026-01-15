package databases

import (
	"bwrs/tools"

	"k8s.io/klog"
)

/*
Databases interface
A global interface has been implemented here
and all database links need to meet this interface
add the new method that needs to be called here
and implement it in the corresponding named file
*/
type Databases interface {
	// Init init func conn databases return conn,err
	Init(config tools.ServiceConfig)
	EnsureUserLoginTable()
	GetUserByUsername(username string) (*UserLogin, error)
	SetUserToken(id int64, token string) error
	UpsertUser(username string, password string) error
	EnsureButtonsTable()
	ListButtons() ([]Button, error)
	AddButton(name string, url string, typ string) error
	DeleteButton(id int64) error
	DatabaseMeta() (DatabaseMeta, error)
	ListTables() ([]string, error)
	EnsureLocalIndexBindingTable()
	CreateLocalIndexTable(table string) error
	SaveLocalIndexEntries(table string, entries []LocalEntry) error
	UpsertLocalIndexBinding(table string, displayName string, description string) error
	ListLocalIndexBindings() ([]LocalIndexBinding, error)
	GetLocalIndexBinding(table string) (*LocalIndexBinding, error)
	ListLocalIndexEntries(table string, offset int, limit int) ([]LocalEntry, int, error)
	CountLocalIndexEntries(table string) (int, error)
	SearchLocalIndexEntries(table string, q string, offset int, limit int) ([]LocalEntry, int, error)
	EnsureTagsTables()
	SearchTags(q string) ([]Tag, error)
	ListAllTags() ([]Tag, error)
	AddTag(name string) error
	UpsertFavorite(dirPath string, originalName string, favoriteName string, description string) error
	SetDirectoryTags(dirPath string, tags []string) error
	ListFavorites(q string, page int, pageSize int, tags []string) ([]Favorite, int, error)
	EnsureDownloadSaveTable()
	GetDownloadSaveByName(name string) (*DownloadSave, error)
	InsertDownloadSave(name string, group string, desc string, localAddress string) (int64, error)
	EnsureAskTable()
	CreateAsk() (*AskKey, error)
	ListAsk() ([]AskKey, error)
	CheckAsk(token string) (bool, error)
	DeleteAsk(id int64) error
}

func NewDatabases(databaseType string) Databases {
	klog.Infof("databases type: %v", databaseType)

	switch databaseType {
	case "mysql":
		return NewMysql()
	case "mongodb":
		return NewMongodb()
	default:
		return nil
	}
}

type UserLogin struct {
	Id       int64
	Username string
	Password string
	Token    string
}

type Button struct {
	Id   int64
	Name string
	Url  string
	Type string
}

type DatabaseMeta struct {
	Type string
	Name string
	Host string
	Port int
}

type LocalEntry struct {
	Path  string
	Type  string
	Size  int64
	Mtime int64
}

type Tag struct {
	Id   int64
	Name string
}

type LocalIndexBinding struct {
	TableName   string
	DisplayName string
	Description string
	CreatedAt   string
}

type Favorite struct {
	DirPath      string
	DirHash      string
	OriginalName string
	FavoriteName string
	Description  string
	CreatedAt    string
	Tags         []string
}

type DownloadSave struct {
	Id           int64
	Group        string
	Name         string
	Desc         string
	LocalAddress string
}

type AskKey struct {
	Id  int64
	Ask string
}
