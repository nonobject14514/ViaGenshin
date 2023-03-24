package core

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
	"github.com/Jx2f/ViaGenshin/pkg/transport/kcp"
)

type SystemHint struct {
	Type int32 `json:"type,omitempty"`
}

type ChatInfo struct {
	Time       uint32      `json:"time,omitempty"`
	Sequence   uint32      `json:"sequence,omitempty"`
	ToUid      uint32      `json:"toUid,omitempty"`
	Uid        uint32      `json:"uid,omitempty"`
	IsRead     bool        `json:"isRead,omitempty"`
	Text       string      `json:"text,omitempty"`
	Icon       uint32      `json:"icon,omitempty"`
	SystemHint *SystemHint `json:"systemHint,omitempty"`
}

type PrivateChatNotify struct {
	ChatInfo *ChatInfo `json:"chatInfo,omitempty"`
}

func (s *Session) NotifyPrivateChat(toSession *kcp.Session, to mapper.Protocol, head []byte, chatInfo *ChatInfo) error {
	packet := new(PrivateChatNotify)
	packet.ChatInfo = chatInfo
	data, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	logger.Debug().Msgf("Injecting PrivateChatNotify: %s", data)
	return s.SendPacketJSON(toSession, to, "PrivateChatNotify", head, data)
}

type PrivateChatReq struct {
	TargetUid uint32 `json:"targetUid,omitempty"`
	Text      string `json:"text,omitempty"`
	Icon      uint32 `json:"icon,omitempty"`
}

func (s *Session) OnPrivateChatReq(from, to mapper.Protocol, head, data []byte) ([]byte, error) {
	packet := new(PrivateChatReq)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	s.injectPrivateChat = packet.TargetUid == consoleUid
	if !s.injectPrivateChat {
		return data, nil
	}
	logger.Debug().Msgf("Injecting PrivateChatReq: %s", data)
	if err = s.NotifyPrivateChat(s.endpoint, from, head, &ChatInfo{
		Time:  uint32(time.Now().Unix()),
		ToUid: consoleUid,
		Uid:   s.playerUid,
		Text:  packet.Text,
		Icon:  packet.Icon,
	}); err != nil {
		return data, err
	}
	if packet.Text == "" {
		return data, nil
	}
	packet.Text, err = s.ConsoleExecute(1116, s.playerUid, packet.Text)
	if err != nil {
		packet.Text = fmt.Sprintf("Failed to execute command: %s", err)
	}
	return data, s.NotifyPrivateChat(s.endpoint, from, head, &ChatInfo{
		Time:  uint32(time.Now().Unix()),
		ToUid: s.playerUid,
		Uid:   consoleUid,
		Text:  packet.Text,
	})
}

type PrivateChatRsp struct {
	ChatForbiddenEndtime uint32 `json:"chatForbiddenEndtime,omitempty"`
	Retcode              int32  `json:"retcode,omitempty"`
}

func (s *Session) OnPrivateChatRsp(from, to mapper.Protocol, data []byte) ([]byte, error) {
	if !s.injectPrivateChat {
		return data, nil
	}
	s.injectPrivateChat = false
	packet := new(PrivateChatRsp)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	packet.Retcode = 0
	data, err = json.Marshal(packet)
	if err != nil {
		return data, err
	}
	logger.Debug().Msgf("Injecting PrivateChatRsp: %s", data)
	return data, nil
}

type PullPrivateChatReq struct {
	TargetUid     uint32 `json:"targetUid,omitempty"`
	PullNum       uint32 `json:"pullNum,omitempty"`
	BeginSequence uint32 `json:"beginSequence,omitempty"`
}

func (s *Session) OnPullPrivateChatReq(from, to mapper.Protocol, data []byte) ([]byte, error) {
	packet := new(PullPrivateChatReq)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	s.injectPullPrivateChat = packet.TargetUid == consoleUid
	if !s.injectPullPrivateChat {
		return data, nil
	}
	logger.Debug().Msgf("Injecting PullPrivateChatReq: %s", data)
	return data, nil
}

type PullPrivateChatRsp struct {
	ChatInfo []*ChatInfo `json:"chatInfo,omitempty"`
	Retcode  int32       `json:"retcode,omitempty"`
}

func (s *Session) OnPullPrivateChatRsp(from, to mapper.Protocol, data []byte) ([]byte, error) {
	if !s.injectPullPrivateChat {
		return data, nil
	}
	s.injectPullPrivateChat = false
	packet := new(PullPrivateChatRsp)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	packet.ChatInfo = append(packet.ChatInfo, &ChatInfo{
		Time:  uint32(time.Now().Unix()),
		ToUid: s.playerUid,
		Uid:   consoleUid,
		Text:  consoleWelcomeText,
	})
	packet.Retcode = 0
	data, err = json.Marshal(packet)
	if err != nil {
		return data, err
	}
	logger.Debug().Msgf("Injecting PullPrivateChatRsp: %s", data)
	return data, nil
}

type PullRecentChatReq struct {
	PullNum       uint32 `json:"pullNum,omitempty"`
	BeginSequence uint32 `json:"beginSequence,omitempty"`
}

func (s *Session) OnPullRecentChatReq(from, to mapper.Protocol, data []byte) ([]byte, error) {
	packet := new(PullRecentChatReq)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	s.injectPullRecentChat = packet.BeginSequence == 0
	if !s.injectPullRecentChat {
		return data, nil
	}
	logger.Debug().Msgf("Injecting PullRecentChatReq: %s", data)
	return data, nil
}

type PullRecentChatRsp struct {
	ChatInfo []*ChatInfo `json:"chatInfo,omitempty"`
	Retcode  int32       `json:"retcode,omitempty"`
}

func (s *Session) OnPullRecentChatRsp(from, to mapper.Protocol, data []byte) ([]byte, error) {
	if !s.injectPullRecentChat {
		return data, nil
	}
	s.injectPullRecentChat = false
	packet := new(PullRecentChatRsp)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	packet.ChatInfo = append(packet.ChatInfo, &ChatInfo{
		Time:  uint32(time.Now().Unix()),
		ToUid: s.playerUid,
		Uid:   consoleUid,
		Text:  consoleWelcomeText,
	})
	packet.Retcode = 0
	data, err = json.Marshal(packet)
	if err != nil {
		return data, err
	}
	logger.Debug().Msgf("Injecting PullRecentChatRsp: %s", data)
	return data, nil
}
