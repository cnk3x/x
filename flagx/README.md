# flagx

`flagx` 是一个基于 Go 语言开发的命令行参数解析扩展工具，旨在增强标准库 `flag` 的功能，提供更灵活、强大的 CLI 参数处理能力。它基于 `github.com/spf13/pflag` 库构建，提供了许多高级功能。

## 特性

- **POSIX 风格的命令行参数**: 支持长选项（如 `--long`）和短选项（如 `-s`）
- **多数据类型支持**: 自动绑定和解析多种数据类型（string, int, bool, time.Duration, net.IP 等）
- **结构体标签**: 通过结构体标签快速定义命令行参数
- **环境变量支持**: 支持从环境变量获取参数值
- **配置文件支持**: 可以从配置文件加载参数
- **键值对参数**: 支持 `key:value` 形式的参数对
- **类型安全**: 使用 Go 泛型确保类型安全

## 安装

```bash
go get github.com/cnk3x/flagx
```

## 使用示例

### 基本用法

```go
package main

import (
    "fmt"
    "github.com/cnk3x/flagx"
)

func main() {
    var (
        verbose = false
        port    = 8080
        host    = "localhost"
    )

    // 使用 Var 函数定义不同类型的参数
    flagx.Var(&verbose, "verbose", "v", "Enable verbose logging")
    flagx.Var(&port, "port", "p", "Port to listen on")
    flagx.Var(&host, "host", "H", "Host to listen on")

    flagx.Parse()

    fmt.Printf("Verbose: %t, Host: %s, Port: %d\n", verbose, host, port)
}
```

### 结构体用法

```go
package main

import (
    "fmt"
    "github.com/cnk3x/flagx"
)

type Config struct {
    Verbose bool   `flag:"verbose,v" usage:"Enable verbose logging"`
    Port    int    `flag:"port,p" usage:"Port to listen on"`
    Host    string `flag:"host,H" usage:"Host to listen on"`
}

func main() {
    var config Config

    flagx.Struct(&config)
    flagx.Parse()

    fmt.Printf("Config: %+v\n", config)
}
```

### 文件参数

```go
package main

import (
    "fmt"
    "github.com/cnk3x/flagx"
)

type MyConfig struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func main() {
    var config MyConfig

    flagx.File(&config, "config", "c", "", "Configuration file path")
    flagx.Parse()

    fmt.Printf("Config: %+v\n", config)
}
```

### 键值对参数

```go
package main

import (
    "fmt"
    "github.com/cnk3x/flagx"
)

func main() {
    var headers [][2]string

    flagx.Pairs(&headers, "header", "H", nil, "HTTP headers in key:value format")
    flagx.Parse()

    for _, h := range headers {
        fmt.Printf("Header: %s=%s\n", h[0], h[1])
    }
}
```

## API

### 基本函数

- `Var[T flagType](val *T, name, shorthand, usage string, env ...string)` - 添加命令行标志
- `VarSet[T flagType](fset *FlagSet, val *T, name, shorthand, usage string, env ...string)` - 向特定 FlagSet 添加标志
- `Struct[T any](pStruct *T)` - 从结构体定义命令行标志
- `StructSet[T any](fset *FlagSet, pStruct *T, errExit bool)` - 从结构体向特定 FlagSet 添加标志
- `Parse()` - 解析命令行参数
- `Usage()` - 显示帮助信息
- `Name()` - 获取应用程序名称

### 文件相关函数

- `File[T any](v *T, name, short, def, usage string)` - 定义文件参数
- `FileSet[T any](fset *FlagSet, v *T, name, short, def, usage string)` - 向特定 FlagSet 添加文件参数
- `FileUnmarshal[T any](v *T, name, short, def, usage string, unmarshal func([]byte, any) error)` - 使用自定义反序列化函数定义文件参数

### 键值对相关函数

- `Pairs(v *[][2]string, name, short string, def [][2]string, usage string)` - 定义键值对参数
- `PairsSet(fset *FlagSet, v *[][2]string, name, short string, usage string)` - 向特定 FlagSet 添加键值对参数
- `Pair(v *[2]string, name, short string, def [][2]string, usage string)` - 定义单个键值对参数
- `PairSet(fset *FlagSet, v *[2]string, name, short string, usage string)` - 向特定 FlagSet 添加单个键值对参数

## 支持的数据类型

flagx 支持以下数据类型：

- 基础类型: `string`, `bool`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`
- 时间类型: `time.Duration`
- 网络类型: `net.IP`, `net.IPNet`
- 切片类型: 对应上述类型的切片
- 键值对类型: `[2]string`, `[][2]string`

## 结构体标签

你可以使用以下结构体标签来自定义参数行为：

- `flag` - 指定参数名、简写和用法（格式："name,shorthand,usage"）
- `short` - 指定简写
- `usage` - 指定用法描述
- `env` - 指定相关的环境变量

## 环境变量

可以通过 `env` 标签指定环境变量名称，当命令行参数未提供时会尝试从环境变量获取值：

```go
type Config struct {
    DatabaseURL string `flag:"db-url" env:"DATABASE_URL"`
}
```

## 版本和描述信息

你可以在程序中设置版本和描述信息：

```go
func main() {
    flagx.Version = "1.0.0"
    flagx.Description = "A sample application using flagx"

    // ... 你的代码 ...
}
```

## 许可证

MIT License
