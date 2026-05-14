package cityparser

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestBasicParse(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input        string
		wantCode     string
		wantProvince string
		wantCity     string
		wantCounty   string
	}{
		// 完整地址 → 返回区县级 code
		{"广东省深圳市南山区科技园", "440305", "广东省", "深圳市", "南山区"},
		{"四川省成都市武侯区", "510107", "四川省", "成都市", "武侯区"},
		{"浙江省杭州市西湖区", "330106", "浙江省", "杭州市", "西湖区"},

		// 简称
		{"深圳南山区", "440305", "广东省", "深圳市", "南山区"},
		{"成都武侯区", "510107", "四川省", "成都市", "武侯区"},

		// 仅城市 → 返回市级 code
		{"深圳市", "440300", "广东省", "深圳市", ""},
		{"成都市", "510100", "四川省", "成都市", ""},

		// 仅省 → 返回省级 code
		{"广东省", "440000", "广东省", "", ""},
		{"四川省", "510000", "四川省", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if result.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", result.Code, tt.wantCode)
			}
			if result.Province != tt.wantProvince {
				t.Errorf("Province = %q, want %q", result.Province, tt.wantProvince)
			}
			if result.City != tt.wantCity {
				t.Errorf("City = %q, want %q", result.City, tt.wantCity)
			}
			if result.County != tt.wantCounty {
				t.Errorf("County = %q, want %q", result.County, tt.wantCounty)
			}
		})
	}
}

func TestMunicipalities(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input        string
		wantCode     string
		wantProvince string
		wantCity     string
		wantCounty   string
	}{
		// 直辖市 + 区 → 返回区县级 code
		{"北京市朝阳区", "110105", "北京市", "北京市", "朝阳区"},
		{"上海市浦东新区", "310115", "上海市", "上海市", "浦东新区"},
		{"天津市南开区", "120104", "天津市", "天津市", "南开区"},
		{"重庆市渝中区", "500103", "重庆市", "重庆市", "渝中区"},

		// 直辖市单独出现 → 返回省级 code（不返回 supplementary 虚拟的 xx0100）
		{"北京市", "110000", "北京市", "北京市", ""},
		{"上海市", "310000", "上海市", "上海市", ""},
		{"重庆市", "500000", "重庆市", "重庆市", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if result.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", result.Code, tt.wantCode)
			}
			if result.Province != tt.wantProvince {
				t.Errorf("Province = %q, want %q", result.Province, tt.wantProvince)
			}
			if result.City != tt.wantCity {
				t.Errorf("City = %q, want %q", result.City, tt.wantCity)
			}
			if result.County != tt.wantCounty {
				t.Errorf("County = %q, want %q", result.County, tt.wantCounty)
			}
		})
	}
}

func TestAliasDisambiguation(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input    string
		wantCity string
		desc     string
	}{
		// "重庆路" 不应匹配为重庆市
		{"大连市重庆路100号", "大连市", "重庆路不应匹配为重庆市"},
		// "太原街" 不应匹配为太原市
		{"沈阳市太原街", "沈阳市", "太原街不应匹配为太原市"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if result.City != tt.wantCity {
				t.Errorf("City = %q, want %q", result.City, tt.wantCity)
			}
		})
	}
}

func TestFreeText(t *testing.T) {
	p := NewCityParser()

	result, err := p.Parse("我住在深圳市南山区")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if result.Code != "440305" {
		t.Errorf("Code = %q, want %q", result.Code, "440305")
	}
	if result.City != "深圳市" {
		t.Errorf("City = %q, want %q", result.City, "深圳市")
	}
	if result.County != "南山区" {
		t.Errorf("County = %q, want %q", result.County, "南山区")
	}
}

func TestEmptyInput(t *testing.T) {
	p := NewCityParser()

	_, err := p.Parse("")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("Parse(\"\") error = %v, want ErrEmptyInput", err)
	}

	_, err = p.Parse("   ")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("Parse(\"   \") error = %v, want ErrEmptyInput", err)
	}
}

