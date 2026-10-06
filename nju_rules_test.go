package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mythologyli/zju-connect/client"
	"inet.af/netaddr"
)

func TestSchoolRulesCoverMultipleHostsWithoutExpandingPermissions(t *testing.T) {
	resources := []client.IPResource{
		{IPMin: net.ParseIP("10.10.0.0"), IPMax: net.ParseIP("10.10.255.255"), PortMin: 1, PortMax: 65535, Protocol: "all"},
		{IPMin: net.ParseIP("172.20.4.10"), IPMax: net.ParseIP("172.20.4.20"), PortMin: 22, PortMax: 22, Protocol: "tcp"},
		{IPMin: net.ParseIP("192.168.0.1"), IPMax: net.ParseIP("192.168.0.255"), PortMin: 53, PortMax: 53, Protocol: "udp"},
	}
	prefixes, err := schoolTCPPrefixes(resources)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		ip      string
		covered bool
	}{
		{"10.10.5.42", true}, {"10.10.8.99", true}, {"172.20.4.15", true},
		{"172.20.4.9", false}, {"172.20.4.21", false}, {"192.168.0.42", false}, {"10.11.1.2", false},
	} {
		covered := false
		for _, prefix := range prefixes {
			covered = covered || prefix.Contains(netaddr.MustParseIP(tt.ip))
		}
		if covered != tt.covered {
			t.Errorf("%s covered = %t, want %t", tt.ip, covered, tt.covered)
		}
	}
}

func TestSchoolRulesReplaceOldRangesAndHandleEmptyPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.yaml")
	resource := client.IPResource{IPMin: net.ParseIP("10.10.0.0"), IPMax: net.ParseIP("10.10.255.255"), Protocol: "tcp", PortMin: 1, PortMax: 65535}
	if count, err := exportClashIPRules(path, "127.0.0.1:12080", []client.IPResource{resource, resource}); err != nil || count != 1 {
		t.Fatalf("export count %d, error %v", count, err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "IP-CIDR,10.10.0.0/16,NJU-VPN,no-resolve") || !strings.Contains(string(data), "port: 12080") {
		t.Fatalf("missing granted subnet: %s", data)
	}
	scriptPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".cfw-mixin.js"
	script, err := os.ReadFile(scriptPath)
	if err != nil || !strings.Contains(string(script), "IP-CIDR,10.10.0.0/16,NJU-VPN,no-resolve") || strings.Contains(string(script), "NJU_FRAGMENT") {
		t.Fatalf("CFW Mixin was not generated correctly: %v", err)
	}
	if _, err := exportClashIPRules(path, "127.0.0.1:12080", nil); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "No school TCP IPv4 ranges") || strings.Contains(string(data), "10.10.") {
		t.Fatalf("old policy survived replacement: %s", data)
	}
}
