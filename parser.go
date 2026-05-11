package cityparser

import (
	"errors"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrEmptyInput = errors.New("empty input text")
	ErrNoMatch    = errors.New("no matching location found")
)

// CityParser 中国城市代码解析器
type CityParser struct {
	adminMapList           []AdminItem
	locAliasString         string
	exceptionSuffixPattern *regexp.Regexp
}

// NewCityParser 创建解析器实例，初始化时即加载全部数据
func NewCityParser() *CityParser {
	p := &CityParser{
		locAliasString:         "【loc_alias】",
		exceptionSuffixPattern: regexp.MustCompile(`(【loc_alias】(路|大街|街|道|巷|弄|里|胡同|大道|公路|高速))`),
		adminMapList:           buildAdminMapList(),
	}
	return p
}

// Parse 解析地址文本，返回省/市/区县三级及对应的 GB/T 2260 代码
//
// 参数:
//   - locationText: 待解析的地址文本
//
// 返回错误:
//   - ErrEmptyInput: 输入为空
//   - ErrNoMatch: 无法解析出有效地址
func (p *CityParser) Parse(locationText string) (*CityResult, error) {
	locationText = strings.TrimSpace(locationText)
	if locationText == "" {
		return nil, ErrEmptyInput
	}

	// Step 1: 获取候选
	candidateIdxList := p.getCandidates(locationText)

	if len(candidateIdxList) == 0 {
		return nil, ErrNoMatch
	}

	// Step 2: 多轮筛选

	// 2.0 去除同一 offset 匹配了多个别名的
	var filtered []int
	for _, idx := range candidateIdxList {
		item := &p.adminMapList[idx]
		offsets := []int{}
		for _, o := range item.Offsets {
			if o.Pos > -1 {
				offsets = append(offsets, o.Pos)
			}
		}
		if hasDuplicate(offsets) {
			dupVal := findMostCommon(offsets)
			var sameOffsetInfos []OffsetInfo
			for _, o := range item.Offsets {
				if o.Pos == dupVal {
					sameOffsetInfos = append(sameOffsetInfos, o)
				}
			}
			if len(sameOffsetInfos) == 2 &&
				sameOffsetInfos[0].AliasIdx == 0 && sameOffsetInfos[1].AliasIdx == 1 {
				continue
			}
			filtered = append(filtered, idx)
		} else {
			filtered = append(filtered, idx)
		}
	}
	candidateIdxList = filtered

	if len(candidateIdxList) == 0 {
		return nil, ErrNoMatch
	}

	// 2.1 找出匹配数量最多的
	maxMatched := 0
	for _, idx := range candidateIdxList {
		if p.adminMapList[idx].MatchCount > maxMatched {
			maxMatched = p.adminMapList[idx].MatchCount
		}
	}
	filtered = nil
	for _, idx := range candidateIdxList {
		if p.adminMapList[idx].MatchCount == maxMatched {
			filtered = append(filtered, idx)
		}
	}
	candidateIdxList = filtered

	// 仅一个候选，直接返回
	if len(candidateIdxList) == 1 {
		return p.buildResult(&p.adminMapList[candidateIdxList[0]], locationText), nil
	}

	// 2.2 找出匹配位置最靠前的
	sort.Slice(candidateIdxList, func(i, j int) bool {
		return offsetSum(&p.adminMapList[candidateIdxList[i]]) < offsetSum(&p.adminMapList[candidateIdxList[j]])
	})

	// 过滤：省市县全匹配到时，必须按省<市<县的顺序
	var newCandidates []int
	for _, idx := range candidateIdxList {
		item := &p.adminMapList[idx]
		if municipalitiesCities[item.Province.Alias] {
			newCandidates = append(newCandidates, idx)
		} else {
			if item.Offsets[0].Pos != -1 && item.Offsets[1].Pos != -1 && item.Offsets[2].Pos != -1 {
				if item.Offsets[0].Pos < item.Offsets[1].Pos && item.Offsets[1].Pos < item.Offsets[2].Pos {
					newCandidates = append(newCandidates, idx)
				}
			} else {
				newCandidates = append(newCandidates, idx)
			}
		}
	}
	candidateIdxList = newCandidates

	if len(candidateIdxList) == 0 {
		return nil, ErrNoMatch
	}

	minOffset := offsetSum(&p.adminMapList[candidateIdxList[0]])
	filtered = nil
	for _, idx := range candidateIdxList {
		if offsetSum(&p.adminMapList[idx]) == minOffset {
			filtered = append(filtered, idx)
		}
	}
	candidateIdxList = filtered

	// 2.3 优先匹配全名，过滤别名
	fullAliasList := make([]int, len(candidateIdxList))
	for i, idx := range candidateIdxList {
		fullAliasList[i] = minAliasIdx(&p.adminMapList[idx])
	}
	fullAliasMin := minInt(fullAliasList)
	filtered = nil
	for i, idx := range candidateIdxList {
		if fullAliasList[i] == fullAliasMin {
			filtered = append(filtered, idx)
		}
	}
	candidateIdxList = filtered

	// 取别名总和最小的
	fullAliasSumList := make([]int, len(candidateIdxList))
	for i, idx := range candidateIdxList {
		fullAliasSumList[i] = sumAliasIdx(&p.adminMapList[idx])
	}
	fullAliasSumMin := minInt(fullAliasSumList)
	filtered = nil
	for i, idx := range candidateIdxList {
		if fullAliasSumList[i] == fullAliasSumMin {
			filtered = append(filtered, idx)
		}
	}
	candidateIdxList = filtered

	// 2.4 全是别名时，匹配级别越高越好
	aliasMatchedNums := make([]int, len(candidateIdxList))
	maxAliasMatchedNum := 0
	for i, idx := range candidateIdxList {
		item := &p.adminMapList[idx]
		cnt := 0
		for _, o := range item.Offsets {
			if o.Pos > -1 {
				cnt++
			}
		}
		aliasMatchedNums[i] = cnt
		if cnt > maxAliasMatchedNum {
			maxAliasMatchedNum = cnt
		}
	}

	if fullAliasMin == 1 && maxAliasMatchedNum == 1 {
		sort.Slice(candidateIdxList, func(i, j int) bool {
			return firstMatchedLevel(&p.adminMapList[candidateIdxList[i]]) <
				firstMatchedLevel(&p.adminMapList[candidateIdxList[j]])
		})
	}

	if len(candidateIdxList) == 0 {
		return nil, ErrNoMatch
	}

	// 取第一个作为最终结果
	return p.buildResult(&p.adminMapList[candidateIdxList[0]], locationText), nil
}

