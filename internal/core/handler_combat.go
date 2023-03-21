package core

import (
	"encoding/json"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
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
	var invokes []*CombatInvokeEntry
	for _, invoke := range notify.InvokeList {
		if len(invoke.CombatData) == 0 {
			invokes = append(invokes, invoke)
			continue
		}
		name := mapper.CombatArgumentTypes[invoke.ArgumentType]
		if name == "" {
			continue
		}
		invoke.CombatData, err = s.ConvertPacketByName(from, to, name, invoke.CombatData)
		if err != nil {
			return data, err
		}
		invokes = append(invokes, invoke)
	}
	notify.InvokeList = invokes
	return json.Marshal(notify)
}
