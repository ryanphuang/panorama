package types

import (
	"regexp"
)

type Filter func(part string) bool
type FilterChain []Filter

func (self *FilterChain) Eval(part string) bool {
	for _, filter := range *self {
		if filter(part) {
			return true
		}
	}
	return false
}

func NegateFilter(filter Filter) Filter {
	return func(part string) bool {
		return !filter(part)
	}
}

func NewInSetFilter(elements ...string) Filter {
	var set map[string]bool
	for _, ele := range elements {
		set[ele] = true
	}
	return func(part string) bool {
		_, ok := set[part]
		return ok
	}
}

func NewRegexFilter(re *regexp.Regexp) Filter {
	return func(part string) bool {
		return re.MatchString(part)
	}
}

func NewEqualsFilter(match string) Filter {
	return func(part string) bool {
		return match == part
	}
}
