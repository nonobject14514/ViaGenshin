package core

import (
	"encoding/json"

	"github.com/Jx2f/ViaGenshin/internal/mapper"
	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

type GetPlayerFriendListRsp struct {
	Retcode       int32             `json:"retcode,omitempty"`
	AskFriendList []*map[string]any `json:"askFriendList,omitempty"`
	FriendList    []*map[string]any `json:"friendList,omitempty"`
}

func (s *Session) OnGetPlayerFriendListRsp(from, to mapper.Protocol, data []byte) ([]byte, error) {
	packet := new(GetPlayerFriendListRsp)
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return data, err
	}
	packet.FriendList = append(packet.FriendList, &map[string]any{
		"uid":        consoleUid,
		"nickname":   consoleNickname,
		"level":      consoleLevel,
		"worldLevel": consoleWorldLevel,
		"signature":  consoleSignature,
		"nameCardId": consoleNameCardId,
		"profilePicture": map[string]any{
			"avatarId":  consoleAvatarId,
			"costumeId": consoleCostumeId,
		},
		"isGameSource": true,
		"onlineState":  uint32(1),
		"platformType": uint32(3),
	})
	data, err = json.Marshal(packet)
	if err != nil {
		return data, err
	}
	logger.Debug().Msgf("Injecting GetPlayerFriendListRsp: %s", data)
	return data, nil
}
