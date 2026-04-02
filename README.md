# 中国城市代码解析器 (Go)

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev/)

Go 语言实现的中国城市代码解析工具，从地址文本中解析出省、市、区/县三级行政区划及对应的 **GB/T 2260** 标准代码。

词典数据通过 `go:embed` 内嵌到库中，**无需额外部署词典文件**，import 后即可直接使用。

## 功能

- ✅ 完整地址解析（如 "广东省深圳市南山区科技园"）
- ✅ 简称/别名匹配（如 "深圳南山区" → 广东省深圳市南山区）
- ✅ 自由文本中提取地址（如 "我住在深圳市南山区"）
- ✅ 别名歧义处理（如 "重庆路" 不会匹配为重庆市）
- ✅ 返回 GB/T 2260 标准 6 位行政区划代码
- ✅ 支持直辖市（北京、上海、天津、重庆）

## 安装

```bash
go get github.com/plutotree/city_parser
```

## 快速开始

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"

    cityparser "github.com/plutotree/city_parser"
)

func main() {
    p := cityparser.NewCityParser()

    result := p.Parse("深圳市南山区科技园")
    if result == nil {
        log.Println("无法解析地址")
        return
    }

    jsonBytes, _ := json.MarshalIndent(result, "", "  ")
    fmt.Println(string(jsonBytes))
}
```

**输出结果**：

```json
{
  "code": "440305",
  "province": "广东省",
  "city": "深圳市",
  "county": "南山区"
}
```

## API 说明

### `NewCityParser() *CityParser`

创建解析器实例，初始化时即加载全部词典数据。

### `Parse(text string) (*CityResult, error)`

解析地址文本，返回结构化结果。

| 参数 | 类型 | 说明 |
|---|---|---|
| `text` | `string` | 待解析的地址文本 |

可能返回的错误：
- `ErrEmptyInput`：输入为空
- `ErrNoMatch`：无法解析出有效地址

### `CityResult` 返回结构

| 字段 | 类型 | 说明 |
|---|---|---|
| `Code` | `string` | 匹配到的最细粒度 GB/T 2260 代码 |
| `Province` | `string` | 省/自治区/直辖市 |
| `City` | `string` | 地级市/自治州（可能为空） |
| `County` | `string` | 区/县/县级市（可能为空） |

`Code` 返回匹配到的最细一级的代码：
- 匹配到区/县 → 返回区县代码（如 `440305`）
- 匹配到市   → 返回市级代码（如 `440300`）
- 匹配到省   → 返回省级代码（如 `440000`）
- 直辖市单独出现时返回省级代码（如 "重庆市" → `500000`）

JSON 序列化字段名：`code`、`province`、`city`、`county`。

## 更多示例

```go
p := cityparser.NewCityParser()

// 完整地址 → 返回区县级 code
p.Parse("四川省成都市武侯区")
// → Code: "510107", Province: "四川省", City: "成都市", County: "武侯区"

// 简称自动补全
p.Parse("深圳南山区")
// → Code: "440305", Province: "广东省", City: "深圳市", County: "南山区"

// 直辖市 + 区 → 返回区县级 code
p.Parse("北京市海淀区")
// → Code: "110108", Province: "北京市", City: "北京市", County: "海淀区"

// 直辖市单独出现 → 返回省级 code
p.Parse("重庆市")
// → Code: "500000", Province: "重庆市", City: "重庆市"

// 自由文本
p.Parse("我住在杭州市西湖区")
// → Code: "330106", Province: "浙江省", City: "杭州市", County: "西湖区"

// 仅城市 → 返回市级 code
p.Parse("成都市")
// → Code: "510100", Province: "四川省", City: "成都市"
```

## GB/T 2260 代码说明

代码为 6 位数字，遵循 GB/T 2260 标准：

| 位置 | 含义 | 示例 |
|---|---|---|
| 前 2 位 | 省级 | 44 = 广东省 |
| 中间 2 位 | 市级 | 4403 = 深圳市 |
| 后 2 位 | 区/县级 | 440305 = 南山区 |

- 省级代码：后 4 位为 `0000`（如 `440000`）
- 市级代码：后 2 位为 `00`（如 `440300`）
- 区县代码：完整 6 位（如 `440305`）

## 测试

```bash
# 运行所有单元测试
go test -v ./...

# 运行性能基准
go test -bench=. -benchmem ./...
```

## 代码结构

```
├── dictionary/
│   └── list.json                 # 内嵌词典数据（GB/T 2260，通过 go:embed 编译到二进制）
├── embed.go                      # go:embed 声明
├── types.go                      # 数据类型定义（CityResult、AdminItem 等）
├── loader.go                     # 词典数据解析，构建行政区划索引
├── parser.go                     # 核心解析逻辑
├── parser_test.go                # 单元测试 & 基准测试
├── go.mod
├── LICENSE
└── README.md
```

## 与 location_parser 的关系

本项目是 [location_parser](https://github.com/plutotree/location_parser) 的简化版本：

| | location_parser | city_parser |
|---|---|---|
| 数据源 | 大词典 (~700K行)，含乡镇村 | list.json (~3500条)，仅省市区 |
| 输出 | 文本地址 | **省市区 + GB/T 2260 代码** |
| 旧地名转换 | ✅ | ❌ |
| 乡镇村五级 | ✅ | ❌ |
| 解析算法 | 候选匹配 + 多轮筛选 | 同（移植自 location_parser） |
