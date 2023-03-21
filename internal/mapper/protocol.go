package mapper

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"

	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

func (m *Mapping) loadBaseProtocol() error {
	logger.Info().Msgf("Loading base protocol %s", m.BaseProtocol)
	return m.loadProtocol(m.BaseProtocol, m.config.Mapping[m.BaseProtocol])
}

func (m *Mapping) loadProtocol(v Protocol, dir string) error {
	logger.Info().Msgf("Loading protocol %s", v)
	data, err := os.ReadFile(path.Join(dir, "protocol.csv"))
	if err != nil {
		return fmt.Errorf("failed to read protocol.csv: %w", err)
	}
	m.CommandNameMap[v] = make(map[uint16]string)
	m.CommandPairMap[v] = make(map[Protocol]map[uint16]uint16)
	if v != m.BaseProtocol {
		m.CommandPairMap[v][m.BaseProtocol] = make(map[uint16]uint16)
		m.CommandPairMap[m.BaseProtocol][v] = make(map[uint16]uint16)
	}
	m.MessageDescMap[v] = make(map[string]*desc.MessageDescriptor)
	parser := &protoparse.Parser{ImportPaths: []string{path.Join(dir, "protocol")}}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(strings.TrimSpace(line), ",")
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		if name == "" || name == "DebugNotify" {
			continue
		}
		command, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			logger.Error().Err(err).Msgf("Failed to parse command %s for %s in %s", parts[1], name, v)
			continue
		}
		if err := m.parseCommandDesc(parser, v, name, uint16(command)); err != nil {
			logger.Error().Err(err).Msgf("Failed to parse command desc for %s in %s", name, v)
			continue
		}
	}
	for _, name := range AbilityInvokeArgumentTypes {
		if err := m.parseMessageDesc(parser, v, name); err != nil {
			continue
		}
	}
	for _, name := range CombatArgumentTypes {
		if err := m.parseMessageDesc(parser, v, name); err != nil {
			continue
		}
	}
	return nil
}

func (m *Mapping) parseCommandDesc(parser *protoparse.Parser, v Protocol, name string, command uint16) error {
	m.CommandNameMap[v][command] = name
	if v == m.BaseProtocol {
		m.BaseCommands[name] = command
	} else if baseCommand, ok := m.BaseCommands[name]; ok && baseCommand != 0 {
		m.CommandPairMap[v][m.BaseProtocol][command] = baseCommand
		m.CommandPairMap[m.BaseProtocol][v][baseCommand] = command
	} else {
		logger.Debug().Msgf("Failed to find base command for %s in %s", name, v)
	}
	return m.parseMessageDesc(parser, v, name)
}

func (m *Mapping) parseMessageDesc(parser *protoparse.Parser, v Protocol, name string) error {
	fd, err := parser.ParseFiles(name + ".proto")
	if err != nil {
		return err
	}
	m.MessageDescMap[v][name] = fd[0].FindMessage(name)
	return nil
}
