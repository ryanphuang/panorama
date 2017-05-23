package util

import (
	"regexp"
)

type MRegexp struct {
	*regexp.Regexp
}

type MRegexpPair struct {
	First, Second *MRegexp
}

type MRegexpMatch map[string]string

type MRegexpMap map[string]*MRegexp
type MRegexpPairList []*MRegexpPair

type MPatternMix struct {
	ReList MRegexpPairList
}

func (r *MRegexp) FindStringSubmatchMap(s string) MRegexpMatch {
	groups := make(map[string]string)
	result := r.FindStringSubmatch(s)
	if result == nil {
		return groups
	}
	for i, name := range r.SubexpNames() {
		if i == 0 || name == "" {
			// to distinguish between non-match
			// and match with no subgroup
			groups["_all_"] = result[i]
			continue
		}
		groups[name] = result[i]
	}
	return groups
}

func NewMRegexpMap(patterns map[string]string) MRegexpMap {
	m := make(MRegexpMap)
	for key, val := range patterns {
		if len(val) == 0 {
			m[key] = nil
		} else {
			m[key] = &MRegexp{regexp.MustCompile(val)}
		}
	}
	return m
}

func NewMPatternMix(patterns map[string]string) *MPatternMix {
	var l MRegexpPairList
	for key, val := range patterns {
		var pair MRegexpPair
		pair.First = &MRegexp{regexp.MustCompile(key)}
		if len(val) == 0 {
			pair.Second = nil
		} else {
			pair.Second = &MRegexp{regexp.MustCompile(val)}
		}
		l = append(l, &pair)
	}
	return &MPatternMix{l}
}

func (p *MPatternMix) IsMatch(key string, value string) bool {
	for _, pair := range p.ReList {
		if pair.First.MatchString(key) {
			if pair.Second != nil {
				return pair.Second.MatchString(value)
			} else {
				return true
			}
		}
	}
	return false
}
