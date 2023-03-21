package kcp

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"
)

var (
	ErrSessionNotFound = errors.New("session not found")
)

type sessionManager struct {
	refCount  sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc

	timeout time.Duration
	pending chan *Session

	convID uint32
	random *rand.Rand

	sync.RWMutex
	conns map[uint32]*Session
}

func newSessionManager(timeout time.Duration) *sessionManager {
	m := &sessionManager{
		timeout: timeout,
		pending: make(chan *Session, 128),
		random:  rand.New(rand.NewSource(time.Now().UnixNano())),
		conns:   make(map[uint32]*Session),
	}
	m.ctx, m.ctxCancel = context.WithCancel(context.Background())
	go m.loopUpdate()
	return m
}

func (m *sessionManager) loopUpdate() {
	ticker := time.NewTicker(time.Millisecond * 20)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.RLock()
			for _, session := range m.conns {
				session.update()
			}
			m.RUnlock()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *sessionManager) accept() (*Session, error) {
	select {
	case session := <-m.pending:
		return session, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *sessionManager) close() {
	m.Lock()
	defer m.Unlock()
	m.ctxCancel()
	m.refCount.Wait()
}

func (m *sessionManager) handleSession(session *Session) {
	select {
	case m.pending <- session:
	default:
		logging(LoggingLevelError, "pending accept session queue is full")
	}
	select {
	case <-m.ctx.Done():
		session.closeSession(DisconnectReasonServerShutdown)
	case <-session.ctx.Done():
	}
}

func (m *sessionManager) nextConvID() uint32 {
	m.convID++
	if m.convID == 0 {
		m.convID = 1
	}
	return m.convID
}

func (m *sessionManager) nextSessionID() uint32 {
	for {
		sessionID := m.random.Uint32()
		if _, ok := m.conns[sessionID]; !ok {
			return sessionID
		}
	}
}

func (m *sessionManager) maybeGetSession(convID, sessionID uint32, addr *net.UDPAddr) (*Session, error) {
	session, ok := m.conns[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (m *sessionManager) getSession(convID, sessionID uint32, addr *net.UDPAddr) (*Session, error) {
	m.RLock()
	defer m.RUnlock()
	return m.maybeGetSession(convID, sessionID, addr)
}

func (m *sessionManager) getOrCreateSession(convID, sessionID uint32, addr *net.UDPAddr, conn *net.UDPConn) (*Session, error) {
	m.Lock()
	defer m.Unlock()
	session, _ := m.maybeGetSession(convID, sessionID, addr)
	if session != nil {
		return session, nil
	}
	if sessionID == 0 {
		sessionID = m.nextSessionID()
	}
	// create a new session
	session = newSession(conn, addr, true)
	session.start(m.ctx, m.nextConvID(), sessionID)
	m.conns[session.sessionID] = session
	// start a goroutine to handle the session
	m.refCount.Add(1)
	go func() {
		defer m.refCount.Done()
		// blocking here until the session is closed
		m.handleSession(session)
	}()
	return session, nil
}

func (m *sessionManager) deleteSession(convID, sessionID uint32, addr *net.UDPAddr) (*Session, error) {
	m.Lock()
	defer m.Unlock()
	session, err := m.maybeGetSession(convID, sessionID, addr)
	if err != nil {
		return nil, err
	}
	delete(m.conns, sessionID)
	return session, nil
}

func (l *Listener) disconnect(convID, sessionID uint32, reason DisconnectReason, addr *net.UDPAddr) error {
	data := controlDataPool.Get().(*controlData)
	defer controlDataPool.Put(data)
	data.Set(controlCommandFin, convID, uint32(sessionID), uint32(reason))
	return writeControlDataToUDP(l.conn, data, addr)
}

func (l *Listener) connectSession(data *controlData, addr *net.UDPAddr) error {
	convID, sessionID := data.ConvID(), data.SessionID()
	session, err := l.conns.getOrCreateSession(convID, sessionID, addr, l.conn)
	if err != nil {
		return l.disconnect(convID, sessionID, DisconnectReasonServerKick, addr)
	}
	return session.connectAck()
}

func (l *Listener) disconnectSession(convID, sessionID uint32, reason DisconnectReason, addr *net.UDPAddr) error {
	session, err := l.conns.deleteSession(convID, sessionID, addr)
	if err != nil {
		return l.disconnect(convID, sessionID, reason, addr)
	}
	session.cb.convID = convID
	return session.closeSession(reason)
}

func (l *Listener) onControlData(data *controlData, addr *net.UDPAddr) error {
	// handle control data
	var err error
	switch data.Command() {
	case controlCommandSyn:
		err = l.connectSession(data, addr)
	case controlCommandFin:
		err = l.disconnectSession(data.ConvID(), data.SessionID(), DisconnectReasonServerKick, addr)
	default:
		err = ErrInvalidPacket
	}
	return err
}

func (l *Listener) onSegmentData(data []byte, addr *net.UDPAddr) error {
	convID := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	sessionID := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	session, err := l.conns.getSession(convID, sessionID, addr)
	if err != nil {
		return l.disconnect(convID, sessionID, DisconnectReasonServerKick, addr)
	}
	return session.onSegmentData(data, addr)
}
