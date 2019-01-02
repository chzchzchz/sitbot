package bot

import (
	"regexp"
)

type Pattern struct {
	Match    string
	Template string
}

type PatternMatcher struct {
	re   []*regexp.Regexp
	tmpl [][]byte
}

func NewPatternMatcher(pats []Pattern) (*PatternMatcher, error) {
	re := make([]*regexp.Regexp, len(pats))
	tmpl := make([][]byte, len(pats))
	for i, pat := range pats {
		r, err := regexp.Compile(pat.Match)
		if err != nil {
			return nil, err
		}
		re[i] = r
		tmpl[i] = []byte(pats[i].Template)
	}
	return &PatternMatcher{re, tmpl}, nil
}

func (pm *PatternMatcher) Apply(txt string) string {
	if len(txt) == 0 {
		return ""
	}
	txtb := []byte(txt)
	for i, re := range pm.re {
		if si := re.FindAllSubmatchIndex(txtb, 1); len(si) != 0 {
			res := []byte{}
			for _, submatches := range si {
				res = re.Expand(res, pm.tmpl[i], txtb, submatches)
			}
			return string(res)
		}
	}
	return ""
}
