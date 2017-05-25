package types

import (
	"fmt"
	"regexp"
	"strings"

	du "deephealth/util"
)

type FieldFilter func(fields map[string]string) (map[string]string, bool)
type FieldFilterChain []FieldFilter
type FieldFilterTree []FieldFilterChain

func (self *FieldFilterTree) Eval(fields map[string]string) (map[string]string, bool) {
	for _, chain := range *self {
		var result map[string]string
		found := true
		for i, filter := range chain {
			ret, ok := filter(fields)
			if !ok {
				// if this is the head of the chain, we continue to try the next chain
				// otherwise, we should stop and return false
				if i == 0 {
					found = false
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
		if found {
			return result, true
		}
	}
	return nil, false
}

func (self *FieldFilterChain) Eval(fields map[string]string) (map[string]string, bool) {
	var result map[string]string
	for _, filter := range *self {
		ret, ok := filter(fields)
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

func NewFieldInSetFilter(field string, elements ...string) FieldFilter {
	var set map[string]bool
	for _, ele := range elements {
		set[ele] = true
	}
	return func(fields map[string]string) (map[string]string, bool) {
		_, ok := set[fields[field]]
		return nil, ok
	}
}

func NewFieldNotEqualFilter(field string, match string) FieldFilter {
	return func(fields map[string]string) (map[string]string, bool) {
		return nil, match != fields[field]
	}
}
func NewFieldEqualsFilter(field string, match string) FieldFilter {
	return func(fields map[string]string) (map[string]string, bool) {
		return nil, match == fields[field]
	}
}

func NewFieldRegexFilter(field string, pattern string) FieldFilter {
	re := regexp.MustCompile(pattern)
	return func(fields map[string]string) (map[string]string, bool) {
		return nil, re.MatchString(fields[field])
	}
}

func NewFieldMRegexpMapFilter(field string, pattern string, prefix_group string) FieldFilter {
	re := &du.MRegexp{regexp.MustCompile(pattern)}
	return func(fields map[string]string) (map[string]string, bool) {
		match := re.FindStringSubmatchMap(fields[field], prefix_group)
		return match, len(match) != 0
	}
}

func NewFieldFilterTree(config *FieldFilterPatternConfig) (FieldFilterTree, error) {
	var tree FieldFilterTree
	for _, chain_config := range config.Chains {
		var chain FieldFilterChain
		for _, filter_config := range chain_config.Filters {
			var filter FieldFilter
			switch filter_config.Operator {
			case "==":
				filter = NewFieldEqualsFilter(filter_config.Field, filter_config.Pattern)
			case "!=":
				filter = NewFieldNotEqualFilter(filter_config.Field, filter_config.Pattern)
			case "~":
				if filter_config.CaptureResult {
					filter = NewFieldMRegexpMapFilter(filter_config.Field, filter_config.Pattern, filter_config.Field+"_")
				} else {
					filter = NewFieldRegexFilter(filter_config.Field, filter_config.Pattern)
				}
			case "(-":
				elements := strings.Split(filter_config.Pattern, ",")
				filter = NewFieldInSetFilter(filter_config.Field, elements...)
			default:
				return nil, fmt.Errorf("Unrecognized filter operator %s", filter_config.Operator)
			}
			chain = append(chain, filter)
		}
		tree = append(tree, chain)
	}
	return tree, nil
}
