package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManagerWithEmptyOption(t *testing.T) {
	manager := NewManager(Options{}, nil)
	assert.NotNil(t, manager)
}

func TestNewManagerWithLoader(t *testing.T) {
	loader := func() []*Session {
		var sessions []*Session
		sessions = append(sessions, NewSession("a", 1))
		sessions = append(sessions, NewSession("b", 2))
		return sessions
	}
	manager := NewManager(Options{SessionLoader: loader}, nil)
	assert.NotNil(t, manager)
	assert.Equal(t, 2, manager.GetSessionSize())
	assert.NotNil(t, manager.GetSessionById("a"))
	assert.NotNil(t, manager.GetSessionById("b"))
	assert.Nil(t, manager.GetSessionById("c"))
}

func TestSession_GetMaskedSessionId(t *testing.T) {
	s := &Session{id: "abc12345abc"}
	assert.Equal(t, "abc*****abc", s.GetMaskedSessionId())
	s.id = "ac"
	assert.Equal(t, "ac", s.GetMaskedSessionId())
}
