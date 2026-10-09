# 快速验证指南：NATS 用户凭证文件读取与分发

本指南说明如何端到端验证「配置中心读取 NATS 凭证文件并分发 JWT/Seed」功能，无需实现细节贴合代码。

## 前置条件

- Go 1.22+、Postgres（用于元数据，可复用 `configcenter_test`）。
- 一个标准 NATS `nsc` credentials 文件（含 `-----BEGIN NATS USER JWT-----` 与 `-----BEGIN USER NKEY SEED-----` 两区块）。

## 场景 1：凭证文件读取 + DSN 分发

1. 创建临时凭证文件：

```text
/tmp/creds/user.creds   （JWT 区块 + Seed 区块，内容为 nsc 生成的格式）
```

2. 在 `config.toml`（或 flag）中新增：

```text
[configCenter]
nats = "nats://127.0.0.1:4222"
natsCredsPath = "/tmp/creds"
natsCredsName = "user.creds"
```

3. 启动服务，注册一个 NATS 类型实例：

```text
cd cmd/configcenter && go run .
```

4. 查询返回的 NATS 连接串：

```text
期望形式：nats://127.0.0.1:4222?jwt=<url转义Base64>&seed=<url转义Base64>
```

- 地址部分保持原样。
- `jwt`/`seed` 值 decode（先 URL 解码、再 Base64 Std 解码）后应还原凭证文件中的 JWT/Seed 原始内容。

## 场景 2：凭证缺失 → 拒绝

1. 指向一个不存在的路径（或删除文件）。
2. 启动/注册 NATS 实例。
3. **期望**：broker 构造或注册失败，返回明确错误；不产生仅含地址、无凭证的连接串。

## 场景 3：SDK 消费方端到端

1. 在 `sdk/configcenter` 编译运行示例：

```text
cd sdk/configcenter/examples/nats && go run main.go <distributedDSN>
```

2. 期望日志显示：成功解析地址、Base64 解码 jwt/seed、经 `nats.UserJWTAndSeed` 建立连接。

## 测试命令

服务端：

```text
make test            # 运行 nats 等全部单测
```

SDK 侧：

```text
cd sdk/configcenter && go test ./dsn -run TestNatsCredentialParams
```

## 预期结果

- 凭证文件就绪时，NATS 实例注册返回含 jwt/seed 参数的 DSN。
- 凭证缺失/异常时，注册被拒绝并报错。
- 三种部署（本地普通文件 / Docker volume / K8s Secret 挂载）仅路径来源不同，读取逻辑一致。