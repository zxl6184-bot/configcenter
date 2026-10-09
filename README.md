# configcenter 开发工作区

> 本目录是一个 **开发管控工作区**，用于在 Claude Code 中管理 configcenter 服务与 SDK 的开发、构建和发布。本工作区本身不是代码仓库，代码存放在下方两个关联项目中。

---

## 1. 关联项目

| 项目                              | 本地路径                                                   | Git 远端                                                                          | Go Module                                                                     | 说明                                                 |
| --------------------------------- | ---------------------------------------------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | ---------------------------------------------------- |
| **configcenter（服务端）**  | `/home/wh8199/xiaolin.zhang/gopath/src/configcente`      | `https://Advantech-EBO@dev.azure.com/Advantech-EBO/IoTSuite/_git/configcenter`  | `configcenter`                                                              | 配置中心服务，负责服务注册与发现、数据库实例统一分配 |
| **configcenter-sdk（SDK）** | `/home/wh8199/xiaolin.zhang/gopath/src/configcenter-sdk` | `https://dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter` | `dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git` | Go SDK，供各微服务接入配置中心使用                   |

### 1.1 版本对应关系

| 维度     | configcenter（服务端）               | configcenter-sdk |
| -------- | ------------------------------------ | ---------------- |
| 当前版本 | `2.1.6-alpha1`（Makefile `TAG`） | `v0.17.0`      |
| 依赖关系 | 服务端 go.mod 引用 SDK`v0.17.0`    | —               |
| Go 版本  | 1.25.0                               | 1.18             |

> 版本号遵循 Semantic Versioning 2.0.0，管理细则见 `.claude/rules/版本管理规范.md`。

---

## 2. 环境准备

### 2.1 Git 凭据

配置 git 保存账密，克隆/推送时触发一次登录即可：

```bash
git config --global credential.helper store
```

### 2.2 Go 私仓访问

SDK 模块托管在 Azure DevOps 私仓，需要设置 `GOPRIVATE`：

```bash
go env -w GOPRIVATE=dev.azure.com
```

### 2.3 本工作区的 Claude Code 能力

- **规则（rules）**：`.claude/rules/` 下的规范会自动注入每次会话：
  - `git提交规范.md`：提交信息必须英文、遵循 Conventional Commits、一个提交只包含一个功能/修复
  - `多语言规范.md`：README/API 文档/思维链输出用中文，代码注释与 git 提交用英文
  - `版本管理规范.md`：SemVer 2.0.0，版本发布流程、CHANGELOG 维护
  - `知识库规范.md`：团队知识文档维护，`knowledge-doc-standards` skill 的触发入口
- **技能（skills）**：`speckit-*`（规范驱动开发工作流）、`knowledge-doc-standards`（知识库）、`session-handoff`（会话交接）、`weekly-report`（周报）等
- **命令（commands）**：`spec`、`superpowers`

---

## 3. configcenter 服务端（`/home/wh8199/xiaolin.zhang/gopath/src/configcente`）

### 3.1 功能简介

解决不同平台（k8s / docker / windows）数据库管理方式不一致、以及微服务地址分散在环境变量中的痛点，提供：

1. 服务注册与发现
2. 跨平台统一管理各服务数据库实例，支持：**Postgres、InfluxDB、MongoDB、RabbitMQ、Redis(Valkey)、Minio、NATS、IoTDB**
3. 支持 SDK 动态注册服务，也支持通过 `services` 配置文件静态注册

### 3.2 目录结构

| 路径                                            | 说明                                        |
| ----------------------------------------------- | ------------------------------------------- |
| `cmd/configcenter`                            | 服务主程序（`main.go` + `config.toml`） |
| `cmd/network`                                 | 网络相关工具                                |
| `pkg/config`                                  | 配置加载                                    |
| `pkg/configcenter`                            | 服务核心逻辑（注册、实例分配等）            |
| `pkg/dotconfig` / `pkg/utils`               | 工具库                                      |
| `charts/` `docker-compose/` `dockerfile/` | 各平台部署编排                              |
| `ip_monitor/`                                 | 主机 IP 监控                                |
| `static/`                                     | 静态资源                                    |

