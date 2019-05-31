package main

import (
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"

	"github.com/eucalytus/session"
)

func main() {
	engine := gin.Default()
	store := cookie.NewStore([]byte("secret"))
	engine.Use(session.UseSession(session.Options{
		MaxInactiveInterval: 30, MaxAge: 84000, HttpOnly: true,
	}, store,
		func(s session.Session, event int) {
			if event == session.Created {
				log.Printf("new session is created, sessionId=%s\n", s.GetMaskedSessionId())
			} else if event == session.Destroyed {
				log.Printf("session is destroyed, sessionId=%s\n", s.GetMaskedSessionId())
			} else {
				log.Printf("session is updated, sessionId=%s\n", s.GetMaskedSessionId())
			}
		},
	))

	//private resource
	engine.GET("/private", func(c *gin.Context) {
		s := session.GetSession(c)
		if s != nil {
			if _, found := s.Get("key"); found {
				c.JSON(http.StatusOK,
					gin.H{"code": "ok", "session": session.GetSessionSize(c)},
				)
				return
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"code": "ok"})
	})

	//login
	engine.GET("/login", func(c *gin.Context) {
		s := session.GetSession(c)
		if s == nil {
			temp, err := session.CreateSession(c)
			if err != nil {
				log.Printf("create session failed: %v\n", err)
			}
			s = temp
		}
		s.Set("key", "login")
		c.JSON(200, gin.H{"code": "ok"})
	})

	//logout
	engine.GET("/logout", func(c *gin.Context) {
		s := session.GetSession(c)
		if s != nil {
			s.Set("key", nil)
			s.Invalidate()
		}
		c.JSON(200, gin.H{"code": "ok"})
	})

	log.Println("start gin engine at 0.0.0.0:8000", engine.Run("0.0.0.0:8000").Error())
}
