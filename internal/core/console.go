package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

const (
	consoleUid         = uint32(99)
	consoleNickname    = "Console"
	consoleLevel       = uint32(60)
	consoleWorldLevel  = uint32(8)
	consoleSignature   = ""
	consoleNameCardId  = uint32(210001)
	consoleAvatarId    = uint32(10000007)
	consoleCostumeId   = uint32(0)
	consoleWelcomeText = "You can type GM commands here."
)

type MuipResponseBody struct {
	Retcode int32  `json:"retcode"`
	Msg     string `json:"msg"`
	Ticket  string `json:"ticket"`
	Data    struct {
		Msg    string `json:"msg"`
		Retmsg string `json:"retmsg"`
	} `json:"data"`
}

func (s *Server) ConsoleExecute(cmd, uid uint32, text string) (string, error) {
	logger.Info().Uint32("uid", uid).Msgf("ConsoleExecute: %s", text)
	uri := s.config.Console.MuipEndpoint + fmt.Sprintf("?cmd=%d&uid=%d&msg=%s", cmd, uid, url.QueryEscape(text))
	resp, err := http.Get(uri)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body := new(MuipResponseBody)
	if err := json.NewDecoder(resp.Body).Decode(body); err != nil {
		return "", err
	}
	if body.Retcode != 0 {
		return "Failed to execute command: " + body.Data.Msg + ", error: " + body.Data.Retmsg, nil
	}
	return "Successfully executed command: " + body.Data.Msg, nil
}
