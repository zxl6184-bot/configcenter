# 接口契约：NATS DSN 凭证参数

## 分发的 DSN 格式

配置中心 `BuildDSN` 返回的 NATS 连接串：

```text
nats://<host>:<port>?jwt=<url-escaped-std-base64-JWT>&seed=<url-escaped-std-base64-Seed>
```

| 部分 | 编码 |
|------|------|
| `<host>` / `<port>` | NATS 实例地址（来自 `dsn.ConnectInfo`，不变） |
| `jwt` | `url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(JWT)))` |
| `seed` | `url.QueryEscape(base64.StdEncoding.EncodeToString([]byte(Seed)))` |

其中 `JWT` / `Seed` 为凭证文件区块内的原始字符串（不含 `-----BEGIN/END-----` 标记行）。

等价实现：

```go
params := url.Values{
    "jwt":  {base64.StdEncoding.EncodeToString([]byte(creds.JWT))},
    "seed": {base64.StdEncoding.EncodeToString([]byte(creds.Seed))},
}
dsn := fmt.Sprintf("nats://%s:%d?%s", host, port, params.Encode())
```

## 消费端解码流程（SDK）

```text
dsn.ParseDSN(distributedDSN)
  ├─ 按 & 分段、按第一个 = 分隔 key/value
  ├─ url.QueryUnescape(key) / url.QueryUnescape(value)   # 解 URL 转义
  └─ Params["jwt"] / Params["seed"]

base64.StdEncoding.DecodeString(Params["jwt"])   # 还原原始 JWT 字符串
base64.StdEncoding.DecodeString(Params["seed"])  # 还原原始 Seed 字符串

nats.Connect("nats://<host>:<port>", nats.UserJWTAndSeed(string(jwt), string(seed)))
```

## 契约来源与定位

- 该契约已由 SDK 仓库 `sdk/configcenter/dsn/dsn_test.go` 的 `TestNatsCredentialParams` 固化：
  - `base64.StdEncoding` 产生 `+` `/` `=` 需经 URL 转义（`+/8=` → `%2B%2F8%3D`）。
  - 参数值内的 `=`（Base64 padding）合法，`strings.Cut(part, "=")` 只按首个 `=` 切分 key。
  - 消费端 `dsn.ParseDSN` 现依 `url.QueryUnescape` 还原。

## 关键校验点

- **地址不变**：`nats://<host>:<port>` 不含 userinfo、不含 database（凭证经 query 传递，不放入 URL 用户信息段）。
- **无凭证即报错**：凭证缺失/解析失败时，不让 NATS Broker 构造成功（FR-008），因此不存在「无参数的半成品 DSN」被分发的路径。