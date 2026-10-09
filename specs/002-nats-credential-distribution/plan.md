# 实施计划：NATS 用户凭证（JWT/Seed）文件读取与分发

**分支**: `feat/nats-credential-distribution` | **日期**: 2026-09-21 | **规格**: [spec.md](./spec.md)

**输入**: `/specs/002-nats-credential-distribution/spec.md` 功能规格

## 摘要

为 NATS Broker 增加用户凭证（JWT/Seed）文件读取与分发能力：

1. 新增配置项 `natsCredsPath`（目录，默认 `.`）+ `natsCredsName`（文件名，默认 `user.creds`），沿用 public-config 的「目录 + 文件名」双配置项模式，读取方式为 `os.ReadFile(path.Join(Path, Name))`。
2. 配置中心启动时读取凭证文件，提取 JWT 与 Seed 两个内容区块，各做 `base64.StdEncoding` 编码 + URL 转义后缓存于 broker。
3. `BuildDSN` 在 `nats://<host>:<port>` 基础上追加 `?jwt=<转义后 Base64>&seed=<转义后 Base64>` 参数，地址部分不变。
4. 凭证文件缺失/解析失败时拒绝服务注册并返回错误（FR-008）。
5. SDK 仓库新增可运行的 NATS 连接示例与说明，演示 Base64 解码后经 `nats.UserJWTAndSeed` 建立客户端连接。

## 技术上下文

**语言/版本**: Go 1.22+

**主要依赖**: 服务端无新增依赖（凭证解析使用标准库 `os`/`path`/`strings`/`encoding/base64`/`net/url`）；SDK 示例新增 `github.com/nats-io/nats.go`（仅示例分发包用）

**存储**: Postgres（GORM，现有方式）；凭证为全局内存态，不新增持久化实体

**测试**: `go test`（`make test`）；SDK 侧 `go test ./...`

**目标平台**: Linux 服务器（Docker + K8s）

**项目类型**: 微服务（gRPC + REST API 后端）

**性能目标**: 凭证仅在 broker 构造时读取一次，注册请求无额外文件 IO

**约束**:
- 遵循 ServiceBroker 插件注册模式（原则 II），不修改 ServiceBroker 接口签名与核心 ConfigCenter 逻辑
- 凭证全局一份（所有服务共享同一 NATS 用户凭证），因 `BuildDSN` 无错误返回通道，凭证加载失败须在 broker 构造阶段（可返回 error）暴露

**规模/范围**: 1 个 broker 包修改（nats）+ 配置项与 flag keys 新增 + SDK 示例/文档 + 测试

## 宪法检查

*门禁：必须在 Phase 0 研究前通过。Phase 1 设计后重新检查。*

| 原则 | 状态 | 说明 |
|------|------|------|
| I. Go 标准项目布局 | ✅ 通过 | 改动局限于 `pkg/configcenter/servicebroker/nats/` 与 `pkg/config/` |
| II. 插件化 Broker 架构 | ✅ 通过 | 扩展既有 NatsServiceBroker，不新增 broker 注册路径、不改核心逻辑 |
| III. 配置管理 | ✅ 通过 | 配置项并入 `pkg/config` 全局单例，读写方式与 public-config 一致 |
| IV. 多语言规范 | ✅ 通过 | 代码注释英文，文档/规格中文，提交英文 |
| V. 语义化版本 | ✅ 通过 | 新增向下兼容功能 → MINOR 递增；SDK 需发布新版本 |
| VI. 服务注册与发现 | ✅ 通过 | 复用 gRPC/REST/h2c 既有出口，仅 NATS DSN 内容变化 |
| VII. 测试与端到端验证 | ✅ 通过 | 新增 nats 单测；SDK dsn 契约测试；集成测试适配 |

**Phase 1 后重检**: 无违规项，无需填写复杂度追踪表。

## 项目结构

### 文档（本功能）

```text
specs/002-nats-credential-distribution/
├── plan.md              # 本文件
├── research.md          # Phase 0 研究输出
├── data-model.md        # Phase 1 数据模型
├── quickstart.md        # Phase 1 验证指南
├── contracts/
│   └── dsn-contracts.md # Phase 1 接口契约（NATS DSN 凭证参数）
└── tasks.md             # Phase 2 任务分解（/speckit-tasks 生成）
```

### 源代码（仓库根）

```text
pkg/config/config.go                           # [修改] 新增 NatsCredsPath/NatsCredsName 与 flag keys
pkg/configcenter/servicebroker/nats/
├── nats.go                                    # [修改] 读取凭证、缓存、BuildDSN 追加 jwt/seed 参数
├── nats_test.go                               # [修改] 凭证读取与 DSN 参数单测
└── nats_integration_test.go                   # [修改] 适配新 DSN 契约

sdk/configcenter/                              # [独立仓库] SDK
├── go.mod                                     # [修改] 新增 github.com/nats-io/nats.go
├── examples/nats/
│   ├── main.go                                # [新增] 可运行消费方示例（Base64 解码 + UserJWTAndSeed）
│   └── README.md                              # [修改] 补充 Base64 解码与建连说明
└── dsn/dsn.go                                 # [已有] 已含 URL 转义 + base64 padding 解析修复
```

**结构决策**: 遵循既有单项目结构；服务端改动集中在一个 broker 子包与配置包，SDK 改动在其独立仓库中单独提交。

## 复杂度追踪

> 无宪法违规项，无需填写。