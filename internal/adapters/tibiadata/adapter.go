package tibiadata

import (
	"net/http"
	"time"

	"death-level-tracker/internal/adapters/tibiadata/api"
	"death-level-tracker/internal/config"
)

type Adapter struct {
	client         *api.Client
	tibiaComClient *http.Client
	config         *config.Config
}

func NewAdapter(client *api.Client, cfg *config.Config) *Adapter {
	return &Adapter{
		client: client,
		config: cfg,
		tibiaComClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
