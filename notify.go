package notify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type INotify interface {
	Send(*http.Client, string) error
	SendRaw(*http.Client, map[string]interface{}) error
}

type Notify struct {
	Client   *http.Client
	BotToken string
	ChatID   string
	Notify   []INotify
}

func New() *Notify {
	return &Notify{
		Client: http.DefaultClient,
	}
}

func (n *Notify) Send(message interface{}) error {
	var newMessage string
	switch v := message.(type) {
	case []string:
		newMessage = strings.Join(v, "\n")
	case string:

		newMessage = v
	default:
		return errors.New("invalid message")
	}

	var errs []error
	for _, notify := range n.Notify {
		if err := notify.Send(n.Client, newMessage); err != nil {
			log.Println("notify send error", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (n *Notify) SendRaw(message map[string]interface{}) error {

	var errs []error
	for _, notify := range n.Notify {
		if err := notify.SendRaw(n.Client, message); err != nil {
			log.Println("notify send error", err)
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (n *Notify) Telegram(botToken, chatId string) *Notify {
	n.Notify = append(n.Notify, &telegram{
		BotToken: botToken,
		ChatID:   chatId,
	})
	return n
}

type telegram struct {
	BotToken string `json:"-"`
	ChatID   string `json:"chat_id"`
	Text     string `json:"text"`
}

func (t *telegram) Send(client *http.Client, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	t.Text = message

	jsonData, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API responded with status: %v", resp.Status)
	}

	return nil
}

func (t *telegram) SendRaw(client *http.Client, message map[string]interface{}) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)

	if _, ok := message["chat_id"]; !ok {
		message["chat_id"] = t.ChatID
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API responded with status: %v", resp.Status)
	}

	return nil
}

func (n *Notify) Line(botToken, chatId string) *Notify {
	n.Notify = append(n.Notify, &line{
		BotToken: botToken,
		ChatID:   chatId,
	})
	return n
}

type line struct {
	BotToken string        `json:"-"`
	ChatID   string        `json:"to"`
	Messages []interface{} `json:"messages"`
}

func (l *line) Send(client *http.Client, message string) error {
	l.Messages = append(l.Messages, map[string]interface{}{
		"type": "text",
		"text": message,
	})

	jsonData, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.BotToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LINE API responded with status: %v", resp.Status)
	}

	return nil
}
func (l *line) SendRaw(client *http.Client, message map[string]interface{}) error {
	l.Messages = append(l.Messages, message)

	jsonData, err := json.Marshal(l)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.line.me/v2/bot/message/push", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.BotToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LINE API responded with status: %v", resp.Status)
	}

	return nil
}

func (n *Notify) Discord(botToken, channelID string) *Notify {
	n.Notify = append(n.Notify, &discord{
		BotToken: botToken,
		ChatID:   channelID,
	})
	return n
}

type discord struct {
	BotToken string `json:"-"`
	ChatID   string `json:"-"`
	Content  string `json:"content"`
}

func (d *discord) Send(client *http.Client, message string) error {
	d.Content = message

	jsonData, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", d.ChatID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.BotToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API responded with status: %v", resp.Status)
	}

	return nil
}

func (d *discord) SendRaw(client *http.Client, message map[string]interface{}) error {

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	url := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", d.ChatID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bot "+d.BotToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord API responded with status: %v", resp.Status)
	}

	return nil
}
