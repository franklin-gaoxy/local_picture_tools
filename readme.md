# local_picture_tools

一个用于本地图片与索引管理的 Go 项目。后端使用 Gin + Cobra + klog，数据库以 MySQL 为主（自动建库/建表），同时保留 MongoDB 的占位实现。前端为静态页面，后端提供 REST API。

## 特性
- 配置化启动（YAML）
- 自动创建数据库与核心表
- 本地索引（local_index_* 动态表）
- 标签、收藏、下载记录与鉴权键管理
- 前后端分离，HTTP API 清晰

## 目录结构
- main.go：程序入口
- server/start.go：启动流程、命令行与路由注册
- server/server.go：HTTP API 实现
- databases/mysql.go：MySQL 连接、建库建表与数据接口
- databases/mongodb.go：MongoDB 占位实现
- databases/database.go：数据库接口定义
- tools/tools.go：配置结构体与日志初始化
- config.yaml：示例配置
- frontend/：前端静态页面

## 快速开始
1. 安装 Go（推荐 1.23.4）
2. 准备 MySQL，并确保账号有建库/建表权限
3. 根据示例配置文件填写数据库连接
4. 构建并运行

```bash
# 构建
go build -o bwrs .

# 运行（必须使用绝对路径指向配置文件）
./bwrs config --config /绝对路径/到/config.yaml
```

默认配置中，服务绑定的端口由 Gin 默认决定；API 路由见下文。

## 配置说明（config.yaml）
```yaml
databaseType: mysql
host: 127.0.0.1
port: 31756
path: local_picture_tools
description:
  username: root
  password: 12345678
```
- databaseType：mysql 或 mongodb（建议使用 mysql）
- host/port：数据库地址与端口
- path：作为数据库名（如 local_picture_tools）
- description.username/password：数据库凭证

启动后，程序会：
- 连接到 MySQL 服务器
- 自动创建数据库 `local_picture_tools`（若不存在）
- 以该数据库重新连接并创建核心表

## 数据库与表
自动创建的核心表包括（部分）：
- buttons：按钮信息
- userlogin：用户登录
- local_index_bindings：本地索引绑定
- local_index_*：本地索引数据（按名称动态创建）
- tags / favorites / dir_tag_map：标签与收藏及映射
- downloadsave：下载记录
- ask_keys：鉴权键

### 按钮信息表（buttons）
结构：
```sql
CREATE TABLE IF NOT EXISTS buttons (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  url VARCHAR(1024) NOT NULL,
  type VARCHAR(64) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```
接口：
- GET /api/buttons：列表
- POST /api/buttons：新增
- DELETE /api/buttons/:id：删除

## API 概览（部分）
- GET /api/dashboard：返回数据库元信息与所有表名
- 本地索引：
  - POST /api/local_index：创建索引表
  - POST /api/index_files：写入索引数据
  - POST /api/index_search：搜索索引
  - GET /api/index_info：索引信息
- 标签与收藏：
  - GET /api/tags_search / POST /api/tags_add / GET /api/tags_all
  - POST /api/favorite_save / GET /api/favorites_list
- 下载与鉴权：
  - POST /api/server_duplicate / POST /api/server_save
  - GET /api/ask_list / POST /api/ask_create / DELETE /api/ask_delete

## 元数据与数据导出
以下示例使用 `mysqldump` 导出 MySQL 元数据与 `buttons` 表完整数据。请按需调整主机、端口与凭证：

```bash
# 导出数据库元数据（仅结构，不含数据）
mysqldump --no-data \
  -h 127.0.0.1 -P 31756 -uroot -p12345678 \
  local_picture_tools > docs/mysql_metadata.sql

# 导出按钮信息表的完整数据（不含建表）
mysqldump --no-create-info \
  -h 127.0.0.1 -P 31756 -uroot -p12345678 \
  local_picture_tools buttons > docs/buttons_data.sql
```

建议将导出文件放在 `docs/` 目录下并纳入版本控制。

## GitHub Actions（后端自动编译）
本仓库提供工作流以在：
- 推送到 `main` 分支时自动构建
- 新发布（Release published）时编译并上传二进制制品

默认构建包含多个平台（linux/darwin/windows，amd64/arm64），产物以构建工件形式保存或附加到 Release。

## 开发与调试
- 使用 `klog` 输出日志，启动时会初始化日志目录与等级
- CLI 通过 `cobra` 管理命令与参数，必须传入 `--config`（绝对路径）
- 数据库接口在 `databases/database.go`，MySQL 实现在 `databases/mysql.go`

## 许可
根据仓库实际需求添加许可证（例如 MIT）。如未指定，默认为保留所有权利。
