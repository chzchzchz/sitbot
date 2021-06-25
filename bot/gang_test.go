package bot

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePatterns_BotletLookup(t *testing.T) {
	g := NewGang()
	// Store a botlet under its Name
	g.Botlets["mybot"] = &Botlet{
		Name:     "mybot",
		Vars:     map[string]string{"nick": "sitbot"},
		Patterns: []Pattern{{Match: `hello`, Template: "hi"}},
	}

	// Profile references the botlet by Name but has a different Id
	p := Profile{
		Botlet:  Botlet{Name: "mybot"},
		Id:      "otherid",
		Botlets: []BotletRef{{Name: "mybot"}},
	}

	pats, rawpats, vars, err := g.ResolvePatterns(&p)
	require.NoError(t, err)
	assert.Len(t, pats, 1)
	assert.Len(t, rawpats, 0)
	assert.Equal(t, "sitbot", vars["nick"])
	assert.Equal(t, "hi", string(pats[0].Template))
}

func TestResolvePatterns_BotletNotExists(t *testing.T) {
	g := NewGang()
	// Profile references a botlet that doesn't exist in g.Botlets
	p := Profile{
		Botlets: []BotletRef{{Name: "nonexistent"}},
	}

	pats, rawpats, vars, err := g.ResolvePatterns(&p)
	assert.Error(t, err)
	assert.Len(t, pats, 0)
	assert.Len(t, rawpats, 0)
	assert.Empty(t, vars)
}

func TestResolvePatterns_VarLayering(t *testing.T) {
	g := NewGang()
	g.Botlets["botlet1"] = &Botlet{
		Name: "botlet1",
		Vars: map[string]string{"a": "frombotlet"},
	}

	p := Profile{
		Botlet: Botlet{Name: "botlet1", Vars: map[string]string{"a": "fromprofile", "b": "fromprofile"}},
		Botlets: []BotletRef{
			{Name: "botlet1", Vars: map[string]string{"a": "fromref"}},
		},
	}

	_, _, vars, err := g.ResolvePatterns(&p)
	require.NoError(t, err)
	// Profile vars < botlet vars < botlet-ref vars
	assert.Equal(t, "fromref", vars["a"])
	assert.Equal(t, "fromprofile", vars["b"])
}

func TestGang_PostStoresBotletUnderName(t *testing.T) {
	g := NewGang()
	g.Botlets["mybot"] = &Botlet{
		Name:     "mybot",
		Vars:     map[string]string{"nick": "sitbot"},
		Patterns: []Pattern{{Match: `hello`, Template: "hi"}},
	}

	// Verify PostBotlet stores under bl.Name
	bl := &Botlet{Name: "mybot", Vars: map[string]string{"nick": "updated"}}
	err := g.PostBotlet(bl)
	require.NoError(t, err)

	// The botlet should be retrievable by Name
	botlet := g.LookupBotlet("mybot")
	require.NotNil(t, botlet)
	assert.Equal(t, "updated", botlet.Vars["nick"])
}

func TestGang_DeleteBotletRemovesBotlet(t *testing.T) {
	g := NewGang()
	g.Botlets["mybot"] = &Botlet{Name: "mybot"}

	err := g.DeleteBotlet("mybot")
	require.NoError(t, err)
	assert.Nil(t, g.LookupBotlet("mybot"))
}

func TestGang_DeleteBotNotFound(t *testing.T) {
	g := NewGang()
	err := g.DeleteBot("nonexistent")
	assert.Error(t, err)
}

func newConfiguredBot(g *Gang, source, resolved Profile) *Bot {
	b := &Bot{}
	b.dispatcher = NewDispatcher(&b.Profile, nil)
	if err := b.update(resolved, source); err != nil {
		panic(err)
	}
	g.Bots[source.Id] = b
	return b
}

func newResolvedBot(t *testing.T, g *Gang, source Profile) *Bot {
	t.Helper()
	pats, rawpats, vars, err := g.ResolvePatterns(&source)
	require.NoError(t, err)
	resolved := source
	resolved.Patterns = pats
	resolved.PatternsRaw = rawpats
	resolved.Vars = vars
	return newConfiguredBot(g, source, resolved)
}

func TestPostBotlet_RebuildsFromSourceProfile(t *testing.T) {
	g := NewGang()
	g.Botlets["greet"] = &Botlet{
		Name:     "greet",
		Patterns: []Pattern{{Match: `^old$`, Template: "old"}},
	}
	source := Profile{
		Id: "bot",
		Botlet: Botlet{Patterns: []Pattern{
			{Match: `^own$`, Template: "own"},
		}},
		Botlets: []BotletRef{{Name: "greet"}},
	}
	b := newResolvedBot(t, g, source)

	err := g.PostBotlet(&Botlet{
		Name:     "greet",
		Patterns: []Pattern{{Match: `^new$`, Template: "new"}},
	})
	require.NoError(t, err)
	assert.Len(t, b.Profile.Patterns, 2)
	assert.Equal(t, "own", b.dispatcher.pm.Apply("own"))
	assert.Equal(t, "new", b.dispatcher.pm.Apply("new"))
	assert.Empty(t, b.dispatcher.pm.Apply("old"))
}

