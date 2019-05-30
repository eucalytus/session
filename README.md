# session

![GitHub](https://img.shields.io/github/license/eucalytus/session.svg)
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/eucalytus/session)](https://goreportcard.com/report/github.com/eucalytus/session)
[![Build Status](https://travis-ci.org/eucalytus/session.svg?branch=master)](https://travis-ci.org/eucalytus/session)
![Codecov](https://img.shields.io/codecov/c/github/eucalytus/session.svg)

# How to use it

```go
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
		//listen session event
		func(s *session.Session, event int) {
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
			s = manager.CreateSession(response)
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
```