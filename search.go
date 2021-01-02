package main

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
)

// AmplitudeID wraps the httpRequest.headers response.
type AmplitudeID struct {
	DeviceID       string `json:"deviceId"`
	UserID         string `json:"userId"`
	OptOut         bool   `json:"optOut"`
	SessionID      int64  `json:"sessionId"`
	LastEventTime  int64  `json:"lastEventTime"`
	EventID        int    `json:"eventId"`
	IdentifyID     int    `json:"identifyId"`
	SequenceNumber int    `json:"sequenceNumber"`
}

var searchCommand = &cli.Command{
	Name:   "search",
	Usage:  "Search elasticsearch",
	Action: searchAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "query",
			Value:   "",
			Aliases: []string{"q"},
			Usage:   "Specify query json file",
		},
	},
}

func searchAction(c *cli.Context) error {
	// Search for the indexed documents
	//
	// Build the request body.
	w := c.App.Writer
	es, err := newClient(c)
	if err != nil {
		return err
	}

	filename := c.String("query")
	logrus.Debugf("filename: %s", filename)
	m, _ := time.ParseDuration("5m")
	query, err := buildQuery(filename)
	logrus.Debugf("query: %s", query)
	if err != nil {
		return err
	}
	// Perform the search request.
	page, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("log-aws-waf-*"),
		es.Search.WithBody(query),
		es.Search.WithPretty(),
		es.Search.WithSize(10000),
		es.Search.WithScroll(m),
		es.Search.WithSource("httpRequest.headers"),
	)
	if err != nil {
		return fmt.Errorf("Error getting response: %s", err)
	}
	defer page.Body.Close()

	if page.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(page.Body).Decode(&e); err != nil {
			return fmt.Errorf("Error parsing the response body: %s", err)
		}
		// Print the response status and error information.
		return fmt.Errorf("[%s] %s: %s",
			page.Status(),
			e["error"].(map[string]interface{})["type"],
			e["error"].(map[string]interface{})["reason"],
		)
	}

	var b bytes.Buffer
	b.ReadFrom(page.Body)
	total := gjson.GetBytes(b.Bytes(), "hits.total.value").Int()
	logrus.Debugf("total hits: %v", total)
	hits := int64(len(gjson.GetBytes(b.Bytes(), "hits.hits").Array()))
	logrus.Debugf("hits: %v", hits)
	took := gjson.GetBytes(b.Bytes(), "took").Int()
	sid := gjson.GetBytes(b.Bytes(), "_scroll_id").String()
	logrus.Debugf("sid: %v", sid)

	amplitudeIDs := make([]AmplitudeID, 0, hits)

	for _, hit := range gjson.GetBytes(b.Bytes(), "hits.hits").Array() {
		headers := gjson.Get(hit.Map()["_source"].String(), "httpRequest.headers").Array()
		for _, header := range headers {
			if header.Map()["name"].Str == "cookie" {
				cookie := header.Map()["value"].Str
				amplitudeID, err := cookieToAmplitudeID(cookie)
				if err != nil {
					return err
				}
				if amplitudeID != (AmplitudeID{}) {
					amplitudeIDs = append(amplitudeIDs, amplitudeID)
				}
			}
		}
	}

	for hits > 0 {
		res, err := es.Scroll(
			es.Scroll.WithScrollID(sid),
			es.Scroll.WithScroll(m),
		)
		if err != nil {
			return fmt.Errorf("Error getting response: %s", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			var e map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
				return fmt.Errorf("Error parsing the response body: %s", err)
			}
			// Print the response status and error information.
			return fmt.Errorf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}

		var b bytes.Buffer
		b.ReadFrom(res.Body)
		for _, hit := range gjson.GetBytes(b.Bytes(), "hits.hits").Array() {
			headers := gjson.Get(hit.Map()["_source"].String(), "httpRequest.headers").Array()
			for _, header := range headers {
				if header.Map()["name"].Str == "cookie" {
					cookie := header.Map()["value"].Str
					amplitudeID, err := cookieToAmplitudeID(cookie)
					if err != nil {
						return err
					}
					if amplitudeID != (AmplitudeID{}) {
						amplitudeIDs = append(amplitudeIDs, amplitudeID)
					}
				}
			}
		}
		hits = int64(len(gjson.GetBytes(b.Bytes(), "hits.hits").Array()))
		took += gjson.GetBytes(b.Bytes(), "took").Int()
		logrus.Debugf("hits: %v", hits)
		logrus.Debugf("amplitude Id: %v", len(amplitudeIDs))
		// in any case, only the most recently received _scroll_id should be used.
		// See: https://www.elastic.co/guide/en/elasticsearch/reference/master/paginate-search-results.html#scroll-search-results
		sid = gjson.GetBytes(b.Bytes(), "_scroll_id").String()
		logrus.Debugf("sid: %v", sid)
	}
	out, err := json.Marshal(&amplitudeIDs)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%v\n", string(out))

	logrus.Debugf("amplitude Id count: %v", len(amplitudeIDs))
	logrus.Debugf(
		"[%s] %d hits; took: %dms\n",
		page.Status(),
		total,
		took,
	)
	printAmplitudeIDSummary(amplitudeIDs)
	return nil
}

