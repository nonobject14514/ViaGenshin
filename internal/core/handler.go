package core

import (
	"encoding/json"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
)

func (s *Session) HandlePacket(from, to mapper.Protocol, name string, head, data []byte) ([]byte, error) {
	switch name {
	case "GetPlayerTokenReq":
		return s.OnGetPlayerTokenReq(from, to, data)
	case "GetPlayerTokenRsp":
		return s.OnGetPlayerTokenRsp(from, to, data)
	case "UnionCmdNotify":
		return s.OnUnionCmdNotify(from, to, data)
	case "ClientAbilityChangeNotify":
		return s.OnClientAbilityChangeNotify(from, to, data)
	case "AbilityInvocationsNotify":
		return s.OnAbilityInvocationsNotify(from, to, data)
	case "CombatInvocationsNotify":
		return s.OnCombatInvocationsNotify(from, to, data)
	}
	if !s.config.Console.Enabled {
		return data, nil
	}
	switch name {
	case "GetPlayerFriendListRsp":
		return s.OnGetPlayerFriendListRsp(from, to, data)
	case "PrivateChatReq":
		return s.OnPrivateChatReq(from, to, head, data)
	case "PrivateChatRsp":
		return s.OnPrivateChatRsp(from, to, data)
	case "PullPrivateChatReq":
		return s.OnPullPrivateChatReq(from, to, data)
	case "PullPrivateChatRsp":
		return s.OnPullPrivateChatRsp(from, to, data)
	case "PullRecentChatReq":
		return s.OnPullRecentChatReq(from, to, data)
	case "PullRecentChatRsp":
		return s.OnPullRecentChatRsp(from, to, data)
	case "MarkMapReq":
		return s.OnMarkMapReq(from, to, head, data)
	case "MarkMapRsp":
		return s.OnMarkMapRsp(from, to, head, data)
	}
	return data, nil
}

type UnionCmdNotify struct {
	CmdList []*UnionCmd `json:"cmdList"`
}

type UnionCmd struct {
	MessageID uint16 `json:"messageId"`
	Body      []byte `json:"body"`
}

func (s *Session) OnUnionCmdNotify(from, to mapper.Protocol, data []byte) ([]byte, error) {
	notify := new(UnionCmdNotify)
	err := json.Unmarshal(data, notify)
	if err != nil {
		return data, err
	}
	for _, cmd := range notify.CmdList {
		name := s.mapping.CommandNameMap[from][cmd.MessageID]
		cmd.MessageID = s.mapping.CommandPairMap[from][to][cmd.MessageID]
		cmd.Body, err = s.ConvertPacketByName(from, to, name, cmd.Body)
		if err != nil {
			return data, err
		}
	}
	return json.Marshal(notify)
}
