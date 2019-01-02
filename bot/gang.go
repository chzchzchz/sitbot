package bot

import (
	"context"
	"fmt"
	"sync"
)

type Gang struct {
	Bots map[string]*Bot
	mu   sync.RWMutex
}

func NewGang() *Gang { return &Gang{Bots: make(map[string]*Bot)} }

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
	if bot := g.Lookup(p.Id); bot != nil {
		return bot.Update(p)
	}
	bot, err := NewBot(context.TODO(), p)
	if err != nil {
		return err
	}
	g.mu.Lock()
	ob := g.Bots[p.Id]
	g.Bots[p.Id] = bot
	g.mu.Unlock()
	if ob != nil {
		ob.Close()
	}
	return nil
}

func (g *Gang) Delete(id string) error {
	g.mu.Lock()
	b, ok := g.Bots[id]
	if ok {
		delete(g.Bots, id)
	}
	g.mu.Unlock()
	if !ok {
		return fmt.Errorf("%s does not exist", id)
	}
	b.Close()
	return nil
}

func (g *Gang) Lookup(id string) *Bot {
	defer g.mu.Unlock()
	g.mu.Lock()
	return g.Bots[id]
}
