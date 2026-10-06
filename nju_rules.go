package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mythologyli/zju-connect/client"
	"inet.af/netaddr"
)

//go:embed CFW-Mixin.template.js
var cfwMixinTemplate string

// Export every granted TCP IPv4 range, not just a fixed SSH destination.
// Ports remain enforced by the VPN dialer using the original server policy.
func schoolTCPPrefixes(resources []client.IPResource) ([]netaddr.IPPrefix, error) {
	var builder netaddr.IPSetBuilder
	for _, resource := range resources {
		if resource.Protocol != "tcp" && resource.Protocol != "all" {
			continue
		}
		if resource.PortMin < 1 || resource.PortMax > 65535 || resource.PortMin > resource.PortMax {
			return nil, fmt.Errorf("invalid school resource port range")
		}
		start, end := resource.IPMin.To4(), resource.IPMax.To4()
		if start == nil || end == nil {
			// The current aTrust resource parser and TCP dialer are IPv4-only.
			continue
		}
		builder.AddRange(netaddr.IPRangeFrom(netaddr.MustParseIP(net.IP(start).String()), netaddr.MustParseIP(net.IP(end).String())))
	}
	set, err := builder.IPSet()
	if err != nil {
		return nil, fmt.Errorf("invalid school resource IP range: %w", err)
	}
	return set.Prefixes(), nil
}

func exportClashIPRules(path, socksBind string, resources []client.IPResource) (int, error) {
	host, port, err := net.SplitHostPort(socksBind)
	if err != nil {
		return 0, fmt.Errorf("cannot export SOCKS node: %w", err)
	}
	if err := validateLoopbackListener(socksBind); err != nil {
		return 0, err
	}
	prefixes, err := schoolTCPPrefixes(resources)
	if err != nil {
		return 0, err
	}
	var output strings.Builder
	output.WriteString("# Merge these entries into the existing config; this is not a full config.\n")
	output.WriteString("# Generated from all granted school TCP IPv4 resources. No school credentials.\n")
	fmt.Fprintf(&output, "proxies:\n  - name: NJU-VPN\n    type: socks5\n    server: %q\n    port: %s\n    udp: false\n\n", host, port)
	output.WriteString("# Insert these rules BEFORE existing LAN DIRECT / private-IP rule sets / MATCH.\n")
	output.WriteString("rules:\n  - PROCESS-NAME,nju-connect.exe,DIRECT\n  - DOMAIN,vpn.nju.edu.cn,DIRECT\n")
	for _, prefix := range prefixes {
		fmt.Fprintf(&output, "  - IP-CIDR,%s,NJU-VPN,no-resolve\n", prefix)
	}
	if len(prefixes) == 0 {
		output.WriteString("  # No school TCP IPv4 ranges were supplied. No campus destinations are matched.\n")
	}
	// Replace atomically so Clash never reads a partially written rule set.
	file, err := os.CreateTemp(filepath.Dir(path), ".nju-rules-*.tmp")
	if err != nil {
		return 0, err
	}
	temp := file.Name()
	defer os.Remove(temp)
	if _, err := file.WriteString(output.String()); err != nil {
		_ = file.Close()
		return 0, err
	}
	if err := file.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(temp, path); err != nil {
		return 0, err
	}
	// CFW's JS Mixin adds to the existing lists instead of replacing the
	// complete profile. It intentionally exports no network settings.
	portNumber, _ := strconv.Atoi(port) // Checked by validateLoopbackListener.
	fragment := struct {
		Proxies []map[string]any `json:"proxies"`
		Rules   []string         `json:"rules"`
	}{
		Proxies: []map[string]any{{"name": "NJU-VPN", "type": "socks5", "server": host, "port": portNumber, "udp": false}},
		Rules:   []string{"PROCESS-NAME,nju-connect.exe,DIRECT", "DOMAIN,vpn.nju.edu.cn,DIRECT"},
	}
	for _, prefix := range prefixes {
		fragment.Rules = append(fragment.Rules, fmt.Sprintf("IP-CIDR,%s,NJU-VPN,no-resolve", prefix))
	}
	jsonFragment, err := json.MarshalIndent(fragment, "  ", "  ")
	if err != nil {
		return 0, err
	}
	script := strings.Replace(cfwMixinTemplate, "/* NJU_FRAGMENT */ {}", string(jsonFragment), 1)
	scriptPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".cfw-mixin.js"
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return 0, fmt.Errorf("YAML exported, but CFW Mixin export failed: %w", err)
	}
	return len(prefixes), nil
}
