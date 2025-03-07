package utils

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    
    "github.com/bestruirui/bestsub/config"
)

type WeworkMessage struct {
    MsgType string `json:"msgtype"`
    Text    struct {
        Content string `json:"content"`
    } `json:"text"`
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

    resp, err := http.Post(config.GlobalConfig.WeworkBot, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("send notification failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("notification API returned non-200 status code: %d", resp.StatusCode)
    }

    return nil
}