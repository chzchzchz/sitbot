package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVars(t *testing.T) {
	tts := []struct {
		match string
		vars  map[string]string
		want  string
	}{
		{"hello", nil, "hello"},
		{"hello", map[string]string{}, "hello"},
		{"{{nick}} is here", map[string]string{"nick": "sitbot"}, "sitbot is here"},
		{"{{a}} {{b}}", map[string]string{"a": "foo", "b": "bar"}, "foo bar"},
		{"no vars here", map[string]string{"nick": "x"}, "no vars here"},
		{"{{nick}} said {{msg}}", map[string]string{"nick": "alice", "msg": "hi"}, "alice said hi"},
		{"{{a}}{{b}}", map[string]string{"a": "1", "b": "2"}, "12"},
		// overlapping keys: longer key should not interfere with shorter
		{"{{ab}} {{a}}", map[string]string{"a": "X", "ab": "Y"}, "Y X"},
	}
	for _, tt := range tts {
		got := resolveVars(tt.match, tt.vars)
		assert.Equal(t, tt.want, got, "resolveVars(%q, %v)", tt.match, tt.vars)
	}
}

func TestNewPatternMatcher_InvalidRegex(t *testing.T) {
	_, err := NewPatternMatcher([]Pattern{{Match: "[invalid", Template: "x"}}, nil)
	assert.Error(t, err)
}

func TestNewPatternMatcher_Valid(t *testing.T) {
	pm, err := NewPatternMatcher([]Pattern{{Match: "hello", Template: "world"}}, nil)
	require.NoError(t, err)
	assert.NotNil(t, pm)
	assert.Len(t, pm.re, 1)
}

func TestPatternMatcher_Apply(t *testing.T) {
	tts := []struct {
		name string
		pats []Pattern
		vars map[string]string
		txt  string
		want string
	}{
		{
			name: "no match",
			pats: []Pattern{{Match: `^hello$`, Template: "world"}},
			txt:  "goodbye",
			want: "",
		},
		{
			name: "simple match",
			pats: []Pattern{{Match: `^hello$`, Template: "world"}},
			txt:  "hello",
			want: "world",
		},
		{
			name: "backreference $1",
			pats: []Pattern{{Match: `(\w+) is here`, Template: "$1 says hi"}},
			txt:  "sitbot is here",
			want: "sitbot says hi",
		},
		{
			name: "multiple patterns tried in order",
			pats: []Pattern{
				{Match: `^foo$`, Template: "matched foo"},
				{Match: `^bar$`, Template: "matched bar"},
			},
			txt:  "bar",
			want: "matched bar",
		},
		{
			name: "first pattern wins",
			pats: []Pattern{
				{Match: `.*`, Template: "first"},
				{Match: `.*`, Template: "second"},
			},
			txt:  "anything",
			want: "first",
		},
		{
			name: "empty input returns empty",
			pats: []Pattern{{Match: `.*`, Template: "x"}},
			txt:  "",
			want: "",
		},
		{
			name: "var interpolation in match",
			pats: []Pattern{{Match: `{{prefix}} (\w+)`, Template: "$1"}},
			vars: map[string]string{"prefix": "hello"},
			txt:  "hello world",
			want: "world",
		},
		{
			name: "no patterns",
			pats: []Pattern{},
			txt:  "anything",
			want: "",
		},
	}
	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := NewPatternMatcher(tt.pats, tt.vars)
			require.NoError(t, err)
			got := pm.Apply(tt.txt)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPatternMatcher_Apply_BackreferenceExpand(t *testing.T) {
	pm, err := NewPatternMatcher(
		[]Pattern{{Match: `(\w+) (\w+)`, Template: "$2, $1"}},
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, "Smith, John", pm.Apply("John Smith"))
}

func TestPatternMatcher_Apply_MultipleSubmatches(t *testing.T) {
	pm, err := NewPatternMatcher(
		[]Pattern{{Match: `a(b)`, Template: "$1"}},
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, "b", pm.Apply("ab"))
}

func TestPatternMatcher_StructFields(t *testing.T) {
	pm, err := NewPatternMatcher([]Pattern{{Match: `a`, Template: "1"}}, nil)
	require.NoError(t, err)
	assert.Len(t, pm.re, 1)
	assert.Equal(t, "1", string(pm.tmpl[0]))
}

func TestPatternMatcher_InterpolatesTemplate(t *testing.T) {
	pm, err := NewPatternMatcher(
		[]Pattern{{Match: `^INVITE$`, Template: "join {{channel}}"}},
		map[string]string{"channel": "#gamme"},
	)
	require.NoError(t, err)
	assert.Equal(t, "join #gamme", pm.Apply("INVITE"))
}

func TestResolveVars_IsDeterministic(t *testing.T) {
	vars := map[string]string{"a": "{{b}}", "b": "value"}
	for i := 0; i < 100; i++ {
		assert.Equal(t, "{{b}}", resolveVars("{{a}}", vars))
	}
}
