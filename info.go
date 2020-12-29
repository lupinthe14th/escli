package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/lupinthe14th/escli/pkg/version"
	"github.com/urfave/cli/v2"
)

// Info wraps the Elasticsearch information response.
type Info struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	Version     struct {
		Number                           string    `json:"number"`
		BuildFlavor                      string    `json:"build_flavor"`
		BuildType                        string    `json:"build_type"`
		BuildHash                        string    `json:"build_hash"`
		BuildDate                        time.Time `json:"build_date"`
		BuildSnapshot                    bool      `json:"build_snapshot"`
		LuceneVersion                    string    `json:"lucene_version"`
		MinimumWireCompatibilityVersion  string    `json:"minimum_wire_compatibility_version"`
		MinimumIndexCompatibilityVersion string    `json:"minimum_index_compatibility_version"`
	} `json:"version"`
	Tagline string `json:"tagline"`
}

var infoCommand = &cli.Command{
	Name:   "info",
	Usage:  "Display system-wide information",
	Action: infoAction,
}

func infoAction(c *cli.Context) error {
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
	fmt.Fprintf(w, " Name:\t%s\n", info.Name)
	fmt.Fprintf(w, " Cluster Name:\t%s\n", info.ClusterName)
	fmt.Fprintf(w, " Cluster UUID:\t%s\n", info.ClusterUUID)
	fmt.Fprintf(w, " Version:\n")
	fmt.Fprintf(w, "  Number:\t%s\n", info.Version.Number)
	fmt.Fprintf(w, "  Build Flavor:\t%s\n", info.Version.BuildFlavor)
	fmt.Fprintf(w, "  Build Type:\t%s\n", info.Version.BuildType)
	fmt.Fprintf(w, "  Build Hash:\t%s\n", info.Version.BuildHash)
	fmt.Fprintf(w, "  Build Date:\t%s\n", info.Version.BuildDate.Format(time.RFC3339Nano))
	fmt.Fprintf(w, "  Build Snapshot:\t%t\n", info.Version.BuildSnapshot)
	fmt.Fprintf(w, "  Lucene Version:\t%s\n", info.Version.LuceneVersion)
	fmt.Fprintf(w, "  Minimum Wire Compatibility Version:\t%s\n", info.Version.MinimumWireCompatibilityVersion)
	fmt.Fprintf(w, "  Minimum Index Compatibility Version:\t%s\n", info.Version.MinimumIndexCompatibilityVersion)
	fmt.Fprintf(w, " Tagline:\t%s\n", info.Tagline)
	return nil
}
