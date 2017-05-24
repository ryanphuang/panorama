package types

import (
	du "deephealth/util"
	"regexp"
)

type Filter func(part string) (map[string]string, bool)
type FilterChain []Filter
type FilterTree []FilterChain

func (self *FilterTree) Eval(part string) (map[string]string, bool) {
	for _, chain := range *self {
		var result map[string]string
		for i, filter := range chain {
			ret, ok := filter(part)
			if !ok {
				// if this is the head of the chain, we continue to try the next chain
				// otherwise, we should stop and return false
				if i == 0 {
					break
				} else {
					return nil, false
				}
			}
			if ret != nil {
				if result == nil {
					result = ret
				} else {
					for k, v := range ret {
						result[k] = v
					}
				}
			}
		}
		return result, true
	}
	return nil, false // empty tree
}

func (self *FilterChain) Eval(part string) (map[string]string, bool) {
	var result map[string]string
	for _, filter := range *self {
		ret, ok := filter(part)
		if !ok {
			return nil, false
		}
		if ret != nil {
			if result == nil {
				result = ret
			} else {
				for k, v := range ret {
					result[k] = v
				}
			}
		}
	}
	return result, true
}

func NegateFilter(filter Filter) Filter {
	return func(part string) (map[string]string, bool) {
		_, ok := filter(part)
		return nil, !ok // result is not meaningful in negate filter
	}
}

func NewInSetFilter(elements ...string) Filter {
	var set map[string]bool
	for _, ele := range elements {
		set[ele] = true
	}
	return func(part string) (map[string]string, bool) {
		_, ok := set[part]
		return nil, ok
	}
}

func NewEqualsFilter(match string) Filter {
	return func(part string) (map[string]string, bool) {
		return nil, match == part
	}
}

func NewRegexFilter(pattern string) Filter {
	re := regexp.MustCompile(pattern)
	return func(part string) (map[string]string, bool) {
		return nil, re.MatchString(part)
	}
}

func NewMRegexpMapFilter(pattern string, prefix_group string) Filter {
	re := &du.MRegexp{regexp.MustCompile(pattern)}
	return func(part string) (map[string]string, bool) {
		match := re.FindStringSubmatchMap(part, prefix_group)
		return match, len(match) != 0
	}
}
