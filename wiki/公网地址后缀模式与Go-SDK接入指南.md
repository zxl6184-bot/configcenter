# 公网地址后缀模式与 Go SDK 接入指南

适用对象：接入配置中心的第三方 Go 服务开发者、前端集成人员和部署人员。版本核对日期：2026-11-16。

## 1. 功能说明

配置中心支持两种公网地址模式：

| 模式     | 配置                           | 地址规则                                                                                         |
| -------- | ------------------------------ | ------------------------------------------------------------------------------------------------ |
| 前缀模式 | `isSuffixMode=false`，默认值 | 沿用原有地址生成规则；K8s 使用 Ingress 域名，Docker/二进制沿用机器 IP、服务端口及端口映射规则    |
| 后缀模式 | `isSuffixMode=true`          | 使用统一入口域名或 IP，加上服务注册的 URI 路径，例如 `https://gateway.example.com/my-service/` |

后缀模式将服务的**内部连接地址**与**公网访问地址**分开：

- `Router.Address`：内部服务地址，例如 `http://my-service:8080/private/backend`。
- `Router.UriPrefix`：公网入口路径，例如 `/my-service/`。
- `Router.PublicAddress`：配置中心生成的公网地址，例如 `https://gateway.example.com/my-service/`。

后缀模式不会把 `Address` 中的协议、端口或路径附加到公网地址。调用方应读取配置中心返回的 `PublicAddress`，避免自行拼接。

## 2. 使用哪个 Tag

