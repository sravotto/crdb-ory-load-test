package config

import (
	"fmt"
	"os"
	"time"

	"github.com/cockroachdb/errors"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Hydra struct {
		AdminAPI  string `yaml:"admin_api,omitempty"`
		PublicAPI string `yaml:"public_api,omitempty"`
	} `yaml:"hydra"`

	Kratos struct {
		AdminAPI  string `yaml:"admin_api,omitempty"`
		PublicAPI string `yaml:"public_api,omitempty"`
	} `yaml:"kratos"`

	Keto struct {
		WriteAPI string `yaml:"write_api,omitempty"`
		ReadAPI  string `yaml:"read_api,omitempty"`
	} `yaml:"keto"`

	Workload struct {
		Readers        int  `yaml:"readers"`
		ReadRatio      int  `yaml:"read_ratio"`
		DurationSec    int  `yaml:"duration_sec"`
		MaxRate        int  `yaml:"max_rate"`
		TolerateErrors bool `yaml:"tolerate_errors"`
	} `yaml:"workload"`
}

func (c Config) CheckHydra() error {
	if c.Hydra.AdminAPI == "" {
		return errors.New("hydra AdminAPI is not defined")
	}
	if c.Hydra.PublicAPI == "" {
		return errors.New("hydra PublicAPI is not defined")
	}
	return nil
}
func (c Config) CheckKratos() error {
	if c.Kratos.AdminAPI == "" {
		return errors.New("kratos AdminAPI is not defined")
	}
	if c.Kratos.PublicAPI == "" {
		return errors.New("kratos PublicAPI is not defined")
	}
	return nil
}
func (c Config) CheckKeto() error {
	if c.Keto.WriteAPI == "" {
		return errors.New("keto WriteAPI is not defined")
	}
	if c.Keto.ReadAPI == "" {
		return errors.New("keto ReadAPI is not defined")
	}
	return nil
}
func (c Config) Readers() int {
	return c.Workload.Readers
}
func (c Config) Writers() int {
	return c.Workload.Readers / c.Workload.ReadRatio
}
func (c Config) Duration() time.Duration {
	return time.Duration(c.Workload.DurationSec) * time.Second
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return config, nil
}
