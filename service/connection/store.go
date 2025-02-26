package connection

import (
	"runtime"
	"runtime/debug"
	"sync"

	"github.com/dropbox/godropbox/container/set"
	"github.com/pritunl/pritunl-client-electron/service/utils"
	"github.com/sirupsen/logrus"
)

var GlobalStore = &Store{
	conns: map[string]*Connection{},
}

type Store struct {
	dnsForced bool
	lock      sync.RWMutex
	conns     map[string]*Connection
}

func (s *Store) cleanState() {
	defer func() {
		panc := recover()
		if panc != nil {
			logrus.WithFields(logrus.Fields{
				"stack": string(debug.Stack()),
				"panic": panc,
			}).Error("profile: Clean state panic")
		}
	}()

	if runtime.GOOS == "darwin" && len(s.conns) == 0 {
		err := utils.ClearScutilConnKeys()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Error("connection: Failed to clear scutil connection keys")
		}

		if s.dnsForced {
			utils.ClearDns()
			s.dnsForced = false
		}
	}
}

func (s *Store) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.conns)
}

func (s *Store) IsActive() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.conns) > 0
}

func (s *Store) IsConnected() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, conn := range s.conns {
		if conn.Data.Status == Connected {
			return true
		}
	}

	return false
}

func (s *Store) Add(prflId string, conn *Connection) {
	s.lock.RLock()
	c := s.conns[prflId]
	if c == nil {
		s.conns[prflId] = conn
		s.lock.RUnlock()
		return
	}
	s.lock.RUnlock()

	logrus.WithFields(c.Fields(nil)).Error(
		"connection: Overwriting stored connection")
	c.StopWait()

	s.lock.RLock()
	c = s.conns[prflId]
	if c != nil {
		c.State.SetStop()
	}
	s.conns[prflId] = conn
	s.lock.RUnlock()

	logrus.WithFields(conn.Fields(nil)).Error(
		"connection: Overwrote stored connection")

	return
}

func (s *Store) Remove(prflId string, conn *Connection) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	c := s.conns[prflId]
	if c == conn {
		delete(s.conns, prflId)
	} else {
		logrus.WithFields(c.Fields(nil)).Error(
			"connection: Attempting to delete active connection")
		logrus.WithFields(conn.Fields(nil)).Error(
			"connection: Attempted to delete active connection")
	}

	go func() {
		s.lock.RLock()
		defer s.lock.RUnlock()

		s.cleanState()
	}()

	return
}

func (s *Store) Get(prflId string) (conn *Connection) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	s.cleanState()
	conn = s.conns[prflId]

	return
}

func (s *Store) GetData(prflId string) (prfl *Data) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	s.cleanState()
	conn := s.conns[prflId]
	if conn != nil {
		prfl = conn.Data
	}

	return
}

func (s *Store) GetAll() (conns map[string]*Connection) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	conns = map[string]*Connection{}

	s.cleanState()
	for _, conn := range s.conns {
		conns[conn.Id] = conn
	}

	return
}

func (s *Store) GetAllData() (prfls map[string]*Data) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	prfls = map[string]*Data{}

	s.cleanState()
	for _, conn := range s.conns {
		prfls[conn.Id] = conn.Data
	}

	return
}

func (s *Store) GetAllId() (connIds set.Set) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	connIds = set.NewSet()

	s.cleanState()
	for _, conn := range s.conns {
		connIds.Add(conn.Id)
	}

	return
}

func (s *Store) SetDnsForced() {
	s.lock.Lock()
	s.dnsForced = true
	s.lock.Unlock()
}
