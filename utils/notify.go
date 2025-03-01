package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bestruirui/mihomo-check/config"
)

type WeixinMessage struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func SendWeixinNotification(message string) error {
	// 检查是否配置了webhook URL
	if config.GlobalConfig.WebhookURL == "" {
		return fmt.Errorf("未配置企业微信webhook URL")
	}

	msg := WeixinMessage{
		Msgtype: "text",
	}
	msg.Text.Content = message

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(config.GlobalConfig.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("通知发送失败，状态码: %d", resp.StatusCode)
	}

	return nil
}