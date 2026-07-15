package telegram

import (
	"context"
	"fmt"
	"log"

	"github.com/gliedabrennung/notionist/internal/agent"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Bot struct {
	bot *bot.Bot
}

func NewBot(token string, handler bot.HandlerFunc, opts ...bot.Option) (*Bot, error) {
	opts = append([]bot.Option{bot.WithDefaultHandler(handler)}, opts...)
	b, err := bot.New(token, opts...)
	if err != nil {
		return nil, err
	}
	return &Bot{bot: b}, nil
}

func (b *Bot) Run(ctx context.Context) {
	b.bot.Start(ctx)
}

func NewMessageHandler(taskAgent *agent.TaskAgent) bot.HandlerFunc {
	return func(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
		if update.Message == nil {
			return
		}

		chatID := update.Message.Chat.ID
		userID := fmt.Sprintf("%d", update.Message.From.ID)
		sessionID := fmt.Sprintf("tg_%d", chatID)
		messageText := update.Message.Text

		if messageText == "" || messageText == "/start" {
			if _, err := tgBot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Send me a task and I'll create it in Notion. e.g. \"Buy milk tomorrow #groceries #urgent\"",
			}); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			return
		}

		reply, err := taskAgent.ProcessMessage(ctx, userID, sessionID, messageText)
		if err != nil {
			log.Printf("failed to handle message: %v", err)
			reply = "Failed to process your message: " + err.Error()
		}

		if reply == "" {
			reply = "Done."
		}

		if _, err := tgBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   reply,
		}); err != nil {
			log.Printf("failed to send reply: %v", err)
		}
	}
}
