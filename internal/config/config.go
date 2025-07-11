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
		AdminAPI  string `yaml:"admin_api",omitempty`
		PublicAPI string `yaml:"public_api",omitempty`
	} `yaml:"hydra"`

	Kratos struct {
		AdminAPI  string `yaml:"admin_api",omitempty`
		PublicAPI string `yaml:"public_api",omitempty`
	} `yaml:"kratos"`

	Keto struct {
		WriteAPI string `yaml:"write_api",omitempty`
		ReadAPI  string `yaml:"read_api",omitempty`
	} `yaml:"keto"`

	Workload struct {
		Readers     int `yaml:"readers"`
		ReadRatio   int `yaml:"read_ratio"`
		DurationSec int `yaml:"duration_sec"`
	} `yaml:"workload"`
}

func (c Config) CheckHydra() error {
	if c.Hydra.AdminAPI == "" {
		return errors.New("hydra AdminAPI is not defined")
	}
	if c.Hydra.AdminAPI == "" {
		return errors.New("hydra AdminAPI is not defined")
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

var AppConfig Config

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	fmt.Println(AppConfig)
	return nil
}
