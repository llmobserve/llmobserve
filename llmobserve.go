package llmobserve

import (
	"context"
	"fmt"

	"github.com/llmobserve/llmobserve/providers"
	"github.com/llmobserve/llmobserve/tracer"
	openai "github.com/openai/openai-go/v3"

	"log/slog"

	dashboard "github.com/llmobserve/llmobserve/dashboard"
	"github.com/llmobserve/llmobserve/models"
	"github.com/llmobserve/llmobserve/storage"
	ollama "github.com/ollama/ollama/api"
)

// create a init function return new llmobserve init the storage on the basis of path

type Config struct {
	StoragePath string
	DevMode     bool
}

type Scope struct {
	tracer *tracer.Tracer
}

func New(cfg Config) (*Scope, error) {
	if cfg.StoragePath == "" {
		cfg.StoragePath = "./llmobserve"
	}

	sqlStorage, err := storage.NewSQLiteStorage(cfg.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("llmobserve: initialising storage: %w", err)
	}

	t := tracer.New(sqlStorage)

	if cfg.DevMode {
		go dashboard.Start(t, 7890)
	}

	return &Scope{tracer: t}, nil
}

func (s *Scope) WrapOpenAI(client *openai.Client, projectID string) *providers.OpenAIClient {
	s.ensureProject(context.Background(), projectID)
	return providers.WrapOpenAI(client, s.tracer, projectID)
}

func (s *Scope) WrapOllama(client *ollama.Client, projectID string) *providers.OllamaClient {
	s.ensureProject(context.Background(), projectID)
	return providers.WrapOllama(client, s.tracer, projectID)
}

// ensureProject creates a project record if it doesn't already exist.
// Uses the projectID as the name since users pass IDs not names here.
func (s *Scope) ensureProject(ctx context.Context, projectID string) {
	// check if project already exists
	_, err := s.tracer.GetProject(ctx, projectID)
	if err == nil {
		return // already exists
	}

	// create it
	project := models.NewProject(projectID)
	project.ProjectID = projectID // use the user's ID not a generated one

	if err := s.tracer.SaveProject(ctx, project); err != nil {
		slog.Warn("llmobserve: failed to save project",
			"project_id", projectID,
			"error", err,
		)
	}
}

func (s *Scope) Close() error {
	return s.tracer.Close()
}
