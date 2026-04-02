package cityparser

import (
	"encoding/json"
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
			result := p.Parse(tt.input)
			if result == nil {
				t.Fatalf("Parse(%q) returned nil", tt.input)
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
			result := p.Parse(tt.input)
			if result == nil {
				t.Fatalf("Parse(%q) returned nil", tt.input)
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
			result := p.Parse(tt.input)
			if result == nil {
				t.Fatalf("Parse(%q) returned nil", tt.input)
			}
			if result.City != tt.wantCity {
				t.Errorf("City = %q, want %q", result.City, tt.wantCity)
			}
		})
	}
}

func TestFreeText(t *testing.T) {
	p := NewCityParser()

	result := p.Parse("我住在深圳市南山区")
	if result == nil {
		t.Fatal("Parse returned nil")
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

	if result := p.Parse(""); result != nil {
		t.Errorf("Parse(\"\") = %+v, want nil", result)
	}
	if result := p.Parse("   "); result != nil {
		t.Errorf("Parse(\"   \") = %+v, want nil", result)
	}
}

func TestJSONOutput(t *testing.T) {
	p := NewCityParser()
	result := p.Parse("广东省深圳市南山区科技园")
	if result == nil {
		t.Fatal("Parse returned nil")
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent error: %v", err)
	}

	fmt.Println(string(jsonBytes))
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
