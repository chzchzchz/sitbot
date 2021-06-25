package bot

import (
	"regexp"
	"strings"
)

type Pattern struct {
	Match    string
	Template string
}

type PatternMatcher struct {
	re   []*regexp.Regexp
	tmpl [][]byte
}

// Resolve placeholders in one pass so map iteration cannot change results.
func resolveVars(value string, vars map[string]string) string {
	if len(vars) == 0 {
		return value
	}
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); {
		if i+3 < len(value) && value[i] == '{' && value[i+1] == '{' {
			end := strings.Index(value[i+2:], "}}")
			if end >= 0 {
				key := value[i+2 : i+2+end]
				if replacement, ok := vars[key]; ok {
					b.WriteString(replacement)
					i += end + 4
					continue
				}
			}
		}
		b.WriteByte(value[i])
		i++
	}
	return b.String()
}

func NewPatternMatcher(pats []Pattern, vars map[string]string) (*PatternMatcher, error) {
	re := make([]*regexp.Regexp, len(pats))
	tmpl := make([][]byte, len(pats))
	for i, pat := range pats {
		resolved := resolveVars(pat.Match, vars)
		r, err := regexp.Compile(resolved)
		if err != nil {
			return nil, err
		}
		re[i] = r
		tmpl[i] = []byte(resolveVars(pat.Template, vars))
	}
	return &PatternMatcher{re, tmpl}, nil
}

func (pm *PatternMatcher) Apply(txt string) string {
	if len(txt) == 0 {
		return ""
	}
	txtb := []byte(txt)
	for i, re := range pm.re {
		if match := re.FindSubmatchIndex(txtb); match != nil {
			return string(re.Expand(nil, pm.tmpl[i], txtb, match))
		}
	}
	return ""
}
