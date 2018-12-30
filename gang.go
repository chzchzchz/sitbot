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
	g.mu.Lock()
	_, ok := g.bots[p.Id]
	g.mu.Unlock()
	if ok {
		return nil
	}
	bot, err := NewBot(context.TODO(), p)
	if err != nil {
		return err
	}
	g.mu.Lock()
	if _, ok := g.bots[p.Id]; ok {
		err = fmt.Errorf("%s exists", p.Id)
	} else {
		g.bots[p.Id] = bot
	}
	g.mu.Unlock()
	return err
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