// processExceptionAlias 处理异常别名（如 "太原路" 中的 "太原" 不应匹配）
func (p *CityParser) processExceptionAlias(name string, locationText string) bool {
	replaced := strings.Replace(locationText, name, p.locAliasString, 1)
	matched := p.exceptionSuffixPattern.FindString(replaced)
	return matched == ""
}

// getCandidates 从地址中获取所有可能涉及到的候选地址
func (p *CityParser) getCandidates(locationText string) []int {
	textRunes := []rune(locationText)
	var candidateIdxList []int

	for i := range p.adminMapList {
		item := &p.adminMapList[i]
		count := 0
		item.Offsets = [3]OffsetInfo{{-1, -1}, {-1, -1}, {-1, -1}}

		names := [3]NamePair{item.Province, item.City, item.County}

		skip := false
		for idx := 0; idx < 3; idx++ {
			np := names[idx]
			matchFlag := false
			var curName []rune
			var curAlias int

			// 先尝试全名(aliasIdx=0)，再尝试别名(aliasIdx=1)
			for aliasIdx, name := range []string{np.FullName, np.Alias} {
				if name == "" {
					continue
				}
				nameRunes := []rune(name)
				if runeContains(textRunes, nameRunes) {
					if aliasIdx == 1 {
						if !p.processExceptionAlias(name, locationText) {
							continue
						}
					}
					matchFlag = true
					curName = nameRunes
					curAlias = aliasIdx
					break
				}
			}

			if matchFlag {
				count++
				pos := runeIndex(textRunes, curName)
				item.Offsets[idx] = OffsetInfo{Pos: pos, AliasIdx: curAlias}

				// 相邻偏移检查：如 "青海西宁" 不应匹配到 "海西"
				if idx == 1 && item.Offsets[0].Pos >= 0 {
					if item.Offsets[1].Pos-item.Offsets[0].Pos == 1 {
						count = 0
						skip = true
						break
					}
				}
				if idx == 2 {
					if item.Offsets[1].Pos >= 0 {
						if item.Offsets[2].Pos-item.Offsets[1].Pos == 1 {
							count = 0
							skip = true
							break
						}
					}
					if item.Offsets[0].Pos >= 0 {
						if item.Offsets[2].Pos-item.Offsets[0].Pos == 1 {
							count = 0
							skip = true
							break
						}
					}
				}
			}
		}

		if skip {
			continue
		}

		if count > 0 {
			// 直辖市处理
			if municipalitiesCities[item.Province.Alias] {
				provAliasRunes := []rune(item.Province.Alias)
				if runeContains(textRunes, provAliasRunes) {
					count--
				}
			}
			item.MatchCount = count
			candidateIdxList = append(candidateIdxList, i)
		}
	}

	return candidateIdxList
}

