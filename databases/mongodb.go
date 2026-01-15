package databases

import (
	"bwrs/tools"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"k8s.io/klog"
)

type Mongodb struct {
	client   *mongo.Client
	database string
}

func NewMongodb() *Mongodb {
	return &Mongodb{}
}
func (m *Mongodb) Init(ServiceConfig tools.ServiceConfig) {
	var err error
	klog.V(5).Info("begin exec MongoDB init")
	klog.Info(ServiceConfig)

	// format databases
	m.database = ServiceConfig.Database.BaseName

	// 如果连接串不为空则格式化 构建 MongoDB 连接字符串
	var connpath string
	if ServiceConfig.Database.ConnPath != "" {
		connpath = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s", ServiceConfig.Database.Description.Username,
			ServiceConfig.Database.Description.Password, ServiceConfig.Database.Host,
			ServiceConfig.Database.Port, ServiceConfig.Database.BaseName, ServiceConfig.Database.AuthSource)
	} else {
		connpath = ServiceConfig.Database.ConnPath
	}

	klog.V(8).Infoln("Connecting to string:", connpath)
	clientOptions := options.Client().ApplyURI(connpath)
	m.client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		klog.Fatal(connpath, err)
	}

	err = m.client.Ping(context.TODO(), nil)
	if err != nil {
		klog.Fatal(err)
	}

	klog.V(5).Info("Successfully connected to MongoDB")
}

func (m *Mongodb) EnsureUserLoginTable() {}
func (m *Mongodb) GetUserByUsername(username string) (*UserLogin, error) {
	return nil, fmt.Errorf("unsupported")
}
func (m *Mongodb) SetUserToken(id int64, token string) error { return fmt.Errorf("unsupported") }
func (m *Mongodb) UpsertUser(username string, password string) error {
	return fmt.Errorf("unsupported")
}
func (m *Mongodb) EnsureButtonsTable()            {}
func (m *Mongodb) ListButtons() ([]Button, error) { return nil, fmt.Errorf("unsupported") }
func (m *Mongodb) AddButton(name string, url string, typ string) error {
	return fmt.Errorf("unsupported")
}
func (m *Mongodb) DeleteButton(id int64) error { return fmt.Errorf("unsupported") }
func (m *Mongodb) DatabaseMeta() (DatabaseMeta, error) {
	return DatabaseMeta{Type: "mongodb", Name: m.database, Host: "", Port: 0}, nil
}
func (m *Mongodb) ListTables() ([]string, error)            { return nil, fmt.Errorf("unsupported") }
func (m *Mongodb) EnsureLocalIndexBindingTable()            {}
func (m *Mongodb) CreateLocalIndexTable(table string) error { return fmt.Errorf("unsupported") }
func (m *Mongodb) SaveLocalIndexEntries(table string, entries []LocalEntry) error {
	return fmt.Errorf("unsupported")
}
func (m *Mongodb) UpsertLocalIndexBinding(table string, displayName string, description string) error {
	return fmt.Errorf("unsupported")
}
func (m *Mongodb) ListLocalIndexBindings() ([]LocalIndexBinding, error) {
	return nil, fmt.Errorf("unsupported")
}
func (m *Mongodb) GetLocalIndexBinding(table string) (*LocalIndexBinding, error) {
	return nil, fmt.Errorf("unsupported")
}
func (m *Mongodb) ListLocalIndexEntries(table string, offset int, limit int) ([]LocalEntry, int, error) {
	return nil, 0, fmt.Errorf("unsupported")
}
func (m *Mongodb) CountLocalIndexEntries(table string) (int, error) {
	return 0, fmt.Errorf("unsupported")
}
func (m *Mongodb) SearchLocalIndexEntries(table string, q string, offset int, limit int) ([]LocalEntry, int, error) {
	return nil, 0, fmt.Errorf("unsupported")
}
func (m *Mongodb) EnsureTagsTables()                  {}
func (m *Mongodb) SearchTags(q string) ([]Tag, error) { return nil, fmt.Errorf("unsupported") }
func (m *Mongodb) ListAllTags() ([]Tag, error)        { return nil, fmt.Errorf("unsupported") }
func (m *Mongodb) AddTag(name string) error           { return fmt.Errorf("unsupported") }
func (m *Mongodb) UpsertFavorite(dirPath string, originalName string, favoriteName string, description string) error {
	return fmt.Errorf("unsupported")
}
func (m *Mongodb) SetDirectoryTags(dirPath string, tags []string) error {
	return fmt.Errorf("unsupported")
}
func (m *Mongodb) ListFavorites(q string, page int, pageSize int, tags []string) ([]Favorite, int, error) {
	return nil, 0, fmt.Errorf("unsupported")
}

func (m *Mongodb) EnsureDownloadSaveTable() {}
func (m *Mongodb) GetDownloadSaveByName(name string) (*DownloadSave, error) {
	return nil, fmt.Errorf("unsupported")
}
func (m *Mongodb) InsertDownloadSave(name string, group string, desc string, localAddress string) (int64, error) {
	return 0, fmt.Errorf("unsupported")
}
func (m *Mongodb) EnsureAskTable() {}
func (m *Mongodb) CreateAsk() (*AskKey, error) {
	return nil, fmt.Errorf("unsupported")
}
func (m *Mongodb) ListAsk() ([]AskKey, error) {
	return nil, fmt.Errorf("unsupported")
}
func (m *Mongodb) CheckAsk(token string) (bool, error) {
	return false, fmt.Errorf("unsupported")
}
func (m *Mongodb) DeleteAsk(id int64) error {
	return fmt.Errorf("unsupported")
}
