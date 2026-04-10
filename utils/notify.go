package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bestruirui/bestsub/config"
)

type WeworkMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

type WeworkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func SendWeworkNotification(message string) error {
	if config.GlobalConfig.WeworkBot == "" {
		return nil
	}

	msg := WeworkMessage{
		MsgType: "text",
	}
	msg.Text.Content = message

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(config.GlobalConfig.WeworkBot, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("send notification failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read notification response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification API returned non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var weworkResp WeworkResponse
	if len(body) > 0 && json.Unmarshal(body, &weworkResp) == nil && weworkResp.ErrCode != 0 {
		return fmt.Errorf("notification API returned errcode %d: %s", weworkResp.ErrCode, weworkResp.ErrMsg)
	}

	return nil
}
