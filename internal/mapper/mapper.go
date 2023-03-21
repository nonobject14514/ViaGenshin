package mapper

import (
	"github.com/jhump/protoreflect/desc"

	"github.com/Jx2f/ViaGenshin/internal/config"
)

type Protocol = config.Protocol

type Mapping struct {
	config *config.ConfigProtocols

	BaseProtocol Protocol
	BaseCommands map[string]uint16

	CommandNameMap map[Protocol]map[uint16]string
	CommandPairMap map[Protocol]map[Protocol]map[uint16]uint16
	MessageDescMap map[Protocol]map[string]*desc.MessageDescriptor
}

func NewMappingFromConfig(c *config.ConfigProtocols) (*Mapping, error) {
	m := new(Mapping)
	m.config = c
	m.BaseProtocol = m.config.BaseProtocol
	m.BaseCommands = make(map[string]uint16)
	m.CommandNameMap = make(map[Protocol]map[uint16]string)
	m.CommandPairMap = make(map[Protocol]map[Protocol]map[uint16]uint16)
	m.MessageDescMap = make(map[Protocol]map[string]*desc.MessageDescriptor)
	if err := m.loadBaseProtocol(); err != nil {
		return nil, err
	}
	for v, dir := range m.config.Mapping {
		if v == m.BaseProtocol {
			continue
		}
		if err := m.loadProtocol(v, dir); err != nil {
			return nil, err
		}
	}
	return m, nil
}
