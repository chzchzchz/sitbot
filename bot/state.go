package bot

import (
	"log"
	"strings"
	"sync"

	"gopkg.in/sorcix/irc.v2"
)

type State struct {
	Channels map[string]*room
	Users    map[string]*user

	mu sync.RWMutex
}

func NewState() *State {
	return &State{
		Channels: make(map[string]*room),
		Users:    make(map[string]*user),
	}
}

func (s *State) Process(msg irc.Message) error {
	log.Printf("state: %+v", msg)
	if len(msg.Params) < 1 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch msg.Command {
	case irc.JOIN:
		sender, room := msg.Prefix.Name, msg.Params[0]
		r := s.lookupRoom(room)
		r.Joined = true
		s.addModeUser(r, sender)
	case irc.RPL_TOPIC:
		room, topic := msg.Params[1], msg.Params[2]
		s.lookupRoom(room).Topic = topic
	case irc.RPL_NAMREPLY:
		room, users := msg.Params[2], msg.Params[3]
		if room[0] != '#' {
			log.Printf("not a channel: %q", room)
			break
		}
		r := s.lookupRoom(room)
		for _, u := range strings.Split(users, " ") {
			s.addModeUser(r, u)
		}
	case irc.NICK:
		sender, newnick := msg.Params[0], msg.Params[1]
		u, ok := s.Users[sender]
		if !ok {
			break
		}
		u.Nick = newnick
		for ch := range u.Channels {
			r, ok := s.Channels[ch]
			if !ok {
				continue
			}
			oldru, ok := r.Users[sender]
			if !ok {
				continue
			}
			delete(r.Users, sender)
			r.Users[newnick] = oldru
		}
	case irc.PART:
		sender, room := msg.Prefix.Name, msg.Params[0]
		u, ok := s.Users[sender]
		if !ok {
			break
		}
		delete(u.Channels, room)
		r, ok := s.Channels[room]
		if !ok {
			break
		}
		delete(r.Users, sender)
	}
	return nil
}

func (s *State) lookupRoom(rn string) *room {
	r, ok := s.Channels[rn]
	if ok {
		return r
	}
	r = &room{Name: rn, Users: make(map[string]roomUser)}
	s.Channels[rn] = r
	return r
}

func (s *State) addModeUser(r *room, u string) {
	umode, unick := u[:1], u
	if umode == "~" || umode == "@" || umode == "+" || umode == "=" || umode == "!" {
		unick = u[1:]
	} else {
		umode = ""
	}
	if len(unick) <= 1 {
		return
	}
	uptr, ok := s.Users[unick]
	if !ok {
		uptr = &user{Nick: unick, Channels: make(map[string]struct{})}
		s.Users[unick] = uptr
	}
	r.Users[unick] = roomUser{Mode: umode}
	uptr.Channels[r.Name] = struct{}{}
}

type room struct {
	Name   string
	Topic  string
	Joined bool
	Users  map[string]roomUser
}

type roomUser struct {
	Mode string `json:",omitempty"`
}

type user struct {
	Nick     string
	User     string `json:",omitempty"`
	Host     string `json:",omitempty"`
	Channels map[string]struct{}
}
