package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"sync"

	"github.com/Jx2f/ViaGenshin/internal/config"
	"github.com/Jx2f/ViaGenshin/internal/mapper"
	"github.com/Jx2f/ViaGenshin/pkg/crypto/mt19937"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
	"github.com/Jx2f/ViaGenshin/pkg/transport"
	"github.com/Jx2f/ViaGenshin/pkg/transport/kcp"
)

type Server struct {
	*Service
	config *config.ConfigEndpoints

	mu       sync.RWMutex
	protocol mapper.Protocol
	listener *kcp.Listener
	sessions map[uint32]*Session
}

func NewServer(s *Service, c *config.ConfigEndpoints, v config.Protocol) (*Server, error) {
	e := new(Server)
	e.Service = s
	e.config = c
	var err error
	e.protocol = v
	e.listener, err = kcp.Listen(e.config.Mapping[v])
	if err != nil {
		return nil, err
	}
	e.sessions = make(map[uint32]*Session)
	return e, nil
}

func (e *Server) Start(ctx context.Context) error {
	logger.Info().Msgf("Start listening on %s", e.listener.Addr())
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		conn, err := e.listener.Accept()
		if err != nil {
			return err
		}
		go e.handleConn(conn)
	}
}

func (e *Server) handleConn(conn *kcp.Session) {
	logger.Info().Msgf("New session from %s", conn.RemoteAddr())
	if err := e.NewSession(conn).Start(); err != nil {
		logger.Error().Err(err).Msgf("Session %d closed", conn.SessionID())
	}
}

func (s *Server) NewSession(conn *kcp.Session) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := newSession(s, conn)
	s.sessions[conn.SessionID()] = session
	return session
}

type Session struct {
	*Server
	endpoint *kcp.Session
	upstream *kcp.Session

	loginRand uint64
	loginKey  *mt19937.KeyBlock
}

func newSession(s *Server, endpoint *kcp.Session) *Session {
	return &Session{Server: s, endpoint: endpoint}
}

func (s *Session) Start() error {
	var err error
	s.upstream, err = kcp.Dial(s.config.MainEndpoint)
	if err != nil {
		return err
	}
	logger.Info().Msgf("Start forwarding session %d to %s, mapping %s <-> %s", s.endpoint.SessionID(), s.upstream.RemoteAddr(), s.protocol, s.config.MainProtocol)
	return s.Forward()
}

func (s *Session) Forward() error {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			payload, err := s.endpoint.Payload()
			if err != nil {
				logger.Error().Err(err).Msgf("Failed to get endpoint payload")
				return
			}
			if err := s.ConvertPayload(
				s.endpoint, s.upstream, s.protocol, s.config.MainProtocol, payload,
			); err != nil {
				logger.Error().Err(err).Msg("Failed to convert endpoint payload")
			}
			payload.Release()
		}
	}()
	go func() {
		defer wg.Done()
		for {
			payload, err := s.upstream.Payload()
			if err != nil {
				logger.Error().Err(err).Msgf("Failed to get upstream payload")
				return
			}
			if err := s.ConvertPayload(
				s.upstream, s.endpoint, s.config.MainProtocol, s.protocol, payload,
			); err != nil {
				logger.Error().Err(err).Msg("Failed to convert upstream payload")
			}
			payload.Release()
		}
	}()
	wg.Wait()
	return nil
}

func (s *Session) ConvertPayload(
	fromSession, toSession *kcp.Session,
	from, to mapper.Protocol, payload transport.Payload,
) error {
	n := len(payload)
	if n < 12 {
		return errors.New("packet too short")
	}
	if err := s.EncryptPayload(payload, false); err != nil {
		return err
	}
	if payload[0] != 0x45 || payload[1] != 0x67 || payload[n-2] != 0x89 || payload[n-1] != 0xAB {
		return errors.New("invalid payload")
	}
	b := bytes.NewBuffer(payload[2 : n-2])
	fromCmd := binary.BigEndian.Uint16(b.Next(2))
	toCmd := s.mapping.CommandPairMap[from][to][fromCmd]
	n1 := binary.BigEndian.Uint16(b.Next(2))
	n2 := binary.BigEndian.Uint32(b.Next(4))
	if uint32(n) != 12+uint32(n1)+n2 {
		return errors.New("invalid packet length")
	}
	head := b.Next(int(n1))
	fromData := b.Next(int(n2))
	toData, err := s.ConvertPacket(from, to, fromCmd, fromData)
	if err != nil {
		return err
	}
	b = bytes.NewBuffer(nil)
	b.Write([]byte{0x45, 0x67})
	binary.Write(b, binary.BigEndian, uint16(toCmd))
	binary.Write(b, binary.BigEndian, uint16(len(head)))
	binary.Write(b, binary.BigEndian, uint32(len(toData)))
	b.Write(head)
	b.Write(toData)
	b.Write([]byte{0x89, 0xAB})
	payload = b.Bytes()
	name := s.mapping.CommandNameMap[to][toCmd]
	if err := s.EncryptPayload(payload, name == "GetPlayerTokenReq" || name == "GetPlayerTokenRsp"); err != nil {
		return err
	}
	return toSession.SendPayload(payload)
}

func (s *Session) EncryptPayload(payload transport.Payload, first bool) error {
	n := len(payload)
	if n < 4 {
		return errors.New("packet too short")
	}
	var encrypt = payload[0] == 0x45 && payload[1] == 0x67 && payload[n-2] == 0x89 && payload[n-1] == 0xAB
	if s.loginKey != nil && !first {
		s.loginKey.Xor(payload)
		if !encrypt && (payload[0] != 0x45 || payload[1] != 0x67 || payload[n-2] != 0x89 || payload[n-1] != 0xAB) {
			// revert
			s.loginKey.Xor(payload)
		} else {
			return nil
		}
	}
	s.keys.SharedKey.Xor(payload)
	return nil
}
