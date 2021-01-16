package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestTrimNextEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{in: "aaaaaaaa=bbbbbbbbbbb", want: "bbbbbbbbbbb"},
		{in: "aaaaaaaa=bbbbbbbbbbb==", want: "bbbbbbbbbbb=="},
	}
	for i, tt := range tests {
		i, tt := i, tt
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			got := trimNextEqual(tt.in)
			if got != tt.want {
				t.Fatalf("in: %v got: %v want: %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildQuery(t *testing.T) {
	t.Parallel()
	const filename = "./testdata/search.json"
	f, _ := ioutil.ReadFile(filename)
	b := bytes.NewReader(f)
	since := "2020-12-23 13:04:05"
	until := "2020-12-23 14:15:16"
	type in struct {
		filename, rule, since, until string
	}
	tests := []struct {
		in      in
		want    io.Reader
		wantErr bool
	}{
		{in: in{filename: "", since: since, until: until}, want: strings.NewReader(fmt.Sprintf(MatchAllQuery, "2020-12-23T13:04:05Z", "2020-12-23T14:15:16Z")), wantErr: false},
		{in: in{filename: "err", since: since, until: until}, want: nil, wantErr: true},
		{in: in{filename: filename, since: since, until: until}, want: b, wantErr: false},
		{in: in{rule: "AmazonIpReputation", since: since, until: until}, want: strings.NewReader(fmt.Sprintf(AmazonIPReputationQuery, "2020-12-23T13:04:05Z", "2020-12-23T14:15:16Z")), wantErr: false},
		{in: in{rule: "AnonymousIP", since: since, until: until}, want: strings.NewReader(fmt.Sprintf(AnonymousIPQuery, "2020-12-23T13:04:05Z", "2020-12-23T14:15:16Z")), wantErr: false},
	}
	for i, tt := range tests {
		i, tt := i, tt
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			flags := []cli.Flag{
				&cli.StringFlag{Name: "query", Aliases: []string{"q"}},
				&cli.StringFlag{Name: "rule", Aliases: []string{"r"}},
				&cli.TimestampFlag{Name: "since", Aliases: []string{"s"}, Layout: "2006-01-02 15:04:05"},
				&cli.TimestampFlag{Name: "until", Aliases: []string{"u"}, Layout: "2006-01-02 15:04:05"},
			}
			set := flag.NewFlagSet("test", 0)
			for _, fl := range flags {
				_ = fl.Apply(set)
			}
			set.Parse([]string{"--query", tt.in.filename, "--rule", tt.in.rule, "--since", tt.in.since, "--until", tt.in.until})
			c := cli.NewContext(nil, set, nil)
			got, err := buildQuery(c)
			if (err != nil) != tt.wantErr {
				t.Fatalf("in: %v err: %v wantErr: %v", tt.in, err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("in: %v got: %v want: %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCookieToAmplitudeID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		want    AmplitudeID
		wantErr bool
	}{
		{in: "", want: AmplitudeID{}, wantErr: false},
		{in: string(`{"name":"Host","value":"www.example.com"},{"name":"user-agent","value":"curl"},{"name":"accept","value":"image/png,image/svg+xml"},{"name":"accept-language","value":"ja-jp"},{"name":"accept-encoding","value":"gzip,deflate"}`), want: AmplitudeID{}, wantErr: false},
		{in: string(`{"name":"Host","value":"www.example.com"},{"name":"user-agent","value":"curl"},{"name":"cookie","value":"amplitude_id_897A0F9D786941B78426F71846B088F0example.jp=eyJkZXZpY2VJZCI6ImMyY2MxOWNiLTQzOTAtNDgyOS05NzIzLWE4NGYzMmUwYTUxMyIsInVzZXJJZCI6ImIzZDc2NzA4LWNhZDktNDAyZS1iYjI5LTgxN2MxNDBjZWQ2ZiIsIm9wdE91dCI6ZmFsc2UsInNlc3Npb25JZCI6MTIzNDU2Nzg5MDEyMywibGFzdEV2ZW50VGltZSI6MTIzNDU2Nzg5MDEyMywiZXZlbnRJZCI6MSwiaWRlbnRpZnlJZCI6MSwic2VxdWVuY2VOdW1iZXIiOjF9Cg==; _ga=GA1.2.123456789.1234567890; _gid=GA1.2.123456789.1234567890;"}`), want: AmplitudeID{DeviceID: "c2cc19cb-4390-4829-9723-a84f32e0a513", UserID: "b3d76708-cad9-402e-bb29-817c140ced6f", OptOut: false, SessionID: 1234567890123, LastEventTime: 1234567890123, EventID: 1, IdentifyID: 1, SequenceNumber: 1}, wantErr: false},
		{in: string(`{"name":"Host","value":"www.example.com"},{"name":"user-agent","value":"curl"},{"name":"cookie","value":"amplitude_id_897A0F9D786941B78426F71846B088F0example.jp=eyJkZXZpY2VJZCI6ImMyY2MxOWNiLTQzOTAtNDgyOS05NzIzLWE4NGYzMmUwYTUxMyIsInVzZXJJZCI6ImIzZDc2NzA4LWNhZDktNDAyZS1iYjI5LTgxN2MxNDBjZWQ2ZiIsIm9wdE91dCI6ZmFsc2UsInNlc3Npb25JZCI6MTIzNDU2Nzg5MDEyMywibGFzdEV2ZW50VGltZSI6MTIzNDU2Nzg5MDEyMywiZXZlbnRJZCI6MSwiaWRlbnRpZnlJZCI6MSwic2VxdWVuY2VOdW1iZXIiOjF9Cg; _ga=GA1.2.123456789.1234567890; _gid=GA1.2.123456789.1234567890;"}`), want: AmplitudeID{}, wantErr: true},
		{in: string(`{"name":"Host","value":"www.example.com"},{"name":"user-agent","value":"curl"},{"name":"cookie","value":"amplitude_id_897A0F9D786941B78426F71846B088F0example.jp=eyJkZXZpY2VJZCI6ZmFsc2UsInVzZXJJZCI6ImIzZDc2NzA4LWNhZDktNDAyZS1iYjI5LTgxN2MxNDBjZWQ2ZiIsIm9wdE91dCI6ZmFsc2UsInNlc3Npb25JZCI6MTIzNDU2Nzg5MDEyMywibGFzdEV2ZW50VGltZSI6MTIzNDU2Nzg5MDEyMywiZXZlbnRJZCI6MSwiaWRlbnRpZnlJZCI6MSwic2VxdWVuY2VOdW1iZXIiOjF9Cg==; _ga=GA1.2.123456789.1234567890; _gid=GA1.2.123456789.1234567890;"}`), want: AmplitudeID{}, wantErr: true},
	}
	for i, tt := range tests {
		i, tt := i, tt
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			got, err := cookieToAmplitudeID(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("in: %v err: %v wantErr: %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("in: %v got: %v want: %v", tt.in, got, tt.want)
			}
		})
	}
}
