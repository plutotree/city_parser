package cityparser

import (
	"encoding/json"
	"strings"
)

// supplementary 是 list.json 中缺失的中间层级虚节点（直辖市的市级代码等）
var supplementary = map[string]string{
	// 直辖市
	"110100": "北京市",
	"120100": "天津市",
	"310100": "上海市",
	"500100": "重庆市",
	"500200": "县",
	// 港澳
	"810100": "香港",
	"820100": "澳门",
	// 省直辖县
	"419000": "省直辖县",
	"429000": "省直辖县",
	"469000": "省直辖县",
	"839000": "省直辖县",
	// 自治区直辖县
	"659000": "自治区直辖县",
}

// municipalitiesCities 直辖市别名集合
var municipalitiesCities = map[string]bool{
	"北京": true, "上海": true, "天津": true,
	"重庆": true, "香港": true, "澳门": true,
}

// regionSuffixes 行政区划后缀，用于生成别名（较长的排在前面避免被短后缀截断）
var regionSuffixes = []string{
	"特别行政区",
	"维吾尔自治区", "壮族自治区", "回族自治区",
	"自治区", "自治州", "自治县", "自治旗",
	"地区", "盟",
	"省", "市", "区", "县", "旗",
}

// trimSuffix 去掉地名后缀生成别名
func trimSuffix(name string) string {
	for _, suffix := range regionSuffixes {
		if strings.HasSuffix(name, suffix) {
			alias := strings.TrimSuffix(name, suffix)
			if len([]rune(alias)) >= 2 {
				return alias
			}
		}
	}
	return ""
}

// codeNode 临时节点，用于索引构建
type codeNode struct {
	code string
	name string
}

// buildAdminMapList 从内嵌的 list.json 构建行政区划映射表
func buildAdminMapList() []AdminItem {
	// 解析 list.json
	var rawMap map[string]string
	if err := json.Unmarshal(listJSON, &rawMap); err != nil {
		panic("cityparser: failed to parse embedded list.json: " + err.Error())
	}

	// 合并 supplementary
	for code, name := range supplementary {
		if _, exists := rawMap[code]; !exists {
			rawMap[code] = name
		}
	}

	// 分类为省、市、区县三级
	provinces := make(map[string]codeNode) // code(6位) -> node
	cities := make(map[string]codeNode)
	districts := make(map[string]codeNode)

	for code, name := range rawMap {
		if len(code) != 6 {
			continue
		}
		switch {
		case code[2:] == "0000": // 省级
			provinces[code] = codeNode{code: code, name: name}
		case code[4:] == "00": // 市级
			cities[code] = codeNode{code: code, name: name}
		default: // 区/县级
			districts[code] = codeNode{code: code, name: name}
		}
	}

	var adminList []AdminItem

	// 1. 添加省级条目（非直辖市）
	for _, prov := range provinces {
		provAlias := trimSuffix(prov.name)
		if provAlias == "" {
			provAlias = prov.name
		}
		if municipalitiesCities[provAlias] {
			continue // 直辖市不添加仅省级的条目
		}
		adminList = append(adminList, AdminItem{
			Code:     prov.code,
			Province: NamePair{FullName: prov.name, Alias: provAlias},
			Offsets:  [3]OffsetInfo{{-1, -1}, {-1, -1}, {-1, -1}},
		})
	}

	// 2. 添加市级条目
	for _, city := range cities {
		provCode := city.code[:2] + "0000"
		prov, ok := provinces[provCode]
		if !ok {
			continue
		}
		provAlias := trimSuffix(prov.name)
		if provAlias == "" {
			provAlias = prov.name
		}
		cityAlias := trimSuffix(city.name)
		if cityAlias == "" {
			cityAlias = city.name
		}

		adminList = append(adminList, AdminItem{
			Code:     city.code,
			Province: NamePair{FullName: prov.name, Alias: provAlias},
			City:     NamePair{FullName: city.name, Alias: cityAlias},
			Offsets:  [3]OffsetInfo{{-1, -1}, {-1, -1}, {-1, -1}},
		})
	}

	// 3. 添加区/县级条目
	for _, dist := range districts {
		cityCode := dist.code[:4] + "00"
		provCode := dist.code[:2] + "0000"

		prov, ok := provinces[provCode]
		if !ok {
			continue
		}
		provAlias := trimSuffix(prov.name)
		if provAlias == "" {
			provAlias = prov.name
		}

		city, cityOk := cities[cityCode]
		// 直辖市区县的 cityCode 可能不在 cities 中（如重庆 5001xx → 500100）
		// 尝试 supplementary 中的虚节点
		if !cityOk {
			if name, suppOk := supplementary[cityCode]; suppOk {
				city = codeNode{code: cityCode, name: name}
				cityOk = true
			}
		}
		var cityName, cityAlias string
		if cityOk {
			cityName = city.name
			cityAlias = trimSuffix(city.name)
			if cityAlias == "" {
				cityAlias = city.name
			}
		}

		distAlias := trimSuffix(dist.name)
		if distAlias == "" {
			distAlias = dist.name
		}

		adminList = append(adminList, AdminItem{
			Code:     dist.code,
			Province: NamePair{FullName: prov.name, Alias: provAlias},
			City:     NamePair{FullName: cityName, Alias: cityAlias},
			County:   NamePair{FullName: dist.name, Alias: distAlias},
			Offsets:  [3]OffsetInfo{{-1, -1}, {-1, -1}, {-1, -1}},
		})
	}

	return adminList
}

// provinceCode 从 6 位代码提取省级代码
func provinceCode(code string) string {
	if len(code) == 6 {
		return code[:2] + "0000"
	}
	return ""
}


