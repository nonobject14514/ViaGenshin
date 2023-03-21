package kcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Jx2f/ViaGenshin/pkg/transport"
)

type DisconnectReason uint8

const (
	DisconnectReasonTimeout DisconnectReason = iota
	DisconnectReasonClientClose
	DisconnectReasonClientRebindFail
	DisconnectReasonClientShutdown
	DisconnectReasonServerRelogin
	DisconnectReasonServerKick
	DisconnectReasonServerShutdown
	DisconnectReasonNotFoundSession
	DisconnectReasonLoginUnfinished
	DisconnectReasonPacketFreqTooHigh
	DisconnectReasonPingTimeout
	DisconnectReasonTransferFailed
	DisconnectReasonServerKillClient
	DisconnectReasonCheckMoveSpeed
	DisconnectReasonAccountPasswordChange
	DisconnectReasonSecurityKick
	DisconnectReasonLuaShellTimeout
	DisconnectReasonSDKFailKick
	DisconnectReasonPacketCostTime
	DisconnectReasonPacketUnionFreq
	DisconnectReasonWaitSndMax
)

var (
	ErrInvalidPacket = errors.New("invalid packet")
)

type Session struct {
	sync.Mutex
	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
	sessionID  uint32
	isManaged  bool // if true, the session is managed by sessionManager

	ctx       context.Context
	ctxCancel context.CancelFunc

	cb *ControlBlock

	payload chan transport.Payload

	startErr error
	starting chan struct{}
}

func newSession(conn *net.UDPConn, addr *net.UDPAddr, isManaged bool) *Session {
	s := &Session{
		conn:       conn,
		remoteAddr: addr,
		isManaged:  isManaged,
		payload:    make(chan transport.Payload, 256),
		starting:   make(chan struct{}),
	}
	return s
}

func (s *Session) RemoteAddr() *net.UDPAddr { return s.remoteAddr }
func (s *Session) SessionID() uint32        { return s.sessionID }

func (s *Session) Payload() (transport.Payload, error) {
	select {
	case payload := <-s.payload:
		return payload, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
}

func (s *Session) SendPayload(payload transport.Payload) error {
	s.Lock()
	defer s.Unlock()
	code := s.cb.Send(payload)
	payload.Release()
	if code < 0 {
		return fmt.Errorf("kcp: failed to send payload: %d", code)
	}
	return nil
}

func (s *Session) loopUpdate() {
	ticker := time.NewTicker(time.Millisecond * 20)
	defer ticker.Stop()
	for {
		<-ticker.C
		s.update()
	}
}

func (s *Session) update() {
	s.Lock()
	defer s.Unlock()
	s.cb.Update()
	n := s.cb.PeekSize()
	if n < 1 {
		return
	}
	payload := transport.NewPayload(n)
	if code := s.cb.Recv(payload); code < 0 {
		payload.Release()
		logging(LoggingLevelError, "failed to receive payload: %d", code)
		return
	}
	select {
	case s.payload <- payload:
	default:
		println("session payload channel is full")
	}
}

func (s *Session) Close() error {
	return s.closeSession(DisconnectReasonClientClose)
}

func (s *Session) closeSession(reason DisconnectReason) error {
	// noinspection GoUnhandledErrorResult
	s.disconnect(reason)
	if !s.isManaged {
		// close the underlying UDP connection if the session is not managed by sessionManager
		return s.conn.Close()
	}
	s.ctxCancel()
	return nil
}

func (s *Session) open() error {
	var err error
	err = s.connectSyn()
	if err != nil {
		return err
	}
	select {
	case <-s.starting:
	}
	return s.startErr
}

func (s *Session) connectSyn() error {
	data := controlDataPool.Get().(*controlData)
	defer controlDataPool.Put(data)
	convID := uint32(0)
	if s.cb != nil {
		convID = s.cb.convID
	}
	data.Set(controlCommandSyn, convID, s.sessionID, controlMessageClientAppID)
	return writeControlDataToUDP(s.conn, data, s.remoteAddr)
}

func (s *Session) connectAck() error {
	data := controlDataPool.Get().(*controlData)
	defer controlDataPool.Put(data)
	data.Set(controlCommandAck, s.cb.convID, s.sessionID, controlMessageClientAppID)
	return writeControlDataToUDP(s.conn, data, s.remoteAddr)
}

func (s *Session) disconnect(reason DisconnectReason) error {
	data := controlDataPool.Get().(*controlData)
	defer controlDataPool.Put(data)
	data.Set(controlCommandFin, s.cb.convID, s.sessionID, uint32(reason))
	return writeControlDataToUDP(s.conn, data, s.remoteAddr)
}

func (s *Session) start(ctx context.Context, convID, sessionID uint32) {
	s.sessionID = sessionID
	s.ctx, s.ctxCancel = context.WithCancel(ctx)
	s.cb = NewControlBlock(convID, sessionID, s.output)
	s.cb.SetMtu(1200)
	s.cb.NoDelay(1, 20, 2, 1)
	s.cb.WndSize(255, 255)
	close(s.starting)
}

func (s *Session) onControlData(data *controlData, addr *net.UDPAddr) error {
	// handle control data
	var err error
	switch data.Command() {
	case controlCommandAck:
		// avoid multiple ACK
		select {
		case <-s.starting:
			logging(LoggingLevelDebug, "session already started")
		default:
			s.start(context.Background(), data.ConvID(), data.SessionID())
		}
	case controlCommandFin:
		err = s.closeSession(DisconnectReasonServerKick)
	default:
		err = ErrInvalidPacket
	}
	return err
}

func (s *Session) output(data []byte) {
	if err := writeToUDP(s.conn, data, s.remoteAddr); err != nil {
		logging(LoggingLevelError, "failed to write to %s: %v", s.remoteAddr, err)
	}
}

func (s *Session) onSegmentData(data []byte, addr *net.UDPAddr) error {
	s.Lock()
	defer s.Unlock()
	// handle segmentData data
	code := s.cb.Input(data, true, false)
	if code < 0 {
		return fmt.Errorf("kcp: failed to receive segment data: %d", code)
	}
	return nil
}
