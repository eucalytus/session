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

func NewSession(id string, timeAccessed int64) *Session {
	return &Session{
		id:           id,
		timeAccessed: timeAccessed,
	}
}

func (session *Session) GetId() string {
	return session.id
}

func (session *Session) getTimeAccessed() int64 {
	return session.timeAccessed
}

func (session *Session) Set(key interface{}, value interface{}) {
	session.lock.Lock()
	defer session.lock.Unlock()
	session.values[key] = value
}

func (session *Session) Get(key interface{}) (value interface{}, ok bool) {
	session.lock.RLock()
	defer session.lock.RUnlock()
	value, ok = session.values[key]
	return
}

type Options struct {
	Path                    string
	Domain                  string
	MaxAge                  int
	Secure                  bool
	HttpOnly                bool
	SessionMaxLifeTime      int64
	GcInterval              int
	GcStopChan              <-chan struct{}
	SessionPersistentHandle func(*Session)
	SessionLoader           func() []*Session
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
	if options.SessionLoader != nil {
		initSessions := options.SessionLoader()
		for _, s := range initSessions {
			manager.addSession(s)
		}
	}
	interval := 60
	if options.GcInterval > 0 {
		interval = options.GcInterval
	}
	stopChan := make(<-chan struct{})
	if options.GcStopChan != nil {
		stopChan = options.GcStopChan
	}
	go manager.periodicCleanup(time.Duration(time.Second*time.Duration(interval)), stopChan)
	return manager
}

// SessionRead get memory session store http request
func (manager *Manager) GetSession(request *http.Request) *Session {
	cookie, err := request.Cookie("ID")
	if err != nil {
		return nil
	}
	sessionId := cookie.Value
	return manager.GetSessionById(sessionId)
}

// SessionRead get memory session store by sid
func (manager *Manager) GetSessionById(sessionId string) *Session {
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
	manager.addSession(session)
	manager.addCookie(w, "ID", sid)
	if manager.sessionHandler != nil {
		manager.sessionHandler(session, Created)
	}
	return session
}

// get all managed session size
func (manager *Manager) GetSessionSize() int {
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	return len(manager.sessions)
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

//add session without lock
func (manager *Manager) addSession(session *Session) {
	element := manager.list.PushFront(session)
	manager.sessions[session.id] = element
}

func (manager *Manager) addCookie(w http.ResponseWriter, name string, value string) {
	cookie := http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: manager.options.HttpOnly,
		MaxAge:   manager.options.MaxAge,
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