SDK 仓库：[configcenter SDK（Azure DevOps）](https://dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter)。该仓库名称为 configcenter，与服务端仓库分属不同项目。

| 项目/版本                | 当前状态                                                             | 使用说明                                                                                   |
| ------------------------ | -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| SDK `v0.18.0`          | 已发布，本文按该 Tag 核对                                             | 新增 `Router.UriPrefix` 字段，用于声明服务在统一入口上的公网路径                          |
| SDK `v0.17.0` 及更早版本 | 不包含 `Router.UriPrefix` 字段                                       | 本功能不能按这些 Tag 接入                                                                  |
| 服务端 `v2.1.7`        | 与本文配置项配套的服务端版本，尚未发布                                 | 部署方须提供包含单端口 `suffixPort` 的构建                                                |

SDK 已发布 `v0.18.0`，第三方可直接按该 Tag 接入。接入时只需在 `Router` 中填写 `UriPrefix`，公网地址由配置中心根据服务端配置生成。`isSuffixMode`、`isSSL`、`suffixPort` 是配置中心服务端的配置项，由部署方维护，第三方服务不需要读取或设置。

### 2.1 安装 SDK

需要 Go 1.22 或更高版本，并已具备上述 Azure DevOps 私有仓库的 Git 访问权限。

```bash
# 已配置其他 GOPRIVATE 规则的环境，请将 dev.azure.com 追加到原有规则中。
go env -w GOPRIVATE=dev.azure.com

go get dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git@v0.18.0
```

加入业务代码后执行 `go mod tidy`，提交业务项目的 `go.mod` 和 `go.sum`。依赖固定在该 Tag 上，后续版本发布不会自动升级已锁定的依赖。

## 3. 部署方配置

本节配置写在**配置中心服务端**，不写入第三方服务的 `ModuleConfig`。

| 服务端配置             | 类型/默认值     | 说明                                                                            |
| ---------------------- | --------------- | ------------------------------------------------------------------------------- |
| `isSuffixMode`       | bool，`false` | 是否启用后缀模式                                                                |
| `isSSL`              | bool，`true`  | `false` 为 HTTP，`true` 为 HTTPS                                            |
| `suffixPort`         | int，`0`      | `0` 或省略表示协议默认端口；HTTP 80、HTTPS 443。1–65535 表示实际公网入口端口 |
| `domain`             | string          | 后缀模式的完整入口域名，不带协议、端口或路径，例如 `gateway.example.com`      |
| `globalIP`           | string          | `domain` 为空时使用的入口 IP；非 K8s 平台沿用服务端 IP 检测与监控机制         |
| `autoGeneratePublic` | bool            | 设为 `true` 才会自动生成公网地址                                              |

TOML 示例：

```toml
[configCenter]
isSuffixMode = true
isSSL = true
suffixPort = 0
domain = "gateway.example.com"
autoGeneratePublic = true
```

使用 IP 入口的 Docker/二进制部署，可以将 `domain` 设为空，并确认配置中心实际检测到的 `globalIP` 是客户端可达的入口 IP。现有 K8s 启动域名必填校验仍保留。

等价的命令行配置片段如下；其他数据库、配置文件路径、服务监听端口等启动参数按部署环境保留：

```text
--configCenter.isSuffixMode=true
--configCenter.isSSL=true
--configCenter.suffixPort=0
--configCenter.domain=gateway.example.com
--configCenter.autoGeneratePublic=true
```

当前目录存在 `config.toml` 时读取 TOML；不存在时读取命令行参数。不要依赖命令行覆盖已存在的 TOML。

### 3.1 端口示例

以下假设入口 IP 为 `192.168.55.1`，注册路径为 `/my-service/`：

| `isSSL` | `suffixPort` | 生成的公网地址                           |
| --------- | -------------- | ---------------------------------------- |
| `false` | `0`          | `http://192.168.55.1/my-service/`      |
| `true`  | `0`          | `https://192.168.55.1/my-service/`     |
| `false` | `8080`       | `http://192.168.55.1:8080/my-service/` |
| `true`  | `433`        | `https://192.168.55.1:433/my-service/` |
| `true`  | `443`        | `https://192.168.55.1/my-service/`     |

HTTP 80、HTTPS 443 会在 URL 中省略。`433` 是自定义端口，会按原值保留。

`suffixPort` 表示统一网关的公网入口端口，不是服务内部监听端口，也不是 SDK 连接配置中心所用的端口。

## 4. 第三方 Go 服务接入

### 4.1 设置配置中心连接地址

SDK 通过 `CONFIG_CENTER_ADDRESS` 指定配置中心的内部 gRPC 地址，格式为 `host:port`，不包含 `http://`、`https://` 或 URI 路径。请填写部署方提供的实际监听端口。

```bash
# 示例：配置中心内部 gRPC 地址。
export CONFIG_CENTER_ADDRESS=192.168.55.1:10087
```

Windows PowerShell：

```powershell
$env:CONFIG_CENTER_ADDRESS = "192.168.55.1:10087"
```

`isSSL` 是配置中心服务端的配置项，决定生成的公网 URL 使用什么协议；它不会改变 SDK 的 gRPC 连接方式。

### 4.2 注册路径并读取公网地址

以下示例基于 SDK `v0.18.0`。将 `Address` 换成配置中心及网关可访问的实际后端地址。

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "time"

    "dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git/pkg/configcenter"
    "dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git/pkg/module"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()

    m := module.NewModule(module.ModuleConfig{
        Name: "my-service",
        Routers: []*configcenter.Router{
            {
                Name:      "API",
                Address:   "http://my-service:8080/private/backend",
                UriPrefix: "/my-service/",
            },
        },
    }, ctx)

    // LinkConfigCenter runs the registration, keepalive, and discovery loop.
    go m.LinkConfigCenter()

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            publicAddress, found := m.GetRouterPublicAddress("my-service", "API")
            if !found || publicAddress == "" {
                log.Print("waiting for public address; check path and server configuration")
                continue
            }
            log.Printf("publicAddress=%s", publicAddress)
        }
    }
}
```

将代码保存为业务项目中的 `main.go`，执行 `go mod tidy`、`go build`，配置环境变量后运行。示例只注册 Router（含 `UriPrefix`），不申请数据库实例；`isSuffixMode`、`isSSL`、`suffixPort` 由配置中心服务端配置，业务代码不需要读取。

SDK 默认保活与配置刷新间隔为 5 秒；刚启动时，配置和服务缓存可能尚未就绪。应处理 Router 不存在及空公网地址，不能把创建 `Module` 视为已完成注册和发现。

`Router.UriPrefix` 是 SDK `v0.18.0` 新增的注册字段。旧版 SDK 注册的 Router 没有该字段，只能依赖配置中心按服务名和 Router 名匹配端口 Excel 中的 `uriprefix` 兜底。

### 4.3 SDK 新增字段与服务端配置

SDK 侧新增的接入字段只有一个：

| SDK/API 字段                      | 含义                                                       |
| --------------------------------- | ---------------------------------------------------------- |
| `configcenter.Router.UriPrefix` | Go 字段；protobuf 字段号为 14，JSON 字段名为 `uriPrefix` |

以下公共配置键由配置中心根据服务端配置生成，仅供部署方联调检查，不是 SDK 接入所需，第三方业务代码不需要读取：

| 公共配置键        | 含义                                                       |
| ----------------- | ---------------------------------------------------------- |
| `is_suffix_mode` | 字符串 `"false"` 或 `"true"`，对应服务端 `isSuffixMode` |
| `is_ssl`         | 字符串 `"false"` 或 `"true"`，对应服务端 `isSSL`        |
| `suffix_port`    | 字符串形式的实际入口端口，如 `"80"`、`"443"`、`"433"`，对应服务端 `suffixPort` |

服务端启动参数 `suffixPort=0` 表示使用默认值；公共接口返回的是**解析后的实际端口**，不会因此返回 `"0"`。

如已有静态配置或注册信息手动指定了非空 `PublicAddress`，配置中心保留该值，不用自动生成地址覆盖它。

## 5. 路径兜底与旧客户端兼容

后缀模式按以下顺序获取路径：

1. 优先使用 Router 注册的非空 `UriPrefix`。
2. 未提供时，按服务名和 Router 名精确匹配部署方维护的端口 Excel，读取固定第六列 `uriprefix`。
3. 两处均为空时，返回空 `PublicAddress`；注册和查询本身不会因为缺路径失败。

旧客户端可继续注册，但若要自行传递 `UriPrefix`，必须升级 SDK。URI 路径会补齐前导 `/`、折叠重复 `/`，并保留显式尾部 `/`。服务注册名称、Router 名称应与部署方维护的记录一致。

## 6. 联调确认

部署方可通过 REST 检查公共配置和服务结果。以下地址为示例，使用实际配置中心 HTTP 监听地址；它与公网网关入口地址可能不同。

```bash
curl http://192.168.55.1:10087/v1/public/configs
curl http://192.168.55.1:10087/v1/services
```

当 `isSuffixMode=true`、`isSSL=true`、`suffixPort=0` 时，公共配置中应包含：

```json
{
  "is_suffix_mode": "true",
  "is_ssl": "true",
  "suffix_port": "443"
}
```

以上仅为相关字段节选。接口沿用字符串映射格式。

若注册路径为 `/my-service/` 且完整入口域名为 `gateway.example.com`，服务查询中对应 Router 的 `publicAddress` 应为 `https://gateway.example.com/my-service/`。

