package crun

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"strings"
)

var DefaultConfigFile = "/etc/crun/crun.toml"
var DefaultMutexdir = "/tmp/crun"

type Config struct {
	PreHandlers        []string          `toml:"pre"`
	NoticeHandlers     []string          `toml:"notice"`
	PostHandlers       []string          `toml:"post"`
	SuccessHandlers    []string          `toml:"success"`
	FailureHandlers    []string          `toml:"failure"`
	LogFile            string            `toml:"log_file"`
	LogPrefix          string            `toml:"log_prefix"`
	Tag                string            `toml:"tag"`
	Quiet              bool              `toml:"quiet"`
	WorkingDirectory   string            `toml:"working_directory"`
	Mutexdir           string            `toml:"mutexdir"`
	Environment        []string          `toml:"environments"`
	EnvironmentMap     map[string]string `toml:"-"`
	WithoutOverlapping bool              `toml:"without_overlapping"`
	User               string            `toml:"user"`
	Group              string            `toml:"group"`
}

func newConfig() *Config {
	return &Config{
		PreHandlers:        []string{},
		NoticeHandlers:     []string{},
		PostHandlers:       []string{},
		SuccessHandlers:    []string{},
		FailureHandlers:    []string{},
		Environment:        []string{},
		Mutexdir:           DefaultMutexdir,
		EnvironmentMap:     map[string]string{},
		WithoutOverlapping: false,
	}
}
func (c *Config) LoadConfigFile(path string) error {
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) Prepare() error {
	for _, e := range c.Environment {
		splitString := strings.SplitN(e, "=", 2)
		if len(splitString) != 2 {
			return fmt.Errorf("invalid environment variable format '%s'. must be 'KEY=VALUE'", e)
		}
		c.EnvironmentMap[splitString[0]] = splitString[1]
	}
	return nil
}
