package server

import (
	"bwrs/databases"
	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

/*
This is the function that implements all interfaces of gin
*/

func initEnvironment(str string) {
	klog.Info(str)
}

func Test(c *gin.Context, database databases.Databases) {
	klog.Info(c.Request.RequestURI, database)
}
