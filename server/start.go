package server

import (
	"bwrs/databases"
	"bwrs/tools"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/klog"
)

func InitStart() bool {
	klogInit()
	return true
}

func klogInit() {
	klog.InitFlags(nil)
	// flag.Set("V", "2")
	// flag.Parse()

	parameterProcessing()
	_ = flag.CommandLine.Parse(nil)
	//klog.Infof("klog init: log event %d\n", tools.LogEvent)
	defer klog.Flush()
}

/*
parameterProcessing
Used to handle conflicts between klog framework and cobra framework
Enable the klog framework to correctly receive the parameters of -- v
*/
func parameterProcessing() {
	// 临时存储 os.Args
	args := os.Args[1:]
	remainingArgs := []string{os.Args[0]}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--v=") {
			vValue := strings.TrimPrefix(arg, "--v=")
			fmt.Printf("Handling --v=%s parameter\n", vValue)

			// Force setting the - v parameter of the klog framework
			if err := flag.Set("v", vValue); err != nil {
				fmt.Printf("Failed to set klog -v flag: %v\n", err)
			}
		} else {
			remainingArgs = append(remainingArgs, arg)
		}
	}

	// 重新设置 os.Args 为剩余参数，不包含 --v 参数
	os.Args = remainingArgs
}

/*
cobra相关内容
*/

// configFilePath 此变量用于接受--config参数的内容 然后传递到启动函数里
var configFilePath string

// rootCmd 主命令 也就是不加任何子命令情况 执行此函数
var rootCmd = cobra.Command{
	Use:   "config",
	Short: "input config file address.",
	Run: func(cmd *cobra.Command, args []string) {
		if checkConfigFile(configFilePath) {
			// 默认启动程序 也就是不加任何子命令 只指定--config参数
			NewStart(configFilePath)
		}

	},
}

// 增加一个新的子命令 version
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version.",
	Run: func(cmd *cobra.Command, args []string) {
		klog.Infoln("v1.0")
	},
}

// 增加一个新的子命令 init 需要指定参数 --config 这里是他的启动方法
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "used for initializing the environment for the first time.",
	Run: func(cmd *cobra.Command, args []string) {
		if checkConfigFile(configFilePath) {
			initEnvironment(configFilePath)
		}
	},
}

// init cobra框架 将所有的都添加到rootCmd这个主命令下
func init() {
	rootCmd.PersistentFlags().StringVar(&configFilePath, "config", "", "config file path.")
	rootCmd.AddCommand(versionCmd)
	// 添加一个命令 init 需要指定参数 --config
	initCmd.PersistentFlags().StringVar(&configFilePath, "config", "", "config file path.")
	rootCmd.AddCommand(initCmd)
}

func checkConfigFile(configFilePath string) bool {
	if configFilePath == "" {
		// fmt.Println("please input --config!")
		klog.Fatalln("please input --config!")
		return false
	}
	// fmt.Println("start!Use config file is :", configFilePath)
	klog.V(2).Info("start!Use config file is :", configFilePath)
	return true
}

// Start Cobra's startup function
func Start() {

	if err := rootCmd.Execute(); err != nil {
		klog.Fatalln("start error! please check databases config!")
	}
}

/*
New Start
*/

// NewStart default command execute
func NewStart(configFilePath string) {
	config := readConfig(configFilePath)
	klog.V(3).Infof("config: %+v\n", config)
	// 获取数据库连接
	newDatabase := NewDatabase(config.Database.DataBaseType)
	if newDatabase == nil {
		klog.Fatal("newDatabase Not initialized correctly, is nil!")
	}
	newDatabase.Init(config)
	newDatabase.EnsureUserLoginTable()
	newDatabase.EnsureButtonsTable()
	newDatabase.EnsureTagsTables()
	newDatabase.EnsureDownloadSaveTable()
	newDatabase.EnsureAskTable()
	if config.Login.User.Username != "" {
		_ = newDatabase.UpsertUser(config.Login.User.Username, config.Login.User.Password)
	}

	// start gin server
	startGinServer(int(config.Port), newDatabase)

}

