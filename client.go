package main

import (
	"crypto/x509"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func newClient(c *cli.Context) (*elasticsearch.Client, error) {
	var err error
	tp := http.DefaultTransport.(*http.Transport).Clone()

	if tp.TLSClientConfig.RootCAs, err = x509.SystemCertPool(); err != nil {
		return nil, err
	}

	address := c.String("address")
	username := c.String("username")
	password := c.String("password")
	cfg := elasticsearch.Config{
		Addresses: []string{address},
		Username:  username,
		Password:  password,
		Transport: tp,
		Logger:    &CustomLogger{log.Logger},
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return es, nil
}
