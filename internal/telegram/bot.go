package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Bot struct {
	Token string
}

func (b *Bot) apiURL(method string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.Token, method)
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
		"parse_mode": "HTML",
	}
	buf, _ := json.Marshal(body)
	resp, err := http.Post(b.apiURL("sendMessage"), "application/json", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram error: %s", resp.Status)
	}
	return nil
}
