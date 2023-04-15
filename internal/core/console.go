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
	consoleWelcomeText = "望星开发服 gio 3.4 dev \n进度缓慢，推荐出门左转前往桜开发服"
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
	logger.Info().Uint32("uid", uid).Msgf("控制台执行: %s", text)
	var values []string
	values = append(values, fmt.Sprintf("cmd=%d", cmd))
	values = append(values, fmt.Sprintf("uid=%d", uid))
	values = append(values, fmt.Sprintf("msg=%s", text))
	values = append(values, fmt.Sprintf("region=%s", s.config.Console.MuipRegion))
	ticket := make([]byte, 16)
	if _, err := rand.Read(ticket); err != nil {
		return "", fmt.Errorf("无法生成 ticket: %w", err)
	}
	values = append(values, fmt.Sprintf("ticket=%x", ticket))
	if s.config.Console.MuipSign != "" {
		shaSum := sha256.New()
		sort.Strings(values)
		shaSum.Write([]byte(strings.Join(values, "&") + s.config.Console.MuipSign))
		values = append(values, fmt.Sprintf("sign=%x", shaSum.Sum(nil)))
	}
	uri := s.config.Console.MuipEndpoint + "?" + strings.ReplaceAll(strings.Join(values, "&"), " ", "+")
	logger.Debug().Msgf("Muip 响应: %s", uri)
	resp, err := http.Get(uri)
	if err != nil {
		return "Muip 响应: %s" + "\n温馨提示" + consoleWelcomeText, err
	}
	defer resp.Body.Close()
	p, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	logger.Debug().Msgf("Muip 响应: %s", string(p))
	if resp.StatusCode != 200 {
		return "Muip 响应: %s" + "\n温馨提示" + consoleWelcomeText, fmt.Errorf("状态码: %d", resp.StatusCode)
	}
	body := new(MuipResponseBody)
	if err := json.Unmarshal(p, body); err != nil {
		return "", err
	}
	if (text == "help") {
		return "gm指令往此输入", nil
	}
	if body.Retcode != 0 {
		return "执行命令失败: " + body.Data.Msg + ", 错误: " + body.Msg + "\n温馨提示" + consoleWelcomeText, nil
	}
	return "执行命令成功: " + body.Data.Msg + "\n温馨提示" + consoleWelcomeText, nil
}
