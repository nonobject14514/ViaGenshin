package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Jx2f/ViaGenshin/pkg/logger"
)

const (
	consoleUid         = uint32(1)
	consoleNickname    = "望星忆君"
	consoleLevel       = uint32(60)
	consoleWorldLevel  = uint32(8)
	consoleSignature   = ""
	consoleNameCardId  = uint32(210001)
	consoleAvatarId    = uint32(10000077)
	consoleCostumeId   = uint32(0)
	consoleWelcomeText = "望星开发服 gio 3.4 dev 进度缓慢，推荐出门左转前往桜开发服"
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
	var values []string
	values = append(values, fmt.Sprintf("cmd=%d", cmd))
	values = append(values, fmt.Sprintf("uid=%d", uid))
	values = append(values, fmt.Sprintf("msg=%s", text))
	values = append(values, fmt.Sprintf("region=%s", s.config.Console.MuipRegion))
	ticket := make([]byte, 16)
	if _, err := rand.Read(ticket); err != nil {
		return "", fmt.Errorf("failed to generate ticket: %w", err)
	}
	values = append(values, fmt.Sprintf("ticket=%x", ticket))
	if s.config.Console.MuipSign != "" {
		shaSum := sha256.New()
		sort.Strings(values)
		shaSum.Write([]byte(strings.Join(values, "&") + s.config.Console.MuipSign))
		values = append(values, fmt.Sprintf("sign=%x", shaSum.Sum(nil)))
	}
	uri := s.config.Console.MuipEndpoint + "?" + strings.ReplaceAll(strings.Join(values, "&"), " ", "+")
	logger.Debug().Msgf("Muip request: %s", uri)
	resp, err := http.Get(uri)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	p, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	logger.Debug().Msgf("Muip response: %s", string(p))
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body := new(MuipResponseBody)
	if err := json.Unmarshal(p, body); err != nil {
		return "", err
	}
	if body.Retcode != 0 {
		return "Failed to execute command: " + body.Data.Msg + ", error: " + body.Msg + body.Data.Msg, nil
	}
	return "Successfully executed command: " + body.Data.Msg, nil
}
