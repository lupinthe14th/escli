package main

import (
	"fmt"
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
