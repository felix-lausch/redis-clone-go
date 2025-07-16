package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseResp(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantName    string
		wantArgsLen int
		wantErr     bool
	}{
		{
			name:        "PING command no args",
			input:       "*1\r\n$4\r\nPING\r\n",
			wantName:    "PING",
			wantArgsLen: 0,
			wantErr:     false,
		},
		{
			name:        "ECHO command with 1 arg",
			input:       "*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n",
			wantName:    "ECHO",
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name:        "SET command with 4 args",
			input:       "*5\r\n$3\r\nSET\r\n$6\r\nbanana\r\n$10\r\nstrawberry\r\n$2\r\nPX\r\n$4\r\n5000\r\n",
			wantName:    "SET",
			wantArgsLen: 4,
			wantErr:     false,
		},
		{
			name:        "GET command with 1 arg",
			input:       "*2\r\n$3\r\nGET\r\n$6\r\nbanana\r\n",
			wantName:    "GET",
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name:        "Invalid input - wrong type indicator",
			input:       ":1\r\n",
			wantName:    "",
			wantArgsLen: 0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			cmd, err := parseResp(reader)

			if (err != nil) != tt.wantErr {
				t.Fatalf("parseResp() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				// If error expected, no need to check further
				return
			}

			if cmd.Name != tt.wantName {
				t.Errorf("Expected command name '%s', got '%s'", tt.wantName, cmd.Name)
			}

			if len(cmd.Args) != tt.wantArgsLen {
				t.Errorf("Expected %d args, got %d", tt.wantArgsLen, len(cmd.Args))
			}
		})
	}
}