// buildResult 根据最终匹配的 AdminItem 构建 CityResult。
//
// 关键约束：返回的 Province / City / County 必须**确实在原文里出现过**
// （Offsets[i].Pos > -1）。否则会出现"输入只含上海，但被某个区县级 item 命中
// 后误填某个具体的 County"这类幻觉式结果（v0.3.0 及更早版本的 bug）。
//
// 因此 Code 也按真正命中的最深层级回退：
//   - County 命中 → 返回区县级代码 (item.Code)
//   - 仅 City 命中 → 截断到市级 (XXXX00)
//   - 仅 Province 命中 → 截断到省级 (XX0000)
func (p *CityParser) buildResult(item *AdminItem, inputText string) *CityResult {
	result := &CityResult{}

	provHit := item.Offsets[0].Pos > -1
	cityHit := item.Offsets[1].Pos > -1
	countyHit := item.Offsets[2].Pos > -1

	// 由 item.Code 推导各层级 code（不依赖 item 当前层级）
	code := item.Code
	provCode := code[:2] + "0000"
	cityCodeStr := code[:4] + "00"

	switch {
	case countyHit:
		result.Code = code
		result.Province = item.Province.FullName
		result.City = item.City.FullName
		result.County = item.County.FullName
	case cityHit:
		result.Code = cityCodeStr
		result.Province = item.Province.FullName
		result.City = item.City.FullName
	case provHit:
		result.Code = provCode
		result.Province = item.Province.FullName
	default:
		// 理论上 candidate 至少命中一层，否则不会进 candidate 列表；
		// 兜底返回省级，避免空 Code。
		result.Code = provCode
		result.Province = item.Province.FullName
	}

	// 直辖市处理：省=市的情况下，city 填省名
	if municipalitiesCities[item.Province.Alias] {
		if result.City == "" || strings.Contains(result.City, "直辖") {
			result.City = result.Province
		}
	}

	// 去掉含 "直辖" 的中间层
	if strings.Contains(result.City, "直辖") {
		result.City = ""
	}

	// supplementary 中的虚拟节点不作为最终 code 返回，回退到上一级
	if _, isSuppNode := supplementary[result.Code]; isSuppNode {
		result.Code = provinceCode(result.Code)
	}

	// 计算 Remainder：去除省/市/区县后的剩余文本
	result.Remainder = computeRemainder(item, inputText)

	return result
}

