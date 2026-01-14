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
}

func NewDatabases(databaseType string) Databases {
	klog.Infof("databases type: %v", databaseType)

	switch databaseType {
	//case "mysql":
	//	return NewMysql()
	case "mongodb":
		return NewMongodb()
	default:
		return nil
	}
}
