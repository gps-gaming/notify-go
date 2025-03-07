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
	Client    *http.Client
	BotToken  string
	ChatID    string
	Notifiers []INotify
}

func New() *Notify {
	return &Notify{
		Client: http.DefaultClient,
	}
}

func (n *Notify) Send(message interface{}) error {
	var errs []error

	switch msg := message.(type) {
	case string:
		for _, notify := range n.Notifiers {
			if err := notify.Send(n.Client, msg); err != nil {
				log.Println("notify send error", err)
				errs = append(errs, err)
			}
		}

	case []string:
		newMessage := strings.Join(msg, "\n")
		for _, notify := range n.Notifiers {
			if err := notify.Send(n.Client, newMessage); err != nil {
				log.Println("notify send error", err)
				errs = append(errs, err)
			}
		}

	case map[string]interface{}:
		// 處理 Raw message
		for _, notify := range n.Notifiers {
			if err := notify.SendRaw(n.Client, msg); err != nil {
				log.Println("notify send error", err)
				errs = append(errs, err)
			}
		}

	default:
		return errors.New("invalid message format")
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func request(client *http.Client, req *http.Request) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf(req.Host, " API responded with status: %v", resp.Status)
	}

	return nil
}

func (n *Notify) Telegram(botToken, chatId string) *Notify {
	n.Notifiers = append(n.Notifiers, &telegram{
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

	return request(client, req)
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

	return request(client, req)
}

func (n *Notify) Line(botToken, chatId string) *Notify {
	n.Notifiers = append(n.Notifiers, &line{
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

	return request(client, req)
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

	return request(client, req)
}

func (n *Notify) Discord(botToken, channelID string) *Notify {
	n.Notifiers = append(n.Notifiers, &discord{
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

	return request(client, req)
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

	return request(client, req)
}

func (n *Notify) DiscordWebhook(webHookUrl string) *Notify {
	n.Notifiers = append(n.Notifiers, &discordWebhook{
		WebhookUrl: webHookUrl,
	})
	return n
}

type discordWebhook struct {
	WebhookUrl string
	Content    string `json:"content"`
}

func (d *discordWebhook) Send(client *http.Client, message string) error {
	d.Content = message

	jsonData, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", d.WebhookUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return request(client, req)
}

func (d *discordWebhook) SendRaw(client *http.Client, message map[string]interface{}) error {

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", d.WebhookUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	return request(client, req)
}
