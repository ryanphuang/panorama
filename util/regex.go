package util

import (
	"regexp"
)

type MRegexp struct {
	*regexp.Regexp
}

type MRegexpMatch map[string]string

type MRegexpMap map[string]*MRegexp

func (r *MRegexp) FindStringSubmatchMap(s string) MRegexpMatch {
	groups := make(map[string]string)
	result := r.FindStringSubmatch(s)
	if result == nil {
		return groups
	}
	for i, name := range r.SubexpNames() {
		if i == 0 || name == "" {
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
