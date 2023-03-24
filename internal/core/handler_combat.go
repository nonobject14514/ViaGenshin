package core

import (
	"encoding/json"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

type CombatInvokeEntry struct {
	CombatData   []byte `json:"combatData"`
	ArgumentType uint32 `json:"argumentType"`
	ForwardType  uint32 `json:"forwardType"`
}

type CombatInvocationsNotify struct {
	InvokeList []*CombatInvokeEntry `json:"invokeList"`
}

func (s *Session) OnCombatInvocationsNotify(from, to mapper.Protocol, data []byte) ([]byte, error) {
	notify := new(CombatInvocationsNotify)
	err := json.Unmarshal(data, notify)
	if err != nil {
		return data, err
	}
	notify.InvokeList = s.OnCombatInvocations(from, to, notify.InvokeList)
	return json.Marshal(notify)
}

func (s *Session) OnCombatInvocations(from, to mapper.Protocol, in []*CombatInvokeEntry) []*CombatInvokeEntry {
	var out []*CombatInvokeEntry
	var err error
	for _, invoke := range in {
		if len(invoke.CombatData) == 0 {
			out = append(out, invoke)
			continue
		}
		name := mapper.CombatTypeArguments[invoke.ArgumentType]
		if name == "" {
			logger.Debug().Msgf("Unknown combat invoke packet %d", invoke.ArgumentType)
			continue
		}
		invoke.CombatData, err = s.ConvertPacketByName(from, to, name, invoke.CombatData)
		if err != nil {
			logger.Debug().Err(err).Msgf("Failed to convert combat invoke packet %s", name)
			continue
		}
		out = append(out, invoke)
	}
	return out
}
