package core

import (
	"encoding/json"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

type AbilityInvokeEntryHead struct {
	LocalID               int32  `json:"localId"`
	ServerBuffUID         uint32 `json:"serverBuffUid"`
	TargetID              uint32 `json:"targetId"`
	InstancedAbilityID    uint32 `json:"instancedAbilityId"`
	InstancedModifierID   uint32 `json:"instancedModifierId"`
	IsServerBuffModifier  bool   `json:"isServerBuffModifier"`
	ModifierConfigLocalID int32  `json:"modifierConfigLocalId"`
}

type AbilityInvokeEntry struct {
	Head          *AbilityInvokeEntryHead `json:"head"`
	ForwardType   uint32                  `json:"forwardType"`
	ArgumentType  uint32                  `json:"argumentType"`
	ForwardPeer   uint32                  `json:"forwardPeer"`
	AbilityData   []byte                  `json:"abilityData"`
	EventID       uint32                  `json:"eventId"`
	EntityID      uint32                  `json:"entityId"`
	TotalTickTime float64                 `json:"totalTickTime"`
	IsIgnoreAuth  bool                    `json:"isIgnoreAuth"`
}

type ClientAbilityChangeNotify struct {
	EntityID   uint32                `json:"entityId"`
	IsInitHash bool                  `json:"isInitHash"`
	Invokes    []*AbilityInvokeEntry `json:"invokes"`
}

func (s *Session) OnClientAbilityChangeNotify(from, to mapper.Protocol, data []byte) ([]byte, error) {
	notify := new(ClientAbilityChangeNotify)
	err := json.Unmarshal(data, notify)
	if err != nil {
		return data, err
	}
	notify.Invokes = s.OnAbilityInvocations(from, to, notify.Invokes)
	return json.Marshal(notify)
}

type AbilityInvocationsNotify struct {
	Invokes []*AbilityInvokeEntry `json:"invokes"`
}

func (s *Session) OnAbilityInvocationsNotify(from, to mapper.Protocol, data []byte) ([]byte, error) {
	notify := new(AbilityInvocationsNotify)
	err := json.Unmarshal(data, notify)
	if err != nil {
		return data, err
	}
	notify.Invokes = s.OnAbilityInvocations(from, to, notify.Invokes)
	return json.Marshal(notify)
}

func (s *Session) OnAbilityInvocations(from, to mapper.Protocol, in []*AbilityInvokeEntry) []*AbilityInvokeEntry {
	var out []*AbilityInvokeEntry
	var err error
	for _, invoke := range in {
		if len(invoke.AbilityData) == 0 {
			out = append(out, invoke)
			continue
		}
		name := mapper.AbilityInvokeArguments[invoke.ArgumentType]
		if name == "" {
			logger.Debug().Msgf("Unknown ability invoke packet %d", invoke.ArgumentType)
			continue
		}
		invoke.AbilityData, err = s.ConvertPacketByName(from, to, name, invoke.AbilityData)
		if err != nil {
			logger.Debug().Err(err).Msgf("Failed to convert ability invoke packet %s", name)
			continue
		}
		out = append(out, invoke)
	}
	return out
}
