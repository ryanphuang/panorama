package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pb "panorama/build/gen"
	du "panorama/util"
)

type FieldFilter func(fields map[string]string) (map[string]string, bool)

type FieldClassifierResult struct {
	Context string
	Subject string
	Status  pb.Status
	Score   float32
}

type FieldClassifier func(result map[string]string) *FieldClassifierResult

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
	subject := config.Subject
	index_subject := false
	if len(subject) != 0 && subject[0] == '<' && subject[len(subject)-1] == '>' {
		subject = subject[1 : len(subject)-1]
		index_subject = true
	}
	context := config.Context
	return func(result map[string]string) *FieldClassifierResult {
		if index_subject {
			return &FieldClassifierResult{context, result[subject], status, score32}
		}
		return &FieldClassifierResult{context, subject, status, score32}
	}, nil
}

func NewFieldFilterTree(config *FieldFilterTreeConfig) (FieldFilterTree, error) {
	var tree FieldFilterTree
	branches := make(map[FieldFilterClauseConfig]*FieldFilterBranch)
	for _, chain_config := range config.FilterTree {
		if len(chain_config.Chain) == 0 {
			return nil, fmt.Errorf("Empty chain config")
		}
		head := chain_config.Chain[0]
		branch, ok := branches[*head]
		if !ok {
			branch = new(FieldFilterBranch)
			filter, err := NewFieldFilter(head)
			if err != nil {
				return nil, err
			}
			branch.Head = filter
			branches[*head] = branch
			tree = append(tree, branch)
		}
		var chain []FieldFilter
		for i := 1; i < len(chain_config.Chain); i++ {
			filter, err := NewFieldFilter(chain_config.Chain[i])
			if err != nil {
				return nil, err
			}
			chain = append(chain, filter)
		}
		classifier, err := NewFieldClassifier(&chain_config.Classifier)
		if err != nil {
			return nil, err
		}
		branch.Bodies = append(branch.Bodies, &FieldFilterBody{chain, classifier})
	}
	return tree, nil
}
