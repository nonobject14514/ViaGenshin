package kcp

import (
	"net"
	"time"
)

const (
	DefaultMTU = 1500
)

func Dial(addr string) (*Session, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", nil)
	s := newSession(conn, udpAddr, false)
	go s.loopUpdate()
	go loopReadFromUDP(s.conn, s.onControlData, s.onSegmentData)
	return s, s.open()
}

type Listener struct {
	conn  *net.UDPConn
	conns *sessionManager
}

func Listen(addr string) (*Listener, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	l := &Listener{conn: conn}
	l.conns = newSessionManager(5 * time.Second)
	go loopReadFromUDP(l.conn, l.onControlData, l.onSegmentData)
	return l, nil
}

func (l *Listener) Addr() *net.UDPAddr {
	return l.conn.LocalAddr().(*net.UDPAddr)
}

func (l *Listener) Accept() (*Session, error) {
	return l.conns.accept()
}

func (l *Listener) DisconnectSession(session *Session, reason DisconnectReason) error {
	return l.disconnectSession(session.cb.convID, session.sessionID, reason, session.remoteAddr)
}

func (l *Listener) Close() error {
	// close the session manager
	if l.conns != nil {
		l.conns.close()
	}
	// close the underlying UDP connection
	return l.conn.Close()
}
