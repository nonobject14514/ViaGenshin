package core

import (
	"encoding/json"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
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
	var invokes []*AbilityInvokeEntry
	for _, invoke := range notify.Invokes {
		if len(invoke.AbilityData) == 0 {
			invokes = append(invokes, invoke)
			continue
		}
		name := mapper.AbilityInvokeArgumentTypes[invoke.ArgumentType]
		if name == "" {
			continue
		}
		invoke.AbilityData, err = s.ConvertPacketByName(from, to, name, invoke.AbilityData)
		if err != nil {
			return data, err
		}
		invokes = append(invokes, invoke)
	}
	notify.Invokes = invokes
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
	var invokes []*AbilityInvokeEntry
	for _, invoke := range notify.Invokes {
		if len(invoke.AbilityData) == 0 {
			invokes = append(invokes, invoke)
			continue
		}
		name := mapper.AbilityInvokeArgumentTypes[invoke.ArgumentType]
		if name == "" {
			continue
		}
		invoke.AbilityData, err = s.ConvertPacketByName(from, to, name, invoke.AbilityData)
		if err != nil {
			return data, err
		}
		invokes = append(invokes, invoke)
	}
	notify.Invokes = invokes
	return json.Marshal(notify)
}
