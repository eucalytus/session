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
	Update    = 3
)

type Session interface {
	GetId() string
	GetMaskedSessionId() string
	GetLastAccessTime() int64
	Get(key interface{}) (value interface{}, ok bool)
	Set(key interface{}, value interface{}) error
	Invalidate()
}

type Options struct {
	Path     string
	Domain   string
	MaxAge   int
	Secure   bool
	HttpOnly bool

	MaxInactiveInterval int64
	GcInterval          int
	GcStopChan          <-chan struct{}
	SessionLoader       func() []Session
}

// Manager manage the created sessions
type Manager struct {
	lock                sync.RWMutex             // locker
	sessions            map[string]*list.Element // map in memory
	list                *list.List               // for gc
	maxInactiveInterval int64
	sessionHandler      func(Session, int)
	SessionCreator      func(r *http.Request, w http.ResponseWriter) (Session, error)
	options             Options
}

// NewManager init new session manager and start gc
func NewManager(options Options, SessionCreator func(r *http.Request, w http.ResponseWriter) (Session, error), sessionHandler func(Session, int)) *Manager {
	manager := &Manager{
		list:                list.New(),
		sessions:            make(map[string]*list.Element),
		maxInactiveInterval: options.MaxInactiveInterval,
		sessionHandler:      sessionHandler,
		SessionCreator:      SessionCreator,
	}
	if options.SessionLoader != nil {
		initSessions := options.SessionLoader()
		for _, s := range initSessions {
			//s.manager = manager
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
	manager.options = options
	go manager.periodicCleanup(time.Second*time.Duration(interval), stopChan)
	return manager
}

// GetSession get memory session store http request
func (manager *Manager) GetSession(request *http.Request) Session {
	c, err := request.Cookie("ID")
	if err != nil {
		return nil
	}
	sessionId := c.Value
	return manager.GetSessionById(sessionId)
}

// GetSessionById get memory session store by sid
func (manager *Manager) GetSessionById(sessionId string) Session {
	manager.lock.RLock()
	if element, ok := manager.sessions[sessionId]; ok {
		session := element.Value.(Session)
		go manager.updateSessionAccessTime(session, time.Now().Unix())
		manager.lock.RUnlock()
		return session
	}
	manager.lock.RUnlock()
	return nil
}

func (manager *Manager) CreateSession(r *http.Request, w http.ResponseWriter) (Session, error) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	session, err := manager.SessionCreator(r, w)
	if err != nil {
		return nil, err
	}
	manager.addSession(session)
	manager.addCookie(w, "ID", session.GetId())
	if manager.sessionHandler != nil {
		manager.sessionHandler(session, Created)
	}
	return session, nil
}

// GetSessionSize get all managed session size
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
		if (element.Value.(Session).GetLastAccessTime() + manager.maxInactiveInterval) < time.Now().Unix() {
			manager.lock.RUnlock()
			manager.lock.Lock()
			manager.list.Remove(element)
			delete(manager.sessions, element.Value.(Session).GetId())
			manager.lock.Unlock()
			if manager.sessionHandler != nil {
				manager.sessionHandler(element.Value.(Session), Destroyed)
			}
			manager.lock.RLock()
		} else {
			break
		}
	}
	manager.lock.RUnlock()
}

// SessionUpdate expand lastAccessTime of MemSession store by id in memory MemSession
func (manager *Manager) updateSessionAccessTime(session Session, time int64) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	if element, ok := manager.sessions[session.GetId()]; ok {
		manager.list.MoveToFront(element)
	}
}

//add session without lock
func (manager *Manager) addSession(session Session) {
	element := manager.list.PushFront(session)
	manager.sessions[session.GetId()] = element
}

func (manager *Manager) deleteSession(session Session) {
	manager.updateSessionAccessTime(session, -1)
	manager.sessionGC()
}

func (manager *Manager) addCookie(w http.ResponseWriter, name string, value string) {
	c := http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: manager.options.HttpOnly,
		MaxAge:   manager.options.MaxAge,
		Path:     manager.options.Path,
	}
	http.SetCookie(w, &c)
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
