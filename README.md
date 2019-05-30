# session

# How to use it

```go
	manager := session.NewManager(session.Options{SessionMaxLifeTime: SessionTimeoutInSec, MaxAge: 84000, HttpOnly: true},
		func(s *session.Session, event int) {
			if event == session.Created {
				log.Info("new session is created", zap.String("sessionId", s.GetId()))
			} else if event == session.Destroyed {
				log.Info("session is destroyed", zap.String("sessionId", s.GetId()))
			}
		},
	)
```