## 7. 网关部署要求与常见问题

配置中心负责生成和分发地址。部署方仍需配置 DNS、网关的路径转发、后端连接及 HTTPS 证书。生成了 `PublicAddress` 不代表公网地址已经可访问。

| 现象                                               | 检查项                                                                          |
| -------------------------------------------------- | ------------------------------------------------------------------------------- |
| Go 编译提示 `unknown field UriPrefix`            | SDK 版本过旧；升级到 `v0.18.0`                                                  |
| 公共配置中没有 `suffix_port`，却有两个旧端口字段 | 服务端仍是旧构建；升级到包含单端口配置的构建                                    |
| 返回的公网地址为空                                 | 检查 `autoGeneratePublic`、注册路径、Excel 兜底路径；刚启动时还需等待缓存刷新 |
| 地址仍沿用原有前缀规则                             | 检查 `isSuffixMode` 是否为 `true`，以及实际读取的 TOML/启动参数             |
| 配置的 IP 与返回 IP 不一致                         | 非 K8s 使用现有 IP 检测与监控机制，向部署方核对实际入口 IP 来源                 |
| 地址正确但访问失败或 404                           | 检查 DNS、证书、入口端口和网关路径转发，以及服务是否支持该前端路径              |

SDK 已发布 `v0.18.0`；正式对外交付时，提供方应补充包含单端口配置的服务端镜像版本，并确认第三方有该私有仓库的读取权限。