func readConfig(configFilePath string) tools.ServiceConfig {
	var config tools.ServiceConfig
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		klog.Fatalf("Error reading YAML file: %s\n", err)
	}

	// Parse YAML file content to structure
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		klog.Fatalf("Error parsing YAML file: %s\n", err)
	}

	return config
}

// NewDatabase return databases interface
func NewDatabase(databaseType string) databases.Databases {
	return databases.NewDatabases(databaseType)
}

/*
startGinServer
Add the new interface address that needs to be processed here.
The corresponding method is implemented in the server.go file.
*/
func startGinServer(port int, database databases.Databases) {
	var route *gin.Engine
	route = gin.Default()
	route.Use(TrackMetrics())

	// TODO demo: binding interface
	route.GET("/api/status/information", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "normal",
		})
	})

	// TODO demo: add host
	route.POST("/api/host/add", func(c *gin.Context) {
		Test(c, database)
	})
	route.POST("/api/login", func(c *gin.Context) {
		Login(c, database)
	})
	route.GET("/api/buttons", AuthRequiredAPI(), func(c *gin.Context) {
		ButtonsList(c, database)
	})
	route.POST("/api/buttons", AuthRequiredAPI(), func(c *gin.Context) {
		ButtonsAdd(c, database)
	})
	route.DELETE("/api/buttons/:id", AuthRequiredAPI(), func(c *gin.Context) {
		ButtonsDelete(c, database)
	})
	route.GET("/api/dashboard", AuthRequiredAPI(), func(c *gin.Context) {
		Dashboard(c, database)
	})
	route.POST("/api/local/list", AuthRequiredAPI(), func(c *gin.Context) {
		LocalList(c, database)
	})
	route.POST("/api/local/index", AuthRequiredAPI(), func(c *gin.Context) {
		LocalIndex(c, database)
	})
	route.GET("/api/local/file", AuthRequiredAPI(), func(c *gin.Context) {
		LocalFile(c, database)
	})
	route.GET("/api/tags/search", AuthRequiredAPI(), func(c *gin.Context) {
		TagsSearch(c, database)
	})
	route.POST("/api/tags/add", AuthRequiredAPI(), func(c *gin.Context) {
		TagsAdd(c, database)
	})
	route.GET("/api/tags/all", AuthRequiredAPI(), func(c *gin.Context) {
		TagsAll(c, database)
	})
	route.POST("/api/favorite", AuthRequiredAPI(), func(c *gin.Context) {
		FavoriteSave(c, database)
	})
	route.GET("/api/favorites", AuthRequiredAPI(), func(c *gin.Context) {
		FavoritesList(c, database)
	})
	route.GET("/api/indexes", AuthRequiredAPI(), func(c *gin.Context) {
		IndexesList(c, database)
	})
	route.GET("/api/indexes/files", AuthRequiredAPI(), func(c *gin.Context) {
		IndexFiles(c, database)
	})
	route.GET("/api/indexes/info", AuthRequiredAPI(), func(c *gin.Context) {
		IndexInfo(c, database)
	})
	route.GET("/api/indexes/search", AuthRequiredAPI(), func(c *gin.Context) {
		IndexSearch(c, database)
	})
	route.GET("/api/server/duplicate", func(c *gin.Context) {
		ServerDuplicate(c, database)
	})
	route.GET("/api/server/save", func(c *gin.Context) {
		ServerSave(c, database)
	})
	route.GET("/api/ask/list", AuthRequiredAPI(), func(c *gin.Context) {
		AskList(c, database)
	})
	route.POST("/api/ask/create", AuthRequiredAPI(), func(c *gin.Context) {
		AskCreate(c, database)
	})
	route.DELETE("/api/ask/:id", AuthRequiredAPI(), func(c *gin.Context) {
		AskDelete(c, database)
	})

	klog.V(1).Infof("start gin server on port %d", port)
	_ = route.Run(fmt.Sprintf(":%d", port))
}
