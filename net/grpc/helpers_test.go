package grpc

import (
	"testing"
)

func Test_targetParser(t *testing.T) {
	var tests = []struct {
		input  string
		result string
	}{
		{
			input:  "",
			result: ":",
		},
		{
			input:  "127.0.0.1",
			result: "127.0.0.1:" + defaultPort,
		},
		{
			input:  "registry://127.0.0.1",
			result: "127.0.0.1:" + defaultPort,
		},
		{
			input:  "127.0.0.1:255",
			result: "127.0.0.1:255",
		},
		{
			input:  "registry://127.0.0.1:255",
			result: "127.0.0.1:255",
		},
		{
			input:  "hostname",
			result: "hostname:" + defaultPort,
		},
		{
			input:  "registry://hostname",
			result: "hostname:" + defaultPort,
		},
		{
			input:  "hostname:255",
			result: "hostname:255",
		},
		{
			input:  "registry://hostname:255",
			result: "hostname:255",
		},
		{
			input:  "registry://%",
			result: ":",
		},
	}

	for _, test := range tests {
		if host, port, _ := parseTarget(test.input, ""); host+":"+port != test.result {
			t.Errorf("invalid parsing `%s` -> `%s`, in result have to be `%s`", test.input, host+":"+port, test.result)
		}
	}
}

func Test_formatIP(t *testing.T) {
	var tests = [][2]string{
		{"127.0.0.1", "127.0.0.1"},
		{"::1", "[::1]"},
		{"2001:db8::", "[2001:db8::]"},
		{"277.277.277.277", ""},
		{"", ""},
	}

	for _, test := range tests {
		if ip, _ := formatIP(test[0]); ip != test[1] {
			t.Errorf("invalid formation of IP `%s`, in result have to be `%s`", test[0], test[1])
		}
	}
}