### 3.3 关键配置

- `config.toml`：服务端口（当前 `10087`）、各数据库连接信息、`globalIP`、`dataCenter`、`cluster`、`runningOnLocal` 等
- `services`（JSON）：静态服务列表，含 `routers` 与 `serviceInstances`
- `resourcegroups`（JSON）：资源组，同组服务可共享数据库连线信息
- `publicconfig`：公共配置
- `internal_external_ports.xlsx`：内外网端口映射表
- `.env`：本地数据库连接信息（开发用）

### 3.4 本地运行

```bash
cd D:\gopath\src\configcenter
go run ./cmd/configcenter
```

> 本地运行前需准备依赖数据库（见 `config.toml` / `.env`），并按需设置 `runningOnLocal = true` 以仅允许 127.0.0.1 访问。

### 3.5 构建

通过 Makefile 构建，镜像仓库与版本号在其中定义：

```bash
cd D:\gopath\src\configcenter
make          # 构建镜像，参考 build.mk
```

关键变量：`TAG=2.1.6-alpha1`、`LIB_VERSION=1.1.7.1-develop`、`SDK_VERSION=v0.17.0`。

---

## 4. configcenter SDK（`/home/wh8199/xiaolin.zhang/gopath/src/configcenter-sdk`）

### 4.1 功能简介

SDK 降低微服务接入配置中心的整合难度，提供：

- **Module（模块）**：代表一个服务，可申请服务实例（数据库）与注册 Router（路由）
- **服务实例类型**：Postgres / MongoDB / Redis / InfluxDB / RabbitMQ / Minio / NATS / Valkey / IoTDB
- **Router**：普通、Dashboard 数据源（`isDashboard`）、Desk（`isPortal`）、MCPGateway（`isMCPServer`）四种
- **订阅服务上下线事件**：`SubscribeEvent` / `UnsubscribeEvent`

### 4.2 目录结构

| 路径                                                   | 说明                                        |
| ------------------------------------------------------ | ------------------------------------------- |
| `pkg/configcenter`                                   | gRPC / protobuf 生成代码                    |
| `pkg/module`                                         | Module 核心逻辑（注册、连接、配置获取）     |
| `dsn` / `connectstring` / `dccs` / `portrange` | DSN 解析、连接串、端口范围等工具            |
| `protos/configcenter`                                | protobuf 定义（`make protobuf` 重新生成） |
| `examples/`                                          | 各场景示例（注册、资源组、订阅、MCP 等）    |
| `wiki/`                                              | 设计文档（如 Redis 连接模式与 DSN 约定）    |

### 4.3 在业务服务中引入 SDK

```bash
go get -u dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git@{branch_or_tag}
go mod tidy
```

> branch_or_tag 为空表示 master；`GOPRIVATE` 需包含 `dev.azure.com`。

---

## 5. 常用开发流程

### 5.1 提交流程

遵循 `.claude/rules/git提交规范.md`：

```
<type>(<scope>): <subject>

[optional body]
```

示例：

```
feat(servicebroker): add NATS service instance support

fix(resourcegroup): prevent nil pointer on missing resource
```

### 5.2 版本发布流程

遵循 `.claude/rules/版本管理规范.md`：

```
1. 更新 CHANGELOG.md（服务端为 version_configcenter.md）
2. 更新 go.mod（如有必要）
3. git tag vX.Y.Z
4. git push origin vX.Y.Z
5. 触发 CI Release 流水线
```

### 5.3 protobuf 重新生成（仅 SDK）

```bash
cd D:\gopath\src\configcenter-sdk
make prepare
make protobuf
```

---

## 6. 常用链接

- [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
- [Semantic Versioning 2.0.0](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)

---

## 7. 维护记录

| 日期       | 变更                                       |
| ---------- | ------------------------------------------ |
| 2026-08-18 | 初始化本工作区文档，梳理关联项目与环境准备 |
