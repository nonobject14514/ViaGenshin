package core

import (
	"context"
	"errors"
	"sync"

	"github.com/Jx2f/ViaGenshin/internal/config"
	"github.com/Jx2f/ViaGenshin/internal/mapper"
)

type Service struct {
	config *config.Config

	keys    *Keys
	mapping *mapper.Mapping

	mu      sync.RWMutex
	servers map[config.Protocol]*Server

	ctx       context.Context
	ctxCancel context.CancelFunc
	stopping  sync.WaitGroup
}

func NewService(c *config.Config) *Service {
	s := new(Service)
	s.config = c
	s.servers = make(map[config.Protocol]*Server)
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	s.stopping = sync.WaitGroup{}
	return s
}

func (s *Service) Start() error {
	var err error
	s.keys, err = NewKeysFromConfig(s.config.Keys)
	if err != nil {
		return err
	}
	s.mapping, err = mapper.NewMappingFromConfig(s.config.Protocols)
	if err != nil {
		return err
	}
	for v := range s.config.Endpoints.Mapping {
		server, err := NewServer(s, s.config.Endpoints, v)
		if err != nil {
			return err
		}
		go func() {
			s.stopping.Add(1)
			if err2 := server.Start(s.ctx); err2 != nil {
				err = errors.Join(err, err2)
			}
			s.stopping.Done()
		}()
		s.servers[v] = server
	}
	select {
	case <-s.ctx.Done():
	}
	return err
}

func (s *Service) Stop() error {
	s.ctxCancel()
	return nil
}
