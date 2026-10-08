const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');
const EventEmitter = require('node:events');

function load(fragment, fail = false, available = true) {
  const notices = [];
  const context = {
    module: { exports: {} },
    Buffer,
    require(name) {
      if (name === 'net') return { createConnection() {
        const socket = new EventEmitter();
        socket.setTimeout = () => socket;
        socket.destroy = () => {};
        socket.write = (data) => {
          assert.equal(data.toString('hex'), '050100');
          queueMicrotask(() => socket.emit('data', Buffer.from([5, 0])));
        };
        queueMicrotask(() => socket.emit(available ? 'connect' : 'error', new Error('offline')));
        return socket;
      } };
      assert.equal(name, 'fs');
      return { readFileSync(file, encoding) {
        assert.equal(file, 'C:/path/to/NJU-Connect/clash-nju.generated.yaml');
        assert.equal(encoding, 'utf8');
        if (fail) throw new Error('missing file');
        return JSON.stringify(fragment);
      } };
    },
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '..', 'CFW-Mixin-NJU.js'), 'utf8'), context);
  return { apply: (content) => context.module.exports.parse({ content }, {
    yaml: { parse: JSON.parse }, notify: (...args) => notices.push(args),
  }), notices };
}

test('file Mixin merges user rules and multiple school ranges without modifying networking', async () => {
  const school = ['IP-CIDR,10.49.0.0/16,NJU-VPN,no-resolve', 'IP-CIDR,172.20.0.0/16,NJU-VPN,no-resolve'];
  const { apply } = load({ proxies: [{ name: 'NJU-VPN', server: '127.0.0.1', port: 12080 }], rules: school });
  const content = {
    tun: { enable: true, 'auto-route': true }, dns: { enable: true }, 'mixed-port': 7890,
    rules: ['IP-CIDR,10.0.0.1/32,NJU-VPN,no-resolve', 'DOMAIN-SUFFIX,custom.example,DIRECT', 'MATCH,Existing'],
    proxies: [{ name: 'Existing' }, { name: 'NJU-VPN', port: 11080 }],
  };
  const before = JSON.stringify(content);
  const result = await apply(content);
  assert.equal(JSON.stringify(content), before);
  assert.equal(result.tun, content.tun);
  assert.equal(result.dns, content.dns);
  assert.equal(result['mixed-port'], 7890);
  assert.equal(result.rules[2], school[0]);
  assert.equal(result.rules[3], school[1]);
  assert.equal(result.rules.includes('IP-CIDR,10.0.0.1/32,NJU-VPN,no-resolve'), false);
  assert.equal(result.rules.includes('DOMAIN-SUFFIX,custom.example,DIRECT'), true);
  assert.equal(result.rules.at(-1), 'MATCH,Existing');
  assert.equal(result.proxies.find((item) => item.name === 'NJU-VPN').port, 12080);
  assert.equal(JSON.stringify(await apply(result)), JSON.stringify(result));
});

test('missing or unpopulated rule file retains original profile and user rules', async () => {
  for (const fail of [true, false]) {
    const { apply, notices } = load({ proxies: [], rules: [] }, fail);
    const content = { proxies: [{ name: 'Existing' }], tun: { enable: true }, rules: ['MATCH,Existing'] };
    const result = await apply(content);
    assert.equal(JSON.stringify(result.proxies), JSON.stringify(content.proxies));
    assert.equal(result.tun, content.tun);
    assert.equal(result.rules.at(-1), 'MATCH,Existing');
    assert.equal(notices.length, 1);
  }
});

test('offline VPN removes its node and rules, directs school private IPs, and preserves public routing', async () => {
  const school = ['IP-CIDR,10.0.0.0/10,NJU-VPN,no-resolve', 'IP-CIDR,203.0.113.42/32,NJU-VPN,no-resolve'];
  const { apply } = load({ proxies: [{ name: 'NJU-VPN', server: '127.0.0.1', port: 11080 }], rules: school }, false, false);
  const content = {
    tun: { enable: true }, dns: { enable: true }, 'mixed-port': 7890,
    proxies: [{ name: 'Existing' }, { name: 'NJU-VPN' }],
    rules: [...school, 'MATCH,Existing'],
  };
  const result = await apply(content);
  assert.equal(result.proxies.some((p) => p.name === 'NJU-VPN'), false);
  assert.equal(result.rules.some((r) => r.includes(',NJU-VPN')), false);
  assert.equal(result.rules[0], 'IP-CIDR,10.0.0.0/10,DIRECT,no-resolve');
  assert.equal(result.rules.some((r) => r.includes('203.0.113.42')), false);
  assert.equal(result.rules.at(-1), 'MATCH,Existing');
  assert.equal(result.tun, content.tun);
  assert.equal(result.dns, content.dns);
  assert.equal(result['mixed-port'], 7890);
});
