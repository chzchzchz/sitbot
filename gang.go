package main

import (
	"context"
	"fmt"
	"sync"
)

type Gang struct {
	bots map[string]*Bot
	mu   sync.Mutex
}

func NewGang() *Gang {
	return &Gang{bots: make(map[string]*Bot)}
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
	ob := g.bots[p.Id]
	g.bots[p.Id] = bot
	g.mu.Unlock()
	if ob != nil {
		ob.Close()
	}
	return nil
}

func (g *Gang) Delete(id string) error {
	g.mu.Lock()
	b, ok := g.bots[id]
	if ok {
		delete(g.bots, id)
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
	return g.bots[id]
}

func (g *Gang) Close() {
	for _, b := range g.bots {
		b.Close()
	}
	g.bots = nil
}
