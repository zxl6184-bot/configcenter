# 研究：NATS 用户凭证文件读取与分发

## 研究结论汇总

| 议题 | 决策 | 理由 |
|------|------|------|
| 凭证文件解析方式 | 标准库手工解析（`strings`/`regexp`） | 配置中心只分发不校验；避免引入 nkeys 依赖；契合项目 `dsn` 手工解析惯例 |
| Base64 与 URL 编码契约 | `base64.StdEncoding` + URL 转义（`url.QueryEscape`） | SDK 既有 `dsn` 契约测试已固化此格式 |
| 凭证读取时机 | broker 构造时读取（启动期） | `BuildDSN` 无错误返回通道，且凭证全局一份 |
| 配置键设计 | `NatsCredsPath` + `NatsCredsName` | 与 public-config 的「目录 + 文件名」模式一致 |
| SDK 依赖 | 新增 `github.com/nats-io/nats.go` | 示例 code 需 `nats.Connect` + `nats.UserJWTAndSeed` |
| 凭证缺失行为 | 拒绝注册/启动失败并报错 | FR-008 明确要求 |

---

## 1. 凭证文件解析方式

**Decision**: 使用 Go 标准库手工解析凭证文件，提取 JWT 与 Seed 两段内容，不引入第三方依赖。

**Rationale**:
- 规格明确「配置中心只做分发」，无需用 nkeys 库解密/校验证书——只需按标记行提取文本。
- 项目既有 `dsn`（v0.17.0）即为手工字符串解析，保持同一风格。
- 免新增 `github.com/nats-io/nkeys` 依赖（其引入 `x/crypto` 等传递依赖）。

**格式约定**（标准 NATS `nsc` credentials 文件）：

```text
-----BEGIN NATS USER JWT-----
<JWT，单行>
------END NATS USER JWT------

（重要提示文本，可忽略）

-----BEGIN USER NKEY SEED-----
<Seed，单行>
------END USER NKEY SEED------
```

提取逻辑：以 `-----BEGIN NATS USER JWT-----` 到 `------END NATS USER JWT------` 之间、以及 `-----BEGIN USER NKEY SEED-----` 到 `------END USER NKEY SEED------` 之间的非空行作为 JWT/Seed 原始内容。

**Alternatives considered**: `github.com/nats-io/nkeys` 的 `ParseDecoratedJWT`/`ParseDecoratedNKey` —— 解析更健壮但有额外依赖且超出「仅分发」需求，故弃用。

---

## 2. Base64 与 URL 编码契约

**Decision**: JWT/Seed 各做 `base64.StdEncoding.EncodeToString` → 再 `url.QueryEscape`（等价于 `url.Values.Encode()`）后拼入 DSN query。

**Rationale**: SDK 仓库 `sdk/configcenter/dsn/dsn_test.go` 的 `TestNatsCredentialParams` 已固化契约：
- 服务端编码用 `base64.StdEncoding`（含 `+` `/` `=`）。
- query 值需 URL 转义（`+/8=` → `%2B%2F8%3D`）。
- 消费端 `dsn.ParseDSN` 会 `url.QueryUnescape`，再 `base64.StdEncoding.DecodeString`。

**Alternatives considered**: `base64.RawURLEncoding`（无 `=`、用 `-`/`_`，可直接入 query）——但 SDK 消费端已固定用 `StdEncoding.DecodeString`，改用 RawURLEncoding 会与现有 SDK 文档/测试冲突，故弃用。

---

## 3. 凭证读取时机与失败处理

**Decision**: 在 `NewNatsServiceBroker`（broker 构造，发生于 ConfigCenter 启动过程）读取并解析凭证；失败返回 error，向上传播导致启动失败。

**Rationale**:
- `ServiceBroker.BuildDSN` 返回 `string` 无错误通道；而共享实例复用路径（`ServiceBrokers.NewInstance` 中 `GetSharedInstanceInDatabase` 命中时）会直接调用 `BuildDSN`，无法在调用点报错。
- 因此凭证必须一次性加载进 broker 内存，`BuildDSN` 直接使用缓存；加载失败只能在构造阶段（能返回 error）暴露。
- NATS 凭证代表「所有服务共用的一个 NATS 用户」，全局一份，语义上天然属于 broker 级状态。
- K8s Secret / Docker volume 在容器启动前即挂载，启动期读取满足部署场景。

**Alternatives considered**: 在 `NewInstance` 惰性读取——但共享实例路径绕开 `NewInstance`，会导致凭证缺失，且 `BuildDSN` 无法报错，故弃用。

---

## 4. 配置键设计与默认值

**Decision**: 在 `ConfigCenterConfig` 新增：

```text
NatsCredsPath string   // default "."（目录）
NatsCredsName string   // default "user.creds"（文件名）
```

读取用 `os.ReadFile(path.Join(NatsCredsPath, NatsCredsName))`，并在 `config.Init()` 的 flag keys 列表同步加入 `configCenter.natsCredsPath` / `configCenter.natsCredsName`。

**Rationale**: 用户确认「参考 public-config 的读取方式」；`PublicConfigPath`（默认 `.`）+ `PublicConfigName`（默认 `publicconfig`）是既有双配置项范式，NATS 凭证沿用同构。K8s Secret 卷挂载到目录后以对应参数指定；Docker volume 与二进制普通文件挂载同理。

**Alternatives considered**: 单一绝对路径配置（`natsCredsFile`）——无法天然区分目录与文件名，与既有 public-config 范式不一致，故弃用。

---

## 5. SDK 侧依赖与示例

**Decision**: 在 `sdk/configcenter/go.mod` 新增 `github.com/nats-io/nats.go`（建议 v1.x 稳定版），新增 `examples/nats/main.go` 可运行示例。

**Rationale**: 示例需调用 `nats.Connect(serverURL, nats.UserJWTAndSeed(jwt, seed))`，该符号只存在于 nats.go。SDK 的 `dsn` 解析与 base64 解码契约已在 `sdk/configcenter/dsn/*` 中完成（相对于已发布 v0.17.0 是增量修复，需随本次功能发布新 SDK 版本）。

**Alternatives considered**: 在 README 内联代码而不提供可编译示例——不满足 FR-009「提供示例」，故提供可运行的 `main.go`。

---

## 6. 兼容性与影响面

- 服务端 `dsn` 包（v0.17.0）无需改动：服务端只「构建」不「解析」输出 DSN。
- `ServiceInstance` 表结构不变：凭证为全局内存态，不入库、不写入 `Parameters`。
- 已有 `TestBuildDSN` 等单测/集成测试需适配：默认/无凭证场景在拒绝与携带凭证间需重新对齐（见 quickstart）。