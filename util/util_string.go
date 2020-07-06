package util

import (
	"strconv"
	"strings"
)

// MergeStringSlice 将两个字符串切片去重合并
func MergeStringSlice(s1 []string, s2 []string) []string {
	if s1 == nil && s2 == nil {
		return nil
	}
	if s1 == nil {
		return s2
	}
	if s2 == nil {
		return s1
	}
	s := append(s1, s2...)
	check := make(map[string]bool)
	for _, val := range s {
		check[val] = true
	}
	res := make([]string, 0, len(s1))
	for k := range check {
		res = append(res, k)
	}
	return res
}

// MergeStringMap 将两个map去重合并
func MergeStringMap(m1 map[string]string, m2 map[string]string) map[string]string {
	if m1 == nil && m2 == nil {
		return nil
	}
	if m1 == nil {
		return m2
	}
	if m2 == nil {
		return m1
	}
	for k, v := range m2 {
		_, ok := m1[k]
		if !ok {
			m1[k] = v
		}
	}
	return m1
}

// ParseValueAndUnit 解析包含单位的数值字符串，例如“2.7Gi”，分别返回数值和小写单位
func ParseValueAndUnit(str string) (val float64, unit string) {
	pos := -1
	for i, c := range str {
		if c < '0' || c > '9' {
			if c == '.' {
				continue
			}
			pos = i
			break
		}
	}
	if pos == -1 {
		v, _ := strconv.ParseFloat(str, 64)
		return v, ""
	}
	v, _ := strconv.ParseFloat(str[:pos], 64)
	return v, strings.ToLower(str[pos:])
}
