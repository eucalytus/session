package session

import (
	"errors"
	gonicSession "github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"net/http"
	"sync"
	"time"
)

const DefaultKey = "session-key"
const storeKey = "store"
const sessionName = "GIN"
const sessionId = "__SESSION_ID__"
const sessionTime = "__SESSION_TIME__"

type ginSession struct {
	lock         sync.RWMutex
	id           string
	timeAccessed int64
	request      *http.Request
	write        http.ResponseWriter
	store        gonicSession.Store
	innerSession *sessions.Session
}

func (session *ginSession) GetId() string {
	return session.id
}

//session id is critical information, we should mask it
func (session *ginSession) GetMaskedSessionId() string {
	buf := []byte(session.id)
	for i := 3; i < len(buf)-3; i++ {
		buf[i] = '*'
	}
	return string(buf)
}

func (session *ginSession) GetLastAccessTime() int64 {
	return session.timeAccessed
}

func (session *ginSession) Set(key interface{}, value interface{}) error {
	session.lock.Lock()
	defer session.lock.Unlock()
	session.innerSession.Values[key] = value
	return session.innerSession.Save(session.request, session.write)
}

func (session *ginSession) Get(key interface{}) (value interface{}, ok bool) {
	session.lock.RLock()
	defer session.lock.RUnlock()
	value, ok = session.innerSession.Values[key]
	return
}

//invalidate session will remove session from registry
func (session *ginSession) Invalidate() {
	session.timeAccessed = -1
	session.innerSession.Values = make(map[interface{}]interface{})
	session.innerSession.Save(session.request, session.write)
}

// gin session middleware
func UseSession(options Options, store gonicSession.Store, sessionHandler func(Session, int)) func(*gin.Context) {
	creator := func(r *http.Request, w http.ResponseWriter) (Session, error) {
		sid := genSessionId(48)
		session := &ginSession{id: sid, timeAccessed: time.Now().Unix(), request: r, write: w}
		in, err := store.New(r, sessionName)
		if err != nil {
			return nil, err
		}
		in.Values[sessionId] = sid
		session.innerSession = in
		return session, nil
	}
	manager := NewManager(options, creator, sessionHandler)

	return func(c *gin.Context) {
		c.Set(DefaultKey, manager)
		c.Set(storeKey, store)
		defer context.Clear(c.Request)
		c.Next()
	}
}

//get the session from gin context
func GetSession(c *gin.Context) Session {
	manager, found := c.Get(DefaultKey)
	if manager != nil && found {
		iface := manager.(*Manager).GetSession(c.Request)
		if iface != nil {
			session := iface.(*ginSession)
			session.innerSession.Values[sessionName] = time.Now().Unix()
			session.request = c.Request
			session.write = c.Writer
			return session
		} else {
			return getSessionFromStore(c, manager.(*Manager))
		}
	}
	return nil
}

func GetSessionSize(c *gin.Context) int {
	manager, found := c.Get(DefaultKey)
	if manager != nil && found {
		return manager.(*Manager).GetSessionSize()
	}
	return -1
}

func CreateSession(c *gin.Context) (Session, error) {
	manager, found := c.Get(DefaultKey)
	if manager != nil && found {
		return manager.(*Manager).CreateSession(c.Request, c.Writer)
	}
	return nil, errors.New("no session manager is found")
}

func getSessionFromStore(c *gin.Context, manager *Manager) Session {
	store, found := c.Get(storeKey)
	if store != nil && found {
		session, err := store.(gonicSession.Store).Get(c.Request, sessionName)
		if err == nil && session != nil {
			sid, ok := session.Values[sessionId]
			accessTime, timeFound := session.Values[sessionTime]
			if ok && timeFound {
				if time.Now().Unix()-accessTime.(int64) > manager.maxInactiveInterval {
					old := &ginSession{id: sid.(string), timeAccessed: time.Now().Unix(), innerSession: session, request: c.Request, write: c.Writer}
					manager.addSession(old)
					return old
				}
			}
		}
	}
	return nil
}
