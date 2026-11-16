# 公网地址后缀模式与 Go SDK 接入指南

适用对象：接入配置中心的第三方 Go 服务开发者、部署人员。版本核对日期：2026-11-16。

## 1. 后缀模式说明

后缀模式下，配置中心用统一入口加上服务注册的 URI 前缀生成公网地址：

- `Router.UriPrefix`：服务的公网入口路径，例如 `/my-service/`。
- `Router.PublicAddress`：配置中心生成的公网地址，例如 `https://gateway.example.com/my-service/`。

`isSuffixMode`、`isSSL`、`suffixPort` 是配置中心服务端的配置项，只在服务端启用后缀模式（`isSuffixMode=true`）时生效。第三方服务只需注册 `UriPrefix` 并读取 `PublicAddress`，不需要读取或设置这些配置。

## 2. 安装 SDK

SDK `v0.18.0` 只新增了 `Router.UriPrefix` 字段；`isSuffixMode`、`isSSL`、`suffixPort` 不在 SDK 中，是配置中心服务端的配置。需要 Go 1.22 或更高版本，并具备 Azure DevOps 私有仓库的 Git 访问权限。

```bash
# 已配置其他 GOPRIVATE 规则的环境，请将 dev.azure.com 追加到原有规则中。
go env -w GOPRIVATE=dev.azure.com

go get dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git@v0.18.0
```

## 3. 接入示例

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
                continue
            }
            log.Printf("publicAddress=%s", publicAddress)
        }
    }
}
```

配置中心内部 gRPC 地址通过 `CONFIG_CENTER_ADDRESS` 设置，格式为 `host:port`：

```bash
export CONFIG_CENTER_ADDRESS=192.168.55.1:10087
```

`Address` 填写服务内部地址，`UriPrefix` 填写公网入口路径。公网地址由配置中心生成，调用方读取 `PublicAddress`，不要自行拼接。

## 4. 配置中心服务端配置

以下配置由部署方在配置中心服务端设置，不写入第三方服务的 `ModuleConfig`：

| 配置                   | 类型/默认值     | 说明                                                                            |
| ---------------------- | --------------- | ------------------------------------------------------------------------------- |
| `isSuffixMode`       | bool，`false` | 是否启用后缀模式                                                                |
| `isSSL`              | bool，`true`  | `false` 为 HTTP，`true` 为 HTTPS                                            |
| `suffixPort`         | int，`0`      | `0` 或省略表示协议默认端口；HTTP 80、HTTPS 443。1–65535 表示实际公网入口端口 |
| `domain`             | string          | 后缀模式的入口域名，不带协议、端口或路径，例如 `gateway.example.com`          |
| `globalIP`           | string          | `domain` 为空时使用的入口 IP                                                  |
| `autoGeneratePublic` | bool            | 设为 `true` 才会自动生成公网地址                                              |

```toml
[configCenter]
isSuffixMode = true
isSSL = true
suffixPort = 0
domain = "gateway.example.com"
autoGeneratePublic = true
```

端口说明：

| `isSSL` | `suffixPort` | 生成的公网地址                           |
| --------- | -------------- | ---------------------------------------- |
| `false` | `0`          | `http://192.168.55.1/my-service/`      |
| `true`  | `0`          | `https://192.168.55.1/my-service/`     |
| `false` | `8080`       | `http://192.168.55.1:8080/my-service/` |
| `true`  | `433`        | `https://192.168.55.1:433/my-service/` |

HTTP 80、HTTPS 443 会在 URL 中省略。`suffixPort` 是公网入口端口，不是服务内部监听端口，也不是 SDK 连接配置中心所用的端口。

## 5. 路径兜底

后缀模式下路径按以下顺序获取：

1. 使用 Router 注册的 `UriPrefix`。
2. 未提供时，按服务名和 Router 名从配置中心维护的端口 Excel 第六列 `uriprefix` 读取。
3. 两处都为空时，`PublicAddress` 返回空。

URI 路径会补齐前导 `/`、折叠重复 `/`，并保留显式尾部 `/`。
