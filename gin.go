package session

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/context"
)

const DefaultKey = "session-key"

// gin session middleware
func NewGinSession(manager *Manager) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Set(DefaultKey, manager)
		defer context.Clear(c.Request)
		c.Next()
	}
}

//get the session from gin context
func GetSession(c *gin.Context) *Session {
	manager, found := c.Get(DefaultKey)
	if manager != nil && found {
		return manager.(*Manager).GetSession(c.Request)
	}
	return nil
}
