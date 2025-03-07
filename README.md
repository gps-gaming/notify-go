## Telegram & LineBot & Discord Notification

```go
message := string | []string

err := notify.New().
        Telegram(BotToken, ChatID).
        Line(AccessToken, ChatID).
        Discord(BotToken, ChannelID).
        DiscordWebhook(WebhookUrl).
        Send(message)
```