func TestNoMatch(t *testing.T) {
	p := NewCityParser()

	_, err := p.Parse("hello world")
	if !errors.Is(err, ErrNoMatch) {
		t.Errorf("Parse(\"hello world\") error = %v, want ErrNoMatch", err)
	}
}

func TestJSONOutput(t *testing.T) {
	p := NewCityParser()
	result, err := p.Parse("广东省深圳市南山区科技园")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent error: %v", err)
	}

	fmt.Println(string(jsonBytes))
}

func TestRemainder(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input       string
		wantCode    string
		wantRemain  string
	}{
		// 去除区县级后的剩余文本
		{"广东省深圳市南山区科技园", "440305", "科技园"},
		{"深圳南山区科技园路100号", "440305", "科技园路100号"},
		{"成都武侯区天府大道", "510107", "天府大道"},

		// 仅市级匹配时，去除市后的剩余文本
		{"深圳市科技园", "440300", "科技园"},
		{"成都市高新区", "510100", "高新区"},

		// 仅省级匹配时
		{"广东省某个地方", "440000", "某个地方"},

		// === 行政名出现在中间或多种前缀变体（v0.2.0 之后语义升级）===
		// 行政名前置（不同前缀写法都应得到相同 Remainder）
		{"沈阳市星海国际象棋俱乐部", "210100", "星海国际象棋俱乐部"},
		{"沈阳星海国际象棋俱乐部", "210100", "星海国际象棋俱乐部"},
		{"辽宁沈阳星海国际象棋俱乐部", "210100", "星海国际象棋俱乐部"},
		{"辽宁省沈阳市星海国际象棋俱乐部", "210100", "星海国际象棋俱乐部"},
		// 行政名出现在中间（无标点）—— 旧实现会得到 "国际象棋俱乐部"，新实现保留前文
		{"星海沈阳国际象棋俱乐部", "210100", "星海国际象棋俱乐部"},
		// 行政名出现在中间（括号内）—— 保留括号位置形态，由调用方做后续清洗
		{"星海（沈阳）国际象棋俱乐部", "210100", "星海（）国际象棋俱乐部"},
		// 不应因为后缀变化误匹配（"星海无边" 与 "星海" 不同）
		{"沈阳市星海无边国际象棋俱乐部", "210100", "星海无边国际象棋俱乐部"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if result.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", result.Code, tt.wantCode)
			}
			if result.Remainder != tt.wantRemain {
				t.Errorf("Remainder = %q, want %q", result.Remainder, tt.wantRemain)
			}
		})
	}
}

// TestNoPhantomCounty 验证 buildResult 不会把没在文本里出现的 County 填入结果。
//
// 历史 bug：当输入只有省/市级关键词、加上一些与某区县级 item 同省的非地名词
// （如"上海江宁青少年体育俱乐部"），筛选可能选中某个区县级 item，但其 County
// 在原文中根本没出现。旧实现按 Code 层级直接填 County.FullName，造成"幻觉"。
//
// 修复后：必须按 Offsets[i].Pos > -1 实际命中层级填字段，Code 同步回退。
func TestNoPhantomCounty(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input        string
		wantCode     string
		wantProvince string
		wantCity     string
		wantCounty   string
	}{
		// "江宁" 是江苏南京下辖区名，但因为有"上海"前置且文本是俱乐部名，
		// 不应吸引到任何上海的具体 county
		{"上海江宁青少年体育俱乐部", "310000", "上海市", "上海市", ""},
		// "浦东" 在上海是 County.Alias，但文本里没"浦东新区"全名，
		// "浦东" 也未必命中（取决于筛选）；不论如何 County 不应填错的值
		{"上海浦东弈步体育俱乐部", "310000", "上海市", "上海市", ""},
		// 同样：俱乐部名里有"奉贤"两字也不该乱命中
		{"上海某某俱乐部", "310000", "上海市", "上海市", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if r.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", r.Code, tt.wantCode)
			}
			if r.Province != tt.wantProvince {
				t.Errorf("Province = %q, want %q", r.Province, tt.wantProvince)
			}
			if r.City != tt.wantCity {
				t.Errorf("City = %q, want %q", r.City, tt.wantCity)
			}
			if r.County != tt.wantCounty {
				t.Errorf("County = %q, want %q", r.County, tt.wantCounty)
			}
		})
	}
}

