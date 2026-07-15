package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/gliedabrennung/notionist/internal/config"
	"github.com/gliedabrennung/notionist/internal/notion"
	adkagent "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

const agentName = "notionist"

type TaskAgent struct {
	runner *runner.Runner
}

func NewTaskAgent(cfg *config.Config) (*TaskAgent, error) {
	notionClient := notion.NewClient(cfg)

	notionTools, err := newNotionTools(notionClient)
	if err != nil {
		return nil, err
	}

	instruction, err := loadInstruction(os.Getenv("PROMPT_PATH"))
	if err != nil {
		return nil, fmt.Errorf("loading instruction: %w", err)
	}

	modelName := cfg.Gemini.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	model, err := gemini.NewModel(context.Background(), modelName, &genai.ClientConfig{
		APIKey: cfg.Gemini.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("creating gemini model: %w", err)
	}

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        agentName,
		Description: "Converts user messages into Notion tasks.",
		Model:       model,
		Instruction: instruction,
		Tools:       notionTools,
	})
	if err != nil {
		return nil, fmt.Errorf("creating llm agent: %w", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "notionist",
		Agent:             rootAgent,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("creating runner: %w", err)
	}

	return &TaskAgent{runner: r}, nil
}

func (a *TaskAgent) ProcessMessage(ctx context.Context, userID, sessionID, message string) (string, error) {
	content := &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: message}},
	}

	var reply string
	for ev, err := range a.runner.Run(ctx, userID, sessionID, content, adkagent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if ev.Content == nil || ev.Content.Role != "model" {
			continue
		}
		for _, p := range ev.Content.Parts {
			if p.Text != "" {
				reply = p.Text
			}
		}
	}

	if reply == "" {
		reply = "Done."
	}
	return reply, nil
}
