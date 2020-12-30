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