// TestStableOutput 验证同一个 parser 重复调用同一输入，结果稳定（无内部状态污染）。
// TestSubstringNoiseFilter 验证 step 2.0 的"子串噪音"过滤覆盖两类场景：
//
//  类型 A：一 FullName + 一 Alias 同位置——例如 item=辽宁/朝阳市/朝阳县
//          被"朝阳市"输入误拉入（County.Alias="朝阳" 是 City.FullName="朝阳市"
//          的前缀子串）。这种 item 整条剔除后，结果应是 朝阳市 (211300)。
//
//  类型 B：≥2 个全 Alias 同位置——例如 item=辽宁/朝阳市/朝阳县 被
//          "沈阳市朝阳区某地"误拉入（City 与 County 都用 Alias="朝阳" 同位置命中）。
//          这种 item 整条剔除后，结果应是 沈阳市 (210100)。
//
// 直辖市（Province=City 同名）必然同位置命中，作为例外不参与过滤。
func TestSubstringNoiseFilter(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input        string
		wantCode     string
		wantProvince string
		wantCity     string
		wantCounty   string
		desc         string
	}{
		// 类型 A：朝阳市单独 — 不应误命中朝阳县
		{"朝阳市", "211300", "辽宁省", "朝阳市", "", "类型A: 朝阳市 不应被朝阳县 item 吸引"},
		// 类型 B：沈阳市朝阳区某地 — 不应误命中朝阳市朝阳县
		{"沈阳市朝阳区某地", "210100", "辽宁省", "沈阳市", "", "类型B: 沈阳市 全名命中应胜过朝阳的 alias 噪音"},
		{"沈阳市朝阳", "210100", "辽宁省", "沈阳市", "", "类型B: 同上但末尾无'区'"},
		{"沈阳朝阳区", "210100", "辽宁省", "沈阳市", "", "类型B: 沈阳作 alias 也应胜出"},
		// 反向 case：用户确实写了朝阳县全名，应正常命中
		{"朝阳县", "211321", "辽宁省", "朝阳市", "朝阳县", "朝阳县全名输入应正常命中区县级"},
		{"辽宁省朝阳市朝阳县", "211321", "辽宁省", "朝阳市", "朝阳县", "完整地址不受过滤影响"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			r, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if r.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", r.Code, tt.wantCode)
			}
			if r.Province != tt.wantProvince {
				t.Errorf("Province = %q, want %q", r.Province, tt.wantProvince)
			}
			if r.City != tt.wantCity {
				t.Errorf("City = %q, want %q", r.City, tt.wantCity)
			}
			if r.County != tt.wantCounty {
				t.Errorf("County = %q, want %q", r.County, tt.wantCounty)
			}
		})
	}
}

