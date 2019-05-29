package session

import (
	"container/list"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().Unix())
}

const (
	Created   = 1
	Destroyed = 2
)

type Session struct {
	id           string
	values       map[interface{}]interface{}
	timeAccessed int64
	lock         sync.RWMutex
}

func (session *Session) getId() string {
	return session.id
}

type Options struct {
	Path               string
	Domain             string
	MaxAge             int
	Secure             bool
	HttpOnly           bool
	SessionMaxLifeTime int64
	GcInterval         int
	GcStopChan         <-chan struct{}
}

// Manager manage the created sessions
type Manager struct {
	lock           sync.RWMutex             // locker
	sessions       map[string]*list.Element // map in memory
	list           *list.List               // for gc
	maxLifeTime    int64
	sessionHandler func(*Session, int)
	options        Options
}

//init new session manager and start gc
func NewManager(options Options, sessionHandler func(*Session, int)) *Manager {
	manager := &Manager{
		list:           list.New(),
		sessions:       make(map[string]*list.Element),
		maxLifeTime:    options.SessionMaxLifeTime,
		sessionHandler: sessionHandler,
	}
	manager.periodicCleanup(time.Duration(time.Second*time.Duration(options.GcInterval)), options.GcStopChan)
	return manager
}

// SessionRead get memory session store by sid
func (manager *Manager) GetSession(request *http.Request) *Session {
	cookie, err := request.Cookie("ID")
	if err != nil {
		return nil
	}
	sessionId := cookie.Value
	manager.lock.RLock()
	if element, ok := manager.sessions[sessionId]; ok {
		go manager.updateSessionAccessTime(sessionId)
		manager.lock.RUnlock()
		return element.Value.(*Session)
	}
	manager.lock.RUnlock()
	return nil
}

func (manager *Manager) CreatSession(w http.ResponseWriter) *Session {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	sid := genSessionId(48)
	session := &Session{id: sid, timeAccessed: time.Now().Unix(), values: make(map[interface{}]interface{})}
	element := manager.list.PushFront(session)
	manager.sessions[sid] = element
	addCookie(w, "ID", sid)
	if manager.sessionHandler != nil {
		manager.sessionHandler(session, Created)
	}
	return session
}

// periodicCleanup runs Cleanup every interval. Close quit channel to stop.
func (manager *Manager) periodicCleanup(interval time.Duration, quit <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			manager.sessionGC()
		case <-quit:
			return
		}
	}
}

// SessionGC clean expired sessions
func (manager *Manager) sessionGC() {
	manager.lock.RLock()
	for {
		element := manager.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*Session).timeAccessed + manager.maxLifeTime) < time.Now().Unix() {
			manager.lock.RUnlock()
			manager.lock.Lock()
			manager.list.Remove(element)
			delete(manager.sessions, element.Value.(*Session).id)
			manager.lock.Unlock()
			if manager.sessionHandler != nil {
				manager.sessionHandler(element.Value.(*Session), Destroyed)
			}
			manager.lock.RLock()
		} else {
			break
		}
	}
	manager.lock.RUnlock()
}

// SessionUpdate expand time of Session store by id in memory Session
func (manager *Manager) updateSessionAccessTime(sid string) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	if element, ok := manager.sessions[sid]; ok {
		element.Value.(*Session).timeAccessed = time.Now().Unix()
		manager.list.MoveToFront(element)
	}
}

func addCookie(w http.ResponseWriter, name string, value string) {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
}

const letterBytes = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func genSessionId(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(letterBytes[rand.Int63()%int64(len(letterBytes))])
	}
	return sb.String()
}
