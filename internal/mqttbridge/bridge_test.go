package mqttbridge

import "testing"

func TestParseSwitchPayload(t *testing.T) {
	cases := map[string]bool{
		"ON":    true,
		"true":  true,
		"1":     true,
		"OFF":   false,
		"false": false,
		"0":     false,
	}
	for payload, want := range cases {
		got, err := parseSwitchPayload(payload)
		if err != nil {
			t.Fatalf("parseSwitchPayload(%q) error = %v", payload, err)
		}
		if got != want {
			t.Fatalf("parseSwitchPayload(%q) = %v, want %v", payload, got, want)
		}
	}
	if _, err := parseSwitchPayload("maybe"); err == nil {
		t.Fatal("parseSwitchPayload(maybe) expected error")
	}
}
