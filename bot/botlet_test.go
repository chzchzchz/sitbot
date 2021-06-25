package bot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotletRef_MarshalJSON_StringForm(t *testing.T) {
	ref := BotletRef{Name: "greet", Vars: nil}
	b, err := json.Marshal(ref)
	require.NoError(t, err)
	assert.Equal(t, `"greet"`, string(b))
}

func TestBotletRef_MarshalJSON_ObjectForm(t *testing.T) {
	ref := BotletRef{Name: "greet", Vars: map[string]string{"nick": "sitbot"}}
	b, err := json.Marshal(ref)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "greet", got["Name"])
	vars, ok := got["Vars"].(map[string]any)
	require.True(t, ok, "Vars not present as object")
	assert.Equal(t, "sitbot", vars["nick"])
}

func TestBotletRef_UnmarshalJSON_StringForm(t *testing.T) {
	var ref BotletRef
	require.NoError(t, json.Unmarshal([]byte(`"greet"`), &ref))
	assert.Equal(t, "greet", ref.Name)
	assert.Nil(t, ref.Vars)
}

func TestBotletRef_UnmarshalJSON_ObjectForm(t *testing.T) {
	var ref BotletRef
	require.NoError(t, json.Unmarshal([]byte(`{"Name":"greet","Vars":{"nick":"sitbot"}}`), &ref))
	assert.Equal(t, "greet", ref.Name)
	assert.NotNil(t, ref.Vars)
	assert.Equal(t, "sitbot", ref.Vars["nick"])
}

func TestBotletRef_UnmarshalJSON_Invalid(t *testing.T) {
	var ref BotletRef
	assert.Error(t, json.Unmarshal([]byte(`not valid json`), &ref))
}

func TestBotletRef_UnmarshalJSON_EmptyObject(t *testing.T) {
	var ref BotletRef
	require.NoError(t, json.Unmarshal([]byte(`{"Name":""}`), &ref))
	assert.Equal(t, "", ref.Name)
}

func TestBotletMarshalRoundTrip(t *testing.T) {
	original := BotletRef{Name: "greet", Vars: map[string]string{"nick": "sitbot"}}
	b, err := json.Marshal(original)
	require.NoError(t, err)
	var round BotletRef
	require.NoError(t, json.Unmarshal(b, &round))
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Vars["nick"], round.Vars["nick"])
}

func TestBotletMarshalStringRoundTrip(t *testing.T) {
	original := BotletRef{Name: "greet", Vars: nil}
	b, err := json.Marshal(original)
	require.NoError(t, err)
	var round BotletRef
	require.NoError(t, json.Unmarshal(b, &round))
	assert.Equal(t, original.Name, round.Name)
	assert.Nil(t, round.Vars)
}
