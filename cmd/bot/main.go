package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/gliedabrennung/notionist/internal/agent"
	"github.com/gliedabrennung/notionist/internal/config"
	"github.com/gliedabrennung/notionist/internal/telegram"
)

func main() {
	cfg, err := config.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	taskAgent, err := agent.NewTaskAgent(cfg)
	if err != nil {
		log.Fatal(err)
	}

	b, err := telegram.NewBot(cfg.Telegram.Token, telegram.NewMessageHandler(taskAgent))
	if err != nil {
		log.Fatal(err)
	}

	go b.Run(ctx)
	<-ctx.Done()
}
