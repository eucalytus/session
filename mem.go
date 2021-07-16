package session

import (
	"net/http"
	"sync"
	"time"
)

type MemSession struct {
	id           string
	values       map[interface{}]interface{}
	timeAccessed int64
	lock         sync.RWMutex
}

func NewSession(id string, timeAccessed int64) Session {
	return &MemSession{
		id:           id,
		timeAccessed: timeAccessed,
	}
}

func (session *MemSession) GetId() string {
	return session.id
}

//GetMaskedSessionId session id is critical information, we should mask it
func (session *MemSession) GetMaskedSessionId() string {
	buf := []byte(session.id)
	for i := 3; i < len(buf)-3; i++ {
		buf[i] = '*'
	}
	return string(buf)
}

func (session *MemSession) GetLastAccessTime() int64 {
	return session.timeAccessed
}

func (session *MemSession) Set(key interface{}, value interface{}) error {
	session.lock.Lock()
	defer session.lock.Unlock()
	session.values[key] = value
	return nil
}

func (session *MemSession) Get(key interface{}) (value interface{}, ok bool) {
	session.lock.RLock()
	defer session.lock.RUnlock()
	value, ok = session.values[key]
	return
}

//Invalidate invalidate session will remove session from registry
func (session *MemSession) Invalidate() {
	session.timeAccessed = -1
	session.values = make(map[interface{}]interface{})
}

func CreateMemSession(r *http.Request, w http.ResponseWriter) (Session, error) {
	sid := genSessionId(48)
	session := &MemSession{id: sid, timeAccessed: time.Now().Unix(), values: make(map[interface{}]interface{})}
	return session, nil
}
