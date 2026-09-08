package config

import (
	"encoding/json"
	"os"

	"github.com/karnesanthosh/Averon/internal/server"
)

// RouteConfig represents a single route in the gateway configuration.
type RouteConfig struct {
	Path        string           `json:"path"`
	StripPrefix bool             `json:"strip_prefix"`
	RateLimit *RateLimitConfig   `json:"rate_limit"`
	Auth *Authconfig `json:"auth"`
	Backends    []*server.Server `json:"backends"`
}

// GatewayConfig is the top-level configuration for NebulaGate.
type GatewayConfig struct {
	Port                int           `json:"port"`
	HealthCheckInterval string        `json:"health_check_interval"`
	Routes              []RouteConfig `json:"routes"`
}

type RateLimitConfig struct{
	RequestPerSecond float64 `json:"requests_per_second"`
	Burst int `json:"burst"`
}

type Authconfig struct {
	Type string `json:"type"`
	Secret string `json:"secret"`
	SkipPaths []string `json:"skip_paths"`

}



// LoadConfig loads the full gateway configuration from a JSON file.
// This supports the new route-based config format.
func LoadConfig(path string) (*GatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg GatewayConfig
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	// Default port
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	// Default health check interval
	if cfg.HealthCheckInterval == "" {
		cfg.HealthCheckInterval = "10s"
	}

	// Mark all backends as alive initially
	for i := range cfg.Routes {
		for _, s := range cfg.Routes[i].Backends {
			s.SetAlive(true)
		}
	}

	return &cfg, nil
}

// LoadServers loads backend servers from a flat JSON array.
// Kept for backward compatibility with the old config format.
func LoadServers(path string) ([]*server.Server, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var servers []*server.Server

	err = json.Unmarshal(data, &servers)
	if err != nil {
		return nil, err
	}

	for _, s := range servers {
		s.SetAlive(true)
	}

	return servers, nil
}
