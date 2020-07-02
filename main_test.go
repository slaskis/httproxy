package main

import (
	"fmt"
	"testing"
)

func TestParseArg(t *testing.T) {
	tests := []struct {
		Arg string
		Cfg config
		Err error
	}{
		{"", config{}, fmt.Errorf("invalid argument: ")},
		{"==", config{}, fmt.Errorf("invalid argument: ==")},
		{"=", config{Scheme: "https"}, nil},
		{"/a=/b", config{SrcPath: "/a", DstPath: "/b", Scheme: "https"}, nil},
		{"/a=localhost/", config{SrcPath: "/a", DstPath: "/", Host: "localhost", Scheme: "https"}, nil},
		{"/a=localhost:4000/", config{SrcPath: "/a", DstPath: "/", Host: "localhost:4000", Scheme: "https"}, nil},
		{"/a=http://localhost:4000/", config{SrcPath: "/a", DstPath: "/", Host: "localhost:4000", Scheme: "http"}, nil},
		{"/a=http://localhost:4000/b", config{SrcPath: "/a", DstPath: "/b", Host: "localhost:4000", Scheme: "http"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.Arg, func(t *testing.T) {
			cfg, err := parseArg(tt.Arg, false, false)
			if (err == nil) != (tt.Err == nil) {
				if err == nil || err.Error() == tt.Err.Error() {
					t.Error("unexpected error", err, tt.Err)
				}
			}
			if cfg != tt.Cfg {
				t.Errorf("%#v != %# v", cfg, tt.Cfg)
			}
		})
	}
}