// computeRemainder 计算去除行政区划后的剩余文本。
//
// 语义：把命中的省/市/县名（按 Offsets[i].AliasIdx 决定 FullName 还是 Alias）
// 依次从原文中擦除，返回剩余文本。
//
// 与"截取最后匹配位置之后的文本"不同，本实现支持行政名出现在中间的情况，例如：
//
//	输入: "星海（沈阳）国际象棋俱乐部" → "星海（）国际象棋俱乐部"
//	输入: "星海沈阳国际象棋俱乐部"     → "星海国际象棋俱乐部"
//
// 擦除规则：
//  1. 按命中层级（省→市→县）依次处理，按 AliasIdx 选择擦 FullName 还是 Alias，
//     用 strings.Index 找首次命中位置，仅擦除该一处（避免误擦同名子串）。
//  2. 每次擦除后，若紧邻的下一个字符是常见行政尾缀（市/区/县/镇/乡/旗）之一，
//     一并擦除。这是为了覆盖 Alias 命中场景（如命中"沈阳"但原文是"沈阳市..."）
//     时的尾缀残留，行为与旧版 TrimPrefix 兼容。
//  3. 直辖市的省名与市名 Alias 相同（如"北京"），可能在 Province 与 City 两个
//     Offset 上都登记了同一位置，会用相同的 Index 重复擦除——按 strings.Index
//     的语义，第一次擦掉后再找会找到下一处或返回 -1，结果幂等。
//  4. 最后做一次 TrimSpace 去除首尾空白，不动其他标点。
func computeRemainder(item *AdminItem, inputText string) string {
	if inputText == "" {
		return ""
	}

	names := [3]NamePair{item.Province, item.City, item.County}

	// 收集要擦除的名字（按命中时实际使用的形式）。
	type erase struct {
		name string
		pos  int // rune offset，用于按出现位置升序擦除（保持稳定）
	}
	var toErase []erase
	for idx, o := range item.Offsets {
		if o.Pos < 0 {
			continue
		}
		var name string
		if o.AliasIdx == 0 {
			name = names[idx].FullName
		} else {
			name = names[idx].Alias
		}
		if name == "" {
			continue
		}
		toErase = append(toErase, erase{name: name, pos: o.Pos})
	}

	// 按出现位置升序，从前往后擦除。
	// 这样擦除位置紧邻字符的判断更稳定（不受其他擦除已移除的字符干扰）。
	sort.Slice(toErase, func(i, j int) bool {
		return toErase[i].pos < toErase[j].pos
	})

	result := inputText
	suffixesToTrim := []string{"市", "区", "县", "镇", "乡", "旗"}

	for _, e := range toErase {
		idx := strings.Index(result, e.name)
		if idx < 0 {
			// 直辖市等场景：Province 与 City 命中同位置同名，第二次擦不到。
			continue
		}
		head := result[:idx]
		tail := result[idx+len(e.name):]
		// 紧邻尾缀清理（防御 Alias 命中后残留的尾缀字）
		for _, suf := range suffixesToTrim {
			if strings.HasPrefix(tail, suf) {
				tail = tail[len(suf):]
				break
			}
		}
		result = head + tail
	}

	return strings.TrimSpace(result)
}

// === rune 操作辅助函数 ===

func runeIndex(text []rune, sub []rune) int {
	if len(sub) == 0 || len(sub) > len(text) {
		return -1
	}
	for i := 0; i <= len(text)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if text[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func runeContains(text []rune, sub []rune) bool {
	return runeIndex(text, sub) >= 0
}

// === 筛选辅助函数 ===

func offsetSum(item *AdminItem) int {
	sum := 0
	for _, o := range item.Offsets {
		sum += o.Pos
	}
	return sum
}

func minAliasIdx(item *AdminItem) int {
	min := 999
	for _, o := range item.Offsets {
		if o.Pos > -1 && o.AliasIdx < min {
			min = o.AliasIdx
		}
	}
	if min == 999 {
		return 0
	}
	return min
}

func sumAliasIdx(item *AdminItem) int {
	sum := 0
	for _, o := range item.Offsets {
		if o.Pos > -1 {
			sum += o.AliasIdx
		}
	}
	return sum
}

func firstMatchedLevel(item *AdminItem) int {
	for idx, o := range item.Offsets {
		if o.Pos != -1 {
			return idx
		}
	}
	return 3
}

func hasDuplicate(nums []int) bool {
	seen := map[int]bool{}
	for _, n := range nums {
		if seen[n] {
			return true
		}
		seen[n] = true
	}
	return false
}

func findMostCommon(nums []int) int {
	counts := map[int]int{}
	for _, n := range nums {
		counts[n]++
	}
	maxCount := 0
	maxVal := 0
	for v, c := range counts {
		if c > maxCount {
			maxCount = c
			maxVal = v
		}
	}
	return maxVal
}

func minInt(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	m := nums[0]
	for _, n := range nums[1:] {
		if n < m {
			m = n
		}
	}
	return m
}