func TestDeleteBotlet_RemovesDependentPatterns(t *testing.T) {
	g := NewGang()
	g.Botlets["greet"] = &Botlet{
		Name:     "greet",
		Patterns: []Pattern{{Match: `^greet$`, Template: "greet"}},
	}
	source := Profile{
		Id: "bot",
		Botlet: Botlet{Patterns: []Pattern{
			{Match: `^own$`, Template: "own"},
		}},
		Botlets: []BotletRef{{Name: "greet"}},
	}
	b := newResolvedBot(t, g, source)

	require.NoError(t, g.DeleteBotlet("greet"))
	assert.Nil(t, g.LookupBotlet("greet"))
	assert.Equal(t, "own", b.dispatcher.pm.Apply("own"))
	assert.Empty(t, b.dispatcher.pm.Apply("greet"))
}

func TestPostBotlet_ReportsUpdateErrors(t *testing.T) {
	g := NewGang()
	g.Botlets["greet"] = &Botlet{
		Name:     "greet",
		Patterns: []Pattern{{Match: `^old$`, Template: "old"}},
	}
	source := Profile{
		Id:      "bot",
		Botlets: []BotletRef{{Name: "greet"}},
	}
	b := newResolvedBot(t, g, source)

	err := g.PostBotlet(&Botlet{
		Name:     "greet",
		Patterns: []Pattern{{Match: `[`, Template: "invalid"}},
	})
	assert.Error(t, err)
	assert.Equal(t, "old", b.dispatcher.pm.Apply("old"))
}

func TestGang_DeleteBotLeavesBotlet(t *testing.T) {
	g := NewGang()
	b := &Bot{cancel: func() {}}
	g.Bots["shared"] = b
	g.Botlets["shared"] = &Botlet{Name: "shared"}

	require.NoError(t, g.DeleteBot("shared"))
	assert.Nil(t, g.Lookup("shared"))
	assert.NotNil(t, g.LookupBotlet("shared"))
}

func TestBotUpdate_FailureKeepsProfileAndMatcher(t *testing.T) {
	old := Profile{
		Botlet: Botlet{
			Name:     "old",
			Patterns: []Pattern{{Match: `^old$`, Template: "old"}},
		},
	}
	b := &Bot{Profile: old}
	b.dispatcher = NewDispatcher(&b.Profile, nil)
	require.NoError(t, b.update(old, old))

	next := Profile{
		Botlet: Botlet{
			Name:     "new",
			Patterns: []Pattern{{Match: `[`, Template: "new"}},
		},
	}
	err := b.update(next, next)
	assert.Error(t, err)
	assert.Equal(t, old.Patterns, b.Profile.Patterns)
	assert.Equal(t, "old", b.dispatcher.pm.Apply("old"))
}

func TestPostBotlet_RejectsEmptyName(t *testing.T) {
	g := NewGang()
	assert.Error(t, g.PostBotlet(&Botlet{}))
}

func TestPostBotlet_RejectsInvalidPatterns(t *testing.T) {
	g := NewGang()
	err := g.PostBotlet(&Botlet{
		Name:     "bad",
		Patterns: []Pattern{{Match: `[`, Template: "x"}},
	})
	assert.Error(t, err)
	assert.Nil(t, g.LookupBotlet("bad"))
}

func TestPostBotlet_ValidatesDependentOverridesBeforeStore(t *testing.T) {
	g := NewGang()
	old := &Botlet{
		Name:     "greet",
		Vars:     map[string]string{"bad": "ok"},
		Patterns: []Pattern{{Match: `^old$`, Template: "old"}},
	}
	g.Botlets["greet"] = old
	source := Profile{
		Id: "bot",
		Botlets: []BotletRef{{
			Name: "greet",
			Vars: map[string]string{"bad": "["},
		}},
	}
	newResolvedBot(t, g, source)

	err := g.PostBotlet(&Botlet{
		Name:     "greet",
		Vars:     map[string]string{"bad": "ok"},
		Patterns: []Pattern{{Match: `^{{bad}}$`, Template: "bad"}},
	})
	assert.Error(t, err)
	assert.Same(t, old, g.LookupBotlet("greet"))
}

func TestPost_FailsOnMissingBotlet(t *testing.T) {
	g := NewGang()
	err := g.Post(Profile{
		Id:      "bot",
		Botlets: []BotletRef{{Name: "missing"}},
	})
	assert.Error(t, err)
	assert.Nil(t, g.Lookup("bot"))
}

func TestResolvePatterns_LogsMissingBotlet(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	g := NewGang()
	p := Profile{
		Id:      "bot",
		Botlets: []BotletRef{{Name: "missing"}},
	}
	_, _, _, err := g.ResolvePatterns(&p)
	assert.Error(t, err)
	assert.Contains(t, logs.String(), "referenced botlet not found")
	assert.Contains(t, logs.String(), "botlet=missing")
}
