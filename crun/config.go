package crun

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"strings"
)

var DefaultConfigFile = "/etc/crun/crun.toml"

type Config struct {
	PreHandlers      []string          `toml:"pre"`
	NoticeHandlers   []string          `toml:"notice"`
	PostHandlers     []string          `toml:"post"`
	SuccessHandlers  []string          `toml:"success"`
	FailureHandlers  []string          `toml:"failure"`
	LogFile          string            `toml:"log_file"`
	LogPrefix        string            `toml:"log_prefix"`
	Tag              string            `toml:"tag"`
	Quiet            bool              `toml:"quiet"`
	WorkingDirectory string            `toml:"working_directory"`
	Environment      []string          `toml:"environments"`
	EnvironmentMap   map[string]string `toml:"-"`
	InitByLua        string            `toml:"init_by_lua"`
}

func newConfig() *Config {
	return &Config{
		PreHandlers:     []string{},
		NoticeHandlers:  []string{},
		PostHandlers:    []string{},
		SuccessHandlers: []string{},
		FailureHandlers: []string{},
		Environment:     []string{},
		EnvironmentMap:  map[string]string{},
	}
}
func (c *Config) LoadConfigFile(path string) error {
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) ParseEnvironment() error {
	for _, e := range c.Environment {
		splitString := strings.SplitN(e, "=", 2)
		if len(splitString) != 2 {
			return fmt.Errorf("invalid environment variable format '%s'. must be 'KEY=VALUE'", e)
		}
		c.EnvironmentMap[splitString[0]] = splitString[1]
	}
	return nil
}
