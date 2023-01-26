package pkg

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

type Type string

const (
	ServerType Type = "server"
	ClientType      = "client"
)

type Config struct {
	ServiceType Type   `yaml:"type"`
	Address     string `yaml:"address"`
	Path        string `yaml:"path"`
}

func ReadConfig(file string) (*Config, error) {
	yfile, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	c := Config{}
	err = yaml.Unmarshal(yfile, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
