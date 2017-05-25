package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pb "deephealth/build/gen"
	du "deephealth/util"
)

type FieldFilter func(fields map[string]string) (map[string]string, bool)

type FieldClassifier func(result map[string]string) (pb.Status, float32)

type FieldFilterBody struct {
	Chain      []FieldFilter
	Classifier FieldClassifier
}

type FieldFilterBranch struct {
	Head   FieldFilter
	Bodies []*FieldFilterBody
}

type FieldFilterTree []*FieldFilterBranch

func (self *FieldFilterTree) Eval(fields map[string]string) (map[string]string, FieldClassifier, bool) {
	for _, branch := range *self {
		result, ok := branch.Head(fields)
		// if this is the head of the branch, we continue to try the next branch
		// otherwise, we should will return whatever result it is
		if !ok {
			continue
		}
		for _, body := range branch.Bodies {
			found := true
			for _, filter := range body.Chain {
				ret, ok := filter(fields)
				if !ok {
					found = false
					break
				} else {
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
			}
			if found {
				return result, body.Classifier, true
			}
		}
		return nil, nil, false
	}
	return nil, nil, false
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

func StringArrayToSlice(array string) ([]string, error) {
	e := len(array) - 1
	if array[0] != '[' || array[e] != ']' {
		return nil, fmt.Errorf("Array must be enclosed in [ and ]")
	}
	parts := strings.Split(array[1:e], "',")
	result := make([]string, len(parts))
	for i, part := range parts {
		part = strings.TrimLeft(part, " ")
		if part[0] != '\'' {
			return nil, fmt.Errorf("Element must start with ': %s", part)
		}
		if i == len(parts)-1 {
			if part[len(part)-1] != '\'' {
				return nil, fmt.Errorf("Element must end with': %s", part)
			}
			result[i] = part[1 : len(part)-1]
		} else {
			result[i] = part[1:]
		}
	}
	return result, nil
}

func NewFieldRegexAnyFilter(field string, patterns ...string) FieldFilter {
	res := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		res[i] = regexp.MustCompile(pattern)
	}
	return func(fields map[string]string) (map[string]string, bool) {
		for _, re := range res {
			if re.MatchString(fields[field]) {
				return nil, true
			}
		}
		return nil, false
	}
}

func NewFieldMRegexpMapAnyFilter(field string, prefix_group string, patterns ...string) FieldFilter {
	res := make([]*du.MRegexp, len(patterns))
	for i, pattern := range patterns {
		res[i] = &du.MRegexp{regexp.MustCompile(pattern)}
	}
	return func(fields map[string]string) (map[string]string, bool) {
		for _, re := range res {
			match := re.FindStringSubmatchMap(fields[field], prefix_group)
			if len(match) != 0 {
				return match, true
			}
		}
		return nil, false
	}
}

func NewFieldRegexFilter(field string, pattern string) FieldFilter {
	re := regexp.MustCompile(pattern)
	return func(fields map[string]string) (map[string]string, bool) {
		return nil, re.MatchString(fields[field])
	}
}

func NewFieldMRegexpMapFilter(field string, prefix_group string, pattern string) FieldFilter {
	re := &du.MRegexp{regexp.MustCompile(pattern)}
	return func(fields map[string]string) (map[string]string, bool) {
		match := re.FindStringSubmatchMap(fields[field], prefix_group)
		return match, len(match) != 0
	}
}

func NewFieldFilter(config *FieldFilterClauseConfig) (FieldFilter, error) {
	var filter FieldFilter
	switch config.Operator {
	case "==":
		filter = NewFieldEqualsFilter(config.Field, config.Pattern)
	case "!=":
		filter = NewFieldNotEqualFilter(config.Field, config.Pattern)
	case "~":
		if config.CaptureResult {
			filter = NewFieldMRegexpMapFilter(config.Field, config.Field+"_", config.Pattern)
		} else {
			filter = NewFieldRegexFilter(config.Field, config.Pattern)
		}
	case "(-":
		elements, err := StringArrayToSlice(config.Pattern)
		if err != nil {
			return nil, err
		}
		filter = NewFieldInSetFilter(config.Field, elements...)
	case "(~":
		patterns, err := StringArrayToSlice(config.Pattern)
		if err != nil {
			return nil, err
		}
		fmt.Println(patterns)
		if config.CaptureResult {
			filter = NewFieldMRegexpMapAnyFilter(config.Field, config.Field+"_", patterns...)
		} else {
			filter = NewFieldRegexAnyFilter(config.Field, patterns...)
		}
	default:
		return nil, fmt.Errorf("Unrecognized filter operator %s", config.Operator)
	}
	return filter, nil
}

func NewFieldClassifier(config *ClassifierConfig) (FieldClassifier, error) {
	status := StatusFromFullStr(config.Status)
	if status == pb.Status_INVALID {
		return nil, fmt.Errorf("Invalid status string: %s", config.Status)
	}
	score, err := strconv.ParseFloat(config.Score, 32)
	if err != nil {
		return nil, err
	}
	score32 := float32(score)
	return func(result map[string]string) (pb.Status, float32) {
		return status, score32
	}, nil
}

func NewFieldFilterTree(config *FieldFilterTreeConfig) (FieldFilterTree, error) {
	var tree FieldFilterTree
	for _, branch_config := range config.FilterTree {
		head, err := NewFieldFilter(branch_config.Head)
		if err != nil {
			return nil, err
		}
		var bodies []*FieldFilterBody
		for _, chain_config := range branch_config.Bodies {
			classifier, err := NewFieldClassifier(&chain_config.Classifier)
			if err != nil {
				return nil, err
			}
			var chain []FieldFilter
			for _, filter_config := range chain_config.Chain {
				filter, err := NewFieldFilter(filter_config)
				if err != nil {
					return nil, err
				}
				chain = append(chain, filter)
			}
			bodies = append(bodies, &FieldFilterBody{chain, classifier})
		}
		tree = append(tree, &FieldFilterBranch{head, bodies})
	}
	return tree, nil
}
