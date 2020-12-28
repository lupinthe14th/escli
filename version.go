package main

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/lupinthe14th/escli/pkg/version"
	"github.com/urfave/cli/v2"
)

// Info wraps the Elasticsearch infomation response.
type Info struct {
	Version *struct {
		Number string `json:"number"`
	} `json:"version"`
}

var versionCommand = &cli.Command{
	Name:   "version",
	Usage:  "Shows the version information",
	Action: versionAction,
}

func versionAction(c *cli.Context) error {
	w := c.App.Writer
	fmt.Fprintf(w, "Client:\n")
	fmt.Fprintf(w, " Version:\t%s\n", version.Version)
	fmt.Fprintf(w, " Git commit:\t%s\n", version.Revision)
	fmt.Fprintf(w, "Elasticsearch:\n")
	fmt.Fprintf(w, " Version:\t%s\n", elasticsearch.Version)

	es, err := newClient(c)
	if err != nil {
		return err
	}
	res, err := es.Info()
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check response status
	if res.IsError() {
		return fmt.Errorf("%s", res.String())
	}

	// Deserialize the response into a map.
	var info Info
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return err
	}
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Server:\n")
	fmt.Fprintf(w, " Version:\t%s\n", info.Version.Number)
	return nil
}
