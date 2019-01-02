package main

import (
	"encoding/json"
	"net/url"
)

type Profile struct {
	Server    url.URL
	ServerURL string
	Nick      string
	Chans     []string
	// Id is the way to reference this bot.
	Id          string
	Patterns    []Pattern
	PatternsRaw []Pattern
}

func UnmarshalProfile(b []byte) (*Profile, error) {
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	u, err := url.Parse(p.ServerURL)
	if err != nil {
		return nil, err
	}
	p.Server = *u
	return &p, nil
}
