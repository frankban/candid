// Copyright 2014 Canonical Ltd.

// The config package defines configuration parameters for the id server.
package config

import (
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/errgo.v1"
	"gopkg.in/yaml.v1"
)

// Config holds the configuration parameters for the identity service.
type Config struct {
	MongoAddr string `yaml:"mongo-addr"`
	APIAddr   string `yaml:"api-addr"`
}

func (c *Config) validate() error {
	var missing []string
	if c.MongoAddr == "" {
		missing = append(missing, "mongo-addr")
	}
	if c.APIAddr == "" {
		missing = append(missing, "api-addr")
	}
	if len(missing) != 0 {
		return errgo.Newf("missing fields %s in config file", strings.Join(missing, ", "))
	}
	return nil
}

// Read reads an identity configuration file from the given path.
func Read(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errgo.Notef(err, "cannot open config file")
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errgo.Notef(err, "cannot read %q", path)
	}
	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return nil, errgo.Notef(err, "cannot parse %q", path)
	}
	if err := conf.validate(); err != nil {
		return nil, errgo.Mask(err)
	}
	return &conf, nil
}