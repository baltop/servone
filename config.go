package main

import (
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type EndpointConfig struct {
	Path     string         `yaml:"path"`
	Method   string         `yaml:"method"`
	Response ResponseConfig `yaml:"response"`
}

type ResponseConfig struct {
	Status  int               `yaml:"status"`
	Body    string            `yaml:"body"`
	Headers map[string]string `yaml:"headers"`
}

func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}