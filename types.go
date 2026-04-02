package cityparser

// CityResult 地址解析结果，包含省/市/区县三级名称及一个 GB/T 2260 代码
//
// Code 是匹配到的最细粒度的行政区划代码：
//   - 匹配到区/县级 → 返回区县代码（如 440305）
//   - 匹配到市级   → 返回市级代码（如 440300）
//   - 匹配到省级   → 返回省级代码（如 440000）
//
// 注意：supplementary 中的虚拟节点（如直辖市的 500100）不会作为 Code 返回，
// 会自动回退到上一级真实代码（如 500000）。
type CityResult struct {
	Code     string `json:"code"`               // GB/T 2260 代码
	Province string `json:"province,omitempty"`  // 省
	City     string `json:"city,omitempty"`      // 市
	County   string `json:"county,omitempty"`    // 区/县
}

// NamePair 存储地名的全名和别名
type NamePair struct {
	FullName string // 全名，如 "四川省"、"成都市"、"武侯区"
	Alias    string // 别名，如 "四川"、"成都"、"武侯"
}

// OffsetInfo 存储匹配位置信息
type OffsetInfo struct {
	Pos      int // rune 偏移位置，-1 表示未匹配
	AliasIdx int // 0=全名匹配，1=别名匹配
}

// AdminItem 行政区划条目
type AdminItem struct {
	Code     string   // GB/T 2260 代码，如 "440305"
	Province NamePair // 省
	City     NamePair // 市
	County   NamePair // 区/县

	// 匹配阶段填充
	MatchCount int
	Offsets    [3]OffsetInfo // [省, 市, 县] 的匹配偏移信息
}
