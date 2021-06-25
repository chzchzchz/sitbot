package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chzchzchz/sitbot/bot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotletHandler_PostReturnsOK(t *testing.T) {
	g := bot.NewGang()
	h := NewGangHandler(g)
	req := httptest.NewRequest(http.MethodPost, "/botlet/greet", strings.NewReader(`{"Name":"greet","Vars":{"nick":"sitbot"}}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{ "error" : 0 }`, rec.Body.String())
	bl := g.LookupBotlet("greet")
	require.NotNil(t, bl)
	assert.Equal(t, "sitbot", bl.Vars["nick"])
}

func TestBotletHandler_PutAndDelete(t *testing.T) {
	g := bot.NewGang()
	h := NewGangHandler(g)
	req := httptest.NewRequest(http.MethodPut, "/botlet/greet", strings.NewReader(`{"Patterns":[{"Match":"^x$","Template":"y"}]}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{ "error" : 0 }`, rec.Body.String())
	assert.NotNil(t, g.LookupBotlet("greet"))

	req = httptest.NewRequest(http.MethodDelete, "/botlet/greet", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, g.LookupBotlet("greet"))
}

func TestBotletHandler_RejectsEmptyName(t *testing.T) {
	h := NewGangHandler(bot.NewGang())
	req := httptest.NewRequest(http.MethodPost, "/botlet/", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBotletHandler_RejectsInvalidPatterns(t *testing.T) {
	g := bot.NewGang()
	h := NewGangHandler(g)
	req := httptest.NewRequest(http.MethodPost, "/botlet/bad", strings.NewReader(`{"Name":"bad","Patterns":[{"Match":"[","Template":"x"}]}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, g.LookupBotlet("bad"))
}

func TestBotletHandler_RejectsNameMismatch(t *testing.T) {
	g := bot.NewGang()
	h := NewGangHandler(g)
	req := httptest.NewRequest(http.MethodPost, "/botlet/foo", strings.NewReader(`{"Name":"bar"}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Nil(t, g.LookupBotlet("foo"))
	assert.Nil(t, g.LookupBotlet("bar"))
}

func TestBotletHandler_DeleteMissingReturnsNotFound(t *testing.T) {
	h := NewGangHandler(bot.NewGang())
	req := httptest.NewRequest(http.MethodDelete, "/botlet/missing", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestBotHandler_DeleteMissingReturnsNotFound(t *testing.T) {
	h := NewGangHandler(bot.NewGang())
	req := httptest.NewRequest(http.MethodDelete, "/bot/missing", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
