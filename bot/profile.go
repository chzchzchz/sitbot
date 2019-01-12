package bot

import (
	"context"
	"encoding/json"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

type Profile struct {
	ServerURL string
	ProxyURL  string
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
	return &p, nil
}

type ctxDialer struct {
	ctx context.Context
	fwd net.Dialer
}

func (d *ctxDialer) Dial(network, address string) (net.Conn, error) {
	return d.fwd.DialContext(d.ctx, network, address)
}

func (p *Profile) Dial(ctx context.Context) (c net.Conn, err error) {
	servURL, err := url.Parse(p.ServerURL)
	if err != nil {
		return nil, err
	}
	fwd := &ctxDialer{ctx: ctx}
	var dialer proxy.Dialer
	dialer = fwd
	proxyURL, err := url.Parse(p.ProxyURL)
	if len(p.ProxyURL) != 0 && err != nil {
		return nil, err
	} else if len(p.ProxyURL) != 0 {
		if dialer, err = proxy.FromURL(proxyURL, fwd); err != nil {
			return nil, err
		}
	}
	return dialer.Dial("tcp", servURL.Host)
}