func printAmplitudeIDSummary(amplitudeIDs []AmplitudeID) {
	memo := make(map[string]int)
	for _, v := range amplitudeIDs {
		memo[v.UserID]++
	}
	type user struct {
		uuid  string
		count int
	}
	userIDs := make([]user, 0, len(memo))
	for k, v := range memo {
		userIDs = append(userIDs, user{uuid: k, count: v})
	}
	sort.SliceStable(userIDs, func(i, j int) bool {
		if userIDs[i].count == userIDs[j].count {
			return userIDs[i].uuid < userIDs[j].uuid
		}
		return userIDs[i].count < userIDs[j].count
	})
	for i, userID := range userIDs {
		logrus.Debugf("%v: %v:%v", i, userID.uuid, userID.count)
	}
}

// cookieToAmplitudeID is AmplitudeID extractiong from cookie
func cookieToAmplitudeID(cookie string) (AmplitudeID, error) {
	var amplitudeID AmplitudeID
	for _, values := range strings.Split(cookie, ";") {
		if strings.Contains(values, "amplitude_id") {
			sEnc := trimNextEqual(values)
			sDec, err := b64.StdEncoding.DecodeString(sEnc)
			if err != nil {
				return (AmplitudeID{}), fmt.Errorf("Error encodeings the amplitude_id value: %s", err)
			}
			err = json.Unmarshal(sDec, &amplitudeID)
			if err != nil {
				return (AmplitudeID{}), fmt.Errorf("Error to unmarshal JSON into AmplitudeID struct: %s", err)
			}
		}
	}
	return amplitudeID, nil
}

// trimNexEqual は最初の=の次から末尾までの文字列を返す
func trimNextEqual(s string) string {
	i := 0
	for i = 0; i < len(s); i++ {
		if s[i] == '=' {
			break
		}
	}
	return s[i+1:]
}

func buildQuery(filename string) (io.Reader, error) {
	if filename == "" {
		return strings.NewReader(query), nil
	}
	logrus.Debugf("filename: %v", filename)
	query, err := ioutil.ReadFile(filename)
	logrus.Debugf("query: %v", query)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(query), nil
}

const query = `{
  "query": {
    "bool": {
      "must": [
        {
          "match_all": {}
        }
      ],
      "filter": [
        {
          "bool": {
            "filter": [
              {
                "bool": {
                  "must_not": {
                    "bool": {
                      "should": [
                        {
                          "match": {
                            "ruleGroupList.terminatingRule.ruleId": "HostingProviderIPList"
                          }
                        }
                      ],
                      "minimum_should_match": 1
                    }
                  }
                }
              },
              {
                "bool": {
                  "filter": [
                    {
                      "bool": {
                        "should": [
                          {
                            "match": {
                              "ruleGroupList.terminatingRule.action": "BLOCK"
                            }
                          }
                        ],
                        "minimum_should_match": 1
                      }
                    },
                    {
                      "bool": {
                        "filter": [
                          {
                            "bool": {
                              "should": [
                                {
                                  "match": {
                                    "ruleGroupList.terminatingRule.ruleId": "AnonymousIPList"
                                  }
                                }
                              ],
                              "minimum_should_match": 1
                            }
                          },
                          {
                            "bool": {
                              "filter": [
                                {
                                  "bool": {
                                    "should": [
                                      {
                                        "range": {
                                          "httpRequest.clientIp": {
                                            "gte": "103.208.220.0"
                                          }
                                        }
                                      }
                                    ],
                                    "minimum_should_match": 1
                                  }
                                },
                                {
                                  "bool": {
                                    "should": [
                                      {
                                        "range": {
                                          "httpRequest.clientIp": {
                                            "lte": "103.208.223.255"
                                          }
                                        }
                                      }
                                    ],
                                    "minimum_should_match": 1
                                  }
                                }
                              ]
                            }
                          }
                        ]
                      }
                    }
                  ]
                }
              }
            ]
          }
        },
        {
          "match_phrase": {
            "rule.ruleset": "wafv2-linux"
          }
        },
        {
          "range": {
            "@timestamp": {
              "gte": "2020-12-18T02:23:36.977Z",
              "lte": "2020-12-25T02:23:36.977Z",
              "format": "strict_date_optional_time"
            }
          }
        }
      ]
    }
  },
  "sort": {"_doc":"asc"}
}`
