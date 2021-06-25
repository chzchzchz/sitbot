package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"sync"
)

var ErrNotFound = errors.New("not found")

type Gang struct {
	Bots    map[string]*Bot
	Botlets map[string]*Botlet
	mu      sync.RWMutex
}

func NewGang() *Gang { return &Gang{Bots: make(map[string]*Bot), Botlets: make(map[string]*Botlet)} }

func (g *Gang) LockBots() {
	g.mu.RLock()
	for _, b := range g.Bots {
		b.mu.RLock()
	}
}

func (g *Gang) UnlockBots() {
	for _, b := range g.Bots {
		b.mu.RUnlock()
	}
	g.mu.RUnlock()
}

func (g *Gang) Post(p Profile) error {
	source := p
	pats, rawpats, vars, err := g.ResolvePatterns(&p)
	if err != nil {
		return err
	}
	p.Patterns = pats
	p.PatternsRaw = rawpats
	p.Vars = vars
	if bot := g.Lookup(p.Id); bot != nil {
		return bot.update(p, source)
	}
	bot, err := NewBot(context.TODO(), p, source)
	if err != nil {
		return err
	}
	// Botlets are registered through PostBotlet; posting a profile only creates a bot.
	g.mu.Lock()
	ob := g.Bots[p.Id]
	g.Bots[p.Id] = bot
	g.mu.Unlock()
	if ob != nil {
		ob.Close()
	}
	return nil
}

func (g *Gang) PostBotlet(bl *Botlet) error {
	if bl == nil {
		return fmt.Errorf("nil botlet")
	}
	if bl.Name == "" {
		return fmt.Errorf("botlet name is empty")
	}
	candidate := *bl
	if err := validateBotlet(&candidate); err != nil {
		return err
	}
	g.mu.RLock()
	err := g.validateBotletDependentsLocked(&candidate)
	g.mu.RUnlock()
	if err != nil {
		return err
	}
	g.mu.Lock()
	g.Botlets[candidate.Name] = &candidate
	g.mu.Unlock()
	return g.refreshBotlet(candidate.Name, false)
}

// Rebuild dependents from their source profiles; Bot.Profile already contains resolved patterns.
func (g *Gang) refreshBotlet(name string, allowMissing bool) error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	// Gang entries are live bots; a nil result from Lookup is only a missing ID.
	for _, bot := range g.Bots {
		source := bot.source()
		if !profileReferencesBotlet(source, name) {
			continue
		}
		p := source
		pats, rawpats, vars, err := g.resolvePatternsLocked(&p, nil, allowMissing)
		if err != nil {
			return err
		}
		p.Patterns = pats
		p.PatternsRaw = rawpats
		p.Vars = vars
		if err := bot.update(p, source); err != nil {
			return err
		}
	}
	return nil
}

// Validate a candidate before publishing it so dependent bots cannot observe a broken catalog.
func (g *Gang) validateBotletDependentsLocked(candidate *Botlet) error {
	for _, bot := range g.Bots {
		source := bot.source()
		if !profileReferencesBotlet(source, candidate.Name) {
			continue
		}
		p := source
		pats, rawpats, vars, err := g.resolvePatternsLocked(&p, candidate, false)
		if err != nil {
			return err
		}
		if _, err := NewPatternMatcher(pats, vars); err != nil {
			return fmt.Errorf("bot %q: %w", source.Id, err)
		}
		if _, err := NewPatternMatcher(rawpats, vars); err != nil {
			return fmt.Errorf("bot %q: %w", source.Id, err)
		}
	}
	return nil
}

func validateBotlet(bl *Botlet) error {
	if _, err := NewPatternMatcher(bl.Patterns, bl.Vars); err != nil {
		return fmt.Errorf("invalid patterns: %w", err)
	}
	if _, err := NewPatternMatcher(bl.PatternsRaw, bl.Vars); err != nil {
		return fmt.Errorf("invalid raw patterns: %w", err)
	}
	return nil
}

func profileReferencesBotlet(p Profile, name string) bool {
	for _, ref := range p.Botlets {
		if ref.Name == name {
			return true
		}
	}
	return false
}

func (g *Gang) ResolvePatterns(p *Profile) ([]Pattern, []Pattern, map[string]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.resolvePatternsLocked(p, nil, false)
}

func (g *Gang) resolvePatternsLocked(p *Profile, candidate *Botlet, allowMissing bool) ([]Pattern, []Pattern, map[string]string, error) {
	vars := make(map[string]string)
	maps.Copy(vars, p.Vars)
	for _, bl := range p.Botlets {
		botlet := g.resolveBotletLocked(bl.Name, candidate)
		if botlet == nil {
			if !allowMissing {
				slog.Warn("referenced botlet not found", "profile", p.Id, "botlet", bl.Name)
				return nil, nil, nil, fmt.Errorf("botlet %q not found", bl.Name)
			}
			continue
		}
		maps.Copy(vars, botlet.Vars)
	}
	for _, br := range p.Botlets {
		botlet := g.resolveBotletLocked(br.Name, candidate)
		if botlet == nil {
			continue
		}
		maps.Copy(vars, br.Vars)
	}
	var allPats, allRawPats []Pattern
	allPats = append(allPats, p.Patterns...)
	allRawPats = append(allRawPats, p.PatternsRaw...)
	for _, bl := range p.Botlets {
		botlet := g.resolveBotletLocked(bl.Name, candidate)
		if botlet == nil {
			continue
		}
		allPats = append(allPats, botlet.Patterns...)
		allRawPats = append(allRawPats, botlet.PatternsRaw...)
	}
	return allPats, allRawPats, vars, nil
}

func (g *Gang) resolveBotletLocked(name string, candidate *Botlet) *Botlet {
	if candidate != nil && candidate.Name == name {
		return candidate
	}
	return g.Botlets[name]
}

func (g *Gang) DeleteBot(id string) error {
	g.mu.Lock()
	b, ok := g.Bots[id]
	if ok {
		delete(g.Bots, id)
	}
	g.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s does not exist", ErrNotFound, id)
	}
	b.Close()
	return nil
}

func (g *Gang) DeleteBotlet(id string) error {
	g.mu.Lock()
	_, ok := g.Botlets[id]
	if ok {
		delete(g.Botlets, id)
	}
	g.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: %s does not exist", ErrNotFound, id)
	}
	return g.refreshBotlet(id, true)
}

func (g *Gang) Lookup(id string) *Bot {
	defer g.mu.RUnlock()
	g.mu.RLock()
	return g.Bots[id]
}

func (g *Gang) LookupBotlet(id string) *Botlet {
	defer g.mu.RUnlock()
	g.mu.RLock()
	return g.Botlets[id]
}
