package runtime

import (
	"encoding/gob"
	"log"
	"os"
	"strconv"
	"strings"
)

type Variables struct {
	Globals map[string]string
	locals  map[string]string
}

var mslVar Variables
var mslEv EventContext

type EventContext struct {
	Level    string
	Location string
	Nick     string
	Chan     string
	Msg      string
}

func init() {
	mslVar.Globals = make(map[string]string)
	mslVar.locals = make(map[string]string)
}

func (m *Variables) Load(fname string) {
	inf, err := os.Open(fname)
	if err != nil {
		log.Printf("got error %v on file %q", err, fname)
	} else {
		dec := gob.NewDecoder(inf)
		dec.Decode(m)
	}
	inf.Close()
	log.Printf("loaded %+v", m)
}

func (m *Variables) Save(fname string) {
	outf, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer outf.Close()
	enc := gob.NewEncoder(outf)
	if err := enc.Encode(m); err != nil {
		panic(err)
	}
	log.Printf("saved %+v", m)
}

func (m *Variables) SetLocal(name, val string) {
	mustGoodName(name)
	m.locals[name] = val
}

func (m *Variables) SetGlobal(name, val string) {
	mustGoodName(name)
	m.Globals[name] = val
}

func (m *Variables) SetOverride(name, val string) {
	mustGoodName(name)
	if _, ok := m.locals[name]; ok {
		m.locals[name] = val
	} else if _, ok := m.Globals[name]; ok {
		m.Globals[name] = val
	} else {
		m.Globals[name] = val
	}
}

func (m *Variables) Lookup(name string) (string, bool) {
	if v, ok := m.Globals[name]; ok {
		return v, true
	} else if v, ok := m.locals[name]; ok {
		return v, true
	}
	return "", false
}

func mustGoodName(s string) {
	if strings.Contains(s, "%") {
		panic("should not have % in name:" + s)
	}
	if s == "" {
		panic("variable must have name")
	}
	if _, err := strconv.Atoi(s); err == nil {
		panic("expected non-numeric name")
	}
}
