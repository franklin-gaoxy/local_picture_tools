package server

import (
	"bwrs/databases"
	"bwrs/tools"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"strings"
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

	// TODO demo: binding interface
	route.GET("/status/information", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "normal",
		})
	})

	// TODO demo: add host
	route.POST("/host/add", func(c *gin.Context) {
		Test(c, database)
	})

	klog.V(1).Infof("start gin server on port %d", port)
	_ = route.Run(fmt.Sprintf(":%d", port))
}
