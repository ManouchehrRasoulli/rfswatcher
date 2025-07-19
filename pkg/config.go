package pkg

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Type string

type ServerTLSConfig struct {
	Key  string `yaml:"key"`
	Cert string `yaml:"cert"`
}

type ServerConfig struct {
	PwFile string          `yaml:"pwfile"`
	TLS    ServerTLSConfig `yaml:"tls"`
}

type ClientConfig struct {
	TLS      bool   `yaml:"tls"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

const (
	ServerType Type = "server"
	ClientType      = "client"
)

type Config struct {
	ServiceType Type         `yaml:"type"`
	Address     string       `yaml:"address"`
	Path        string       `yaml:"path"`
	Client      ClientConfig `yaml:"client"`
	Server      ServerConfig `yaml:"server"`
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
