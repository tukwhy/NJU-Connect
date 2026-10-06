const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const test = require("node:test");

test("NJU Mixin preserves network settings and existing routes, replaces only NJU entries", () => {
  const nju = {
    proxies: [{ name: "NJU-VPN", type: "socks5", server: "127.0.0.1", port: 11080, udp: false }],
    rules: ["PROCESS-NAME,nju-connect.exe,DIRECT", "DOMAIN,vpn.nju.edu.cn,DIRECT", "IP-CIDR,10.49.0.0/16,NJU-VPN,no-resolve"],
  };
  const template = fs.readFileSync(path.join(__dirname, "..", "CFW-Mixin.template.js"), "utf8");
  const context = { module: { exports: {} } };
  vm.runInNewContext(template.replace("/* NJU_FRAGMENT */ {}", JSON.stringify(nju)), context);
  const original = {
    proxies: [{ name: "Existing", type: "ss" }, { name: "NJU-VPN", port: 9999 }],
    rules: ["IP-CIDR,10.0.0.42/32,NJU-VPN,no-resolve", "IP-CIDR,192.168.0.0/16,DIRECT", "MATCH,Existing"],
    tun: { enable: true, stack: "gvisor", "auto-route": true },
    dns: { enable: true, "enhanced-mode": "fake-ip" },
    "mixed-port": 7890,
    mode: "rule",
  };
  const before = JSON.stringify(original);
  const apply = context.module.exports.parse;
  const output = apply({ content: original });
  assert.equal(JSON.stringify(original), before, "original profile mutated");
  assert.equal(output.tun, original.tun);
  assert.equal(output.dns, original.dns);
  assert.equal(output["mixed-port"], 7890);
  assert.equal(output.mode, "rule");
  assert.equal(output.proxies.length, 2);
  assert.equal(output.proxies.find((p) => p.name === "NJU-VPN").port, 11080);
  assert.equal(JSON.stringify(output.rules), JSON.stringify([...nju.rules, original.rules[1], original.rules[2]]));
  assert.equal(JSON.stringify(apply({ content: output })), JSON.stringify(output), "repeated Mixin duplicated entries");
});
