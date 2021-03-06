package bot

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

const defaultRateMs = 1000

type Profile struct {
	ProfileLogin
	Chans     []string
	RateMs    int
	Verbosity int

	// Id is the way to reference this bot.
	Id          string
	Patterns    []Pattern
	PatternsRaw []Pattern
}

type ProfileLogin struct {
	ServerURL string
	ProxyURL  string `json:",omitempty"`
	Nick      string
	User      string
	Pass      string `json:",omitempty"`
}

func DecodeProfiles(r io.Reader) (ret []*Profile, err error) {
	d := json.NewDecoder(r)
	for {
		p := &Profile{}
		if err = d.Decode(p); err != nil {
			if err == io.EOF {
				return ret, nil
			}
			return nil, err
		}
		if p.RateMs == 0 {
			p.RateMs = defaultRateMs
		}
		ret = append(ret, p)
	}
}

func UnmarshalProfile(b []byte) (*Profile, error) {
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	if p.RateMs == 0 {
		p.RateMs = defaultRateMs
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
