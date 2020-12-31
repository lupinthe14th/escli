package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
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
	tests := []struct {
		in      string
		want    io.Reader
		wantErr bool
	}{
		{in: "", want: strings.NewReader(query), wantErr: false},
		{in: "err", want: nil, wantErr: true},
		{in: filename, want: b, wantErr: false},
	}
	for i, tt := range tests {
		i, tt := i, tt
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			got, err := buildQuery(tt.in)
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
