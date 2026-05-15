package flags

import "strings"

type StringList []string

func (s *StringList) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *StringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}
