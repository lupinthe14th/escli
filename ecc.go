package main

import (
	"bytes"
	"context"
	"crypto/x509"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/tidwall/gjson"
)

// AmplitudeID wraps the httpresponce.header response.
type AmplitudeID struct {
	DeviceID       string `json:"deviceId"`
	UserID         string `json:"userId,omitempty"`
	OptOut         bool   `json:"optOut"`
	SessionID      int    `json:"sessionId"`
	LastEventTime  int    `json:"lastEventTime"`
	EventID        int    `json:"eventId"`
	IdentifyID     int    `json:"identifyId"`
	SequenceNumber int    `json:"sequenceNumber"`
}

func run() {
	log.SetFlags(0)

	var (
		err error
	)

	tp := http.DefaultTransport.(*http.Transport).Clone()

	if tp.TLSClientConfig.RootCAs, err = x509.SystemCertPool(); err != nil {
		log.Fatalf("ERROR: Problem adding system CA: %s", err)
	}

	cfg := elasticsearch.Config{
		Addresses: []string{os.Getenv("ELASTICSEARCH_URL")},
		Username:  os.Getenv("ELASTICSEARCH_USERNAME"),
		Password:  os.Getenv("ELASTICSEARCH_PASSWORD"),
		Transport: tp,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}

	// Search for the indexed documents
	//
	// Build the request body.
	query := strings.NewReader(`{
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
              "gte": "2020-12-18T01:26:52.298Z",
              "lte": "2020-12-25T01:26:52.299Z",
              "format": "strict_date_optional_time"
            }
          }
        }
      ],
      "should": [],
      "must_not": []
    }
  },
  "sort": ["_doc"]
}`)

	m, _ := time.ParseDuration("2m")
	// Perform the search request.
	page, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex("log-aws-waf-*"),
		es.Search.WithBody(query),
		es.Search.WithPretty(),
		es.Search.WithSize(1000),
		es.Search.WithScroll(m),
		es.Search.WithSearchType("dfs_query_then_fetch"),
	)
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer page.Body.Close()

	if page.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(page.Body).Decode(&e); err != nil {
			log.Fatalf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			log.Fatalf("[%s] %s: %s",
				page.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var b bytes.Buffer
	b.ReadFrom(page.Body)
	total := gjson.GetBytes(b.Bytes(), "hits.total.value").Int()
	scrollSize := total
	log.Printf("scroll size: %v", scrollSize)
	took := gjson.GetBytes(b.Bytes(), "took").Int()
	sid := gjson.GetBytes(b.Bytes(), "_scroll_id").String()
	log.Printf("sid: %v", sid)

	amplitudeID := AmplitudeID{}
	amplitudeIDs := make([]AmplitudeID, 0, scrollSize)

	for _, hit := range gjson.GetBytes(b.Bytes(), "hits.hits").Array() {
		for k, v := range hit.Map() {
			if k == "_source" {
				headers := gjson.Get(v.String(), "httpRequest.headers").Array()
				for _, header := range headers {
					if header.Map()["name"].Str == "cookie" {
						cookie := header.Map()["value"].Str
						for _, values := range strings.Split(cookie, ";") {
							if strings.Contains(values, "amplitude_id") {
								sEnc := trimNextEqual(values)
								sDec, err := b64.StdEncoding.DecodeString(sEnc)
								if err != nil {
									log.Printf("ERROR: %v", err)
								}
								err = json.Unmarshal(sDec, &amplitudeID)
								if err != nil {
									log.Printf("ERROR: %v", err)
								}
								amplitudeIDs = append(amplitudeIDs, amplitudeID)
							}
						}
					}
				}
			}
		}
	}

	for scrollSize > 0 {
		res, err := es.Scroll(
			es.Scroll.WithScrollID(sid),
			es.Scroll.WithScroll(m),
		)
		if err != nil {
			log.Fatalf("Error getting response: %s", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			var e map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
				log.Fatalf("Error parsing the response body: %s", err)
			} else {
				// Print the response status and error information.
				log.Fatalf("[%s] %s: %s",
					res.Status(),
					e["error"].(map[string]interface{})["type"],
					e["error"].(map[string]interface{})["reason"],
				)
			}
		}

		var buf bytes.Buffer
		buf.ReadFrom(res.Body)
		for _, hit := range gjson.GetBytes(buf.Bytes(), "hits.hits").Array() {
			for k, v := range hit.Map() {
				if k == "_source" {
					headers := gjson.Get(v.String(), "httpRequest.headers").Array()
					for _, header := range headers {
						if header.Map()["name"].Str == "cookie" {
							cookie := header.Map()["value"].Str
							for _, values := range strings.Split(cookie, ";") {
								if strings.Contains(values, "amplitude_id") {
									sEnc := trimNextEqual(values)
									sDec, err := b64.StdEncoding.DecodeString(sEnc)
									if err != nil {
										log.Printf("ERROR: %v", err)
									}
									err = json.Unmarshal(sDec, &amplitudeID)
									if err != nil {
										log.Printf("ERROR: %v", err)
									}
									amplitudeIDs = append(amplitudeIDs, amplitudeID)
								}
							}
						}
					}
				}
			}
		}
		scrollSize = int64(len(gjson.GetBytes(buf.Bytes(), "hits.hits").Array()))
		took += gjson.GetBytes(buf.Bytes(), "took").Int()
		log.Printf("scroll size: %v", scrollSize)
		log.Printf("amplitude Id: %v", len(amplitudeIDs))
	}
	out, err := json.Marshal(&amplitudeIDs)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}
	fmt.Println(string(out))

	log.Printf("amplitude Id count: %v", len(amplitudeIDs))
	log.Println(strings.Repeat("=", 37))
	log.Printf(
		"[%s] %d hits; took: %dms\n",
		page.Status(),
		total,
		took,
	)
	log.Println(strings.Repeat("=", 37))
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
		log.Printf("%v: %v:%v", i, userID.uuid, userID.count)
	}
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