func TestStableOutput(t *testing.T) {
	p := NewCityParser()

	inputs := []string{
		"上海江宁青少年体育俱乐部",
		"广东省深圳市南山区科技园",
		"沈阳市星海国际象棋俱乐部",
		"重庆市渝中区",
	}
	// 跑第一遍记录结果
	first := make(map[string]string)
	for _, in := range inputs {
		r, err := p.Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", in, err)
		}
		first[in] = fmt.Sprintf("code=%s prov=%s city=%s county=%s remainder=%s",
			r.Code, r.Province, r.City, r.County, r.Remainder)
	}

	// 多跑几遍，结果必须完全一致
	for round := 0; round < 5; round++ {
		// 故意打散顺序穿插一些其它输入
		for _, in := range []string{
			"广州市天河区",
			"沈阳市朝阳区某地",
			"上海市浦东新区",
		} {
			_, _ = p.Parse(in)
		}
		for _, in := range inputs {
			r, err := p.Parse(in)
			if err != nil {
				t.Fatalf("round %d Parse(%q) error: %v", round, in, err)
			}
			got := fmt.Sprintf("code=%s prov=%s city=%s county=%s remainder=%s",
				r.Code, r.Province, r.City, r.County, r.Remainder)
			if got != first[in] {
				t.Errorf("round %d Parse(%q) unstable:\n  first: %s\n  now:   %s",
					round, in, first[in], got)
			}
		}
	}
}

func BenchmarkParse(b *testing.B) {
	p := NewCityParser()
	// 预热
	p.Parse("深圳市南山区")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse("广东省深圳市南山区科技园")
	}
}

// TestMunicipalityOffsetSync 直辖市 Province 与 City 在同位置命中时，必须把 City
// Offset 同步抹掉（同时扣 MatchCount），否则下游 sumAliasIdx 度量会错把直辖市
// item 算成"双层 alias 命中"，反而输给只命中 County 的远方同字 item。
//
// 典型 v0.5.0 误判：
//
//	"上海宁弈" → 浙江省/嘉兴市/海宁市
//	"上海林峰" → 黑龙江省/牡丹江市/海林市
func TestMunicipalityOffsetSync(t *testing.T) {
	p := NewCityParser()

	tests := []struct {
		input    string
		wantCode string
	}{
		// "上海"两字命中直辖市 Province+City 同位置；
		// "宁弈"中的"宁"不应让浙江/海宁市抢走结果。
		{"上海宁弈", "310000"},
		// 同上，"林"不应让黑龙江/海林市抢走。
		{"上海林峰", "310000"},
		// 直辖市+区 仍应解析到区县级。
		{"上海市黄浦区", "310101"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			r, err := p.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if r.Code != tt.wantCode {
				t.Errorf("Parse(%q).Code = %q, want %q (province=%s city=%s county=%s)",
					tt.input, r.Code, tt.wantCode, r.Province, r.City, r.County)
			}
		})
	}
}

// TestStableAcrossInstances 同一输入在多个独立 NewCityParser 实例上的解析结果
// 必须完全一致。
//
// 历史背景：v0.5.0 之前 buildAdminMapList 通过 map 遍历构建候选切片，Go 会随机化
// map 遍历顺序，导致不同进程/不同 parser 实例对"真歧义"输入（如"普陀棋院"——
// 上海/普陀区 vs 浙江/舟山/普陀区）输出不同结果。v0.6.0 在 loader 里按 code
// 升序排序后稳定。
func TestStableAcrossInstances(t *testing.T) {
	inputs := []string{
		"普陀棋院",       // 真歧义：仅"普陀"无其他线索
		"广东省深圳市南山区科技园",
		"上海市浦东新区",
		"沈阳市朝阳区某地",
	}

	const N = 30 // 30 个独立 parser 实例
	parsers := make([]*CityParser, N)
	for i := 0; i < N; i++ {
		parsers[i] = NewCityParser()
	}

	for _, in := range inputs {
		want, err := parsers[0].Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q) error: %v", in, err)
		}
		for i := 1; i < N; i++ {
			got, err := parsers[i].Parse(in)
			if err != nil {
				t.Fatalf("instance %d Parse(%q) error: %v", i, in, err)
			}
			if got.Code != want.Code ||
				got.Province != want.Province ||
				got.City != want.City ||
				got.County != want.County {
				t.Errorf("instance %d Parse(%q) drift:\n  want %+v\n  got  %+v",
					i, in, want, got)
			}
		}
	}
}
