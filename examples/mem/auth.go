package main

import (
	"log"
	"net/http"

	"github.com/eucalytus/session"
)

func main() {
	manager := session.NewManager(session.Options{
		MaxInactiveInterval: 1800, MaxAge: 84000, HttpOnly: true,
	},
		session.CreateMemSession,
		//listen session event
		func(s session.Session, event int) {
			if event == session.Created {
				log.Printf("new session is created, sessionId=%s\n", s.GetMaskedSessionId())
			} else if event == session.Destroyed {
				log.Printf("session is destroyed, sessionId=%s\n", s.GetMaskedSessionId())
			} else {
				log.Printf("session is updated, sessionId=%s\n", s.GetMaskedSessionId())
			}
		},
	)

	//private resource
	http.HandleFunc("/private", func(response http.ResponseWriter, request *http.Request) {
		s := manager.GetSession(request)
		if s != nil {
			if _, found := s.Get("key"); found {
				response.Write([]byte("OK"))
				return
			}
		}
		response.WriteHeader(http.StatusUnauthorized)
		response.Write([]byte("StatusUnauthorized"))
	})

	//login
	http.HandleFunc("/login", func(response http.ResponseWriter, request *http.Request) {
		s := manager.GetSession(request)
		if s == nil {
			temp, err := manager.CreateSession(request, response)
			if err != nil {
				log.Printf("create session failed: %v\n", err)
			}
			s = temp
		}
		s.Set("key", "login")
		response.Write([]byte("OK"))
	})

	//logout
	http.HandleFunc("/logout", func(response http.ResponseWriter, request *http.Request) {
		s := manager.GetSession(request)
		if s != nil {
			s.Set("key", nil)
			s.Invalidate()
		}
		response.Write([]byte("OK"))
	})

	http.ListenAndServe("0.0.0.0:8000", nil)
}
