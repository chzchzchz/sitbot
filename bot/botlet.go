package bot

import (
	"encoding/json"
)

type Botlet struct {
	// Name is the way to reference a botlet / bot.
	Name        string `json:"Name"`
	Vars        map[string]string
	Patterns    []Pattern
	PatternsRaw []Pattern
}

// BotletRef represents a botlet reference in a Profile's Botlets list.
// JSON supports both string form ("greet") and object form
// ({"Name":"greet","Vars":{"nick":"sitbot"}}).
type BotletRef struct {
	Name string `json:"Name"`
	Vars map[string]string
}

type botletRefJSON struct {
	Name string            `json:"Name"`
	Vars map[string]string `json:"Vars"`
}

func (r BotletRef) MarshalJSON() ([]byte, error) {
	if len(r.Vars) == 0 {
		return json.Marshal(r.Name)
	}
	return json.Marshal(botletRefJSON{Name: r.Name, Vars: r.Vars})
}

func (r *BotletRef) UnmarshalJSON(data []byte) error {
	// Try string form first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.Name = s
		r.Vars = nil
		return nil
	}
	// Try object form
	var obj botletRefJSON
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	r.Name = obj.Name
	r.Vars = obj.Vars
	return nil
}
