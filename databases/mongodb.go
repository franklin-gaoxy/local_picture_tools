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
