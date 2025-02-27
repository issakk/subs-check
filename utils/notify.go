package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const webhookURL = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=fdd31e44-d357-4b9d-95aa-d8d632cbada2"

type WeixinMessage struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func SendWeixinNotification(message string) error {
	msg := WeixinMessage{
		Msgtype: "text",
	}
	msg.Text.Content = message

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("通知发送失败，状态码: %d", resp.StatusCode)
	}

	return nil
}