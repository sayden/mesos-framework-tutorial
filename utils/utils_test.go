package utils

import "testing"

func TestParseIp(t *testing.T) {
	res := ParseIP("127.0.0.1")
	t.Log(res)
}
