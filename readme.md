# golang project template

一些基础的golang代码.

包括了 cobra klog 数据库interface,可以正确的接受参数并且处理,接下来只需要在对应位置填充代码逻辑即可.

## 文件及功能描述

databases/database.go: 存储了基础的数据库需要实现的interface.

databases/mongodb.go: 此文件是用来实现链接数据库的位置,可以实现查询等.不同的数据库使用不同名称的文件,统一实现databases/database.go文件的Databases interface.

server/start.go: 此文件代码处理了cobra klog等框架的冲突,进行了一些检查,如配置文件是否存在等,注册了一些cobra的基础命令,是程序的启动入口.

server/server.go: 此文件主要是处理gin框架注册的http或者https的接口的代码.

tools/tools.go: 此文件主要定义了一些全局变量
