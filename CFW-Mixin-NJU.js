module.exports.parse = async ({ content, name, url }, { yaml, axios, notify }) => {
  const fs = require('fs');
  const net = require('net');
  // Change this one path to the generated YAML in your unpacked project.
  const file = 'C:/path/to/NJU-Connect/clash-nju.generated.yaml';

  // Optional custom rules. Existing subscription rules are retained below.
  const myRules = [];

  const originalRules = Array.isArray(content.rules) ? content.rules : [];
  const originalProxies = Array.isArray(content.proxies) ? content.proxies : [];
  const unique = (rules) => [...new Set(rules)];
  const isNjuRule = (rule) => {
    if (typeof rule !== 'string') return false;
    const fields = rule.split(',').map((value) => value.trim());
    return ['IP-CIDR', 'IP-CIDR6'].includes(fields[0]) && fields[2] === 'NJU-VPN';
  };
  const cleanProxies = originalProxies.filter((item) => item.name !== 'NJU-VPN');
  const cleanRules = originalRules.filter((rule) => !isNjuRule(rule));
  const withoutVpn = (directRules = []) => ({
    ...content,
    proxies: cleanProxies,
    rules: unique([...directRules, ...myRules, ...cleanRules])
  });
  const privateDirectRule = (rule) => {
    const fields = rule.split(',').map((value) => value.trim());
    if (fields[0] !== 'IP-CIDR') return null;
    const [address, bitsText] = fields[1].split('/');
    const octets = address.split('.').map(Number);
    const bits = Number(bitsText);
    if (octets.length !== 4 || octets.some((value) => !Number.isInteger(value) || value < 0 || value > 255)
        || !Number.isInteger(bits) || bits < 0 || bits > 32) return null;
    const ip = octets.reduce((value, octet) => (value * 256 + octet) >>> 0, 0);
    const mask = bits === 0 ? 0 : (0xffffffff << (32 - bits)) >>> 0;
    const first = (ip & mask) >>> 0;
    const last = (first | ~mask) >>> 0;
    const privateRanges = [[0x0a000000, 0x0affffff], [0xac100000, 0xac1fffff], [0xc0a80000, 0xc0a8ffff]];
    if (!privateRanges.some(([start, end]) => first >= start && last <= end)) return null;
    return `IP-CIDR,${fields[1]},DIRECT,no-resolve`;
  };
  const isVpnAvailable = (proxy) => new Promise((resolve) => {
    if (!['127.0.0.1', '::1', 'localhost'].includes(proxy.server)) return resolve(false);
    const socket = net.createConnection({ host: proxy.server, port: Number(proxy.port) });
    let settled = false;
    let received = Buffer.alloc(0);
    const finish = (available) => {
      if (settled) return;
      settled = true;
      socket.destroy();
      resolve(available);
    };
    socket.setTimeout(800);
    socket.once('error', () => finish(false));
    socket.once('timeout', () => finish(false));
    socket.once('end', () => finish(false));
    socket.once('connect', () => socket.write(Buffer.from([5, 1, 0])));
    socket.on('data', (data) => {
      received = Buffer.concat([received, data]);
      if (received.length >= 2) finish(received[0] === 5 && received[1] === 0);
    });
  });

  let proxy;
  let schoolRules;
  try {
    const fragment = yaml.parse(fs.readFileSync(file, 'utf8'));
    proxy = (fragment.proxies || []).find((item) => item.name === 'NJU-VPN');
    schoolRules = (fragment.rules || []).filter(isNjuRule);
    if (!proxy || !schoolRules.length) {
      throw new Error('尚未生成校内网段，请先运行 Start-NJU.ps1 完成学校登录');
    }
  } catch (error) {
    if (typeof notify === 'function') {
      notify('NJU 分流未加载', String(error.message || error), false);
    }
    return withoutVpn();
  }

  if (!(await isVpnAvailable(proxy))) {
    return withoutVpn(schoolRules.map(privateDirectRule).filter(Boolean));
  }

  const guardRules = [
    'PROCESS-NAME,nju-connect.exe,DIRECT',
    'DOMAIN,vpn.nju.edu.cn,DIRECT'
  ];

  return {
    ...content,
    proxies: [
      ...cleanProxies,
      proxy
    ],
    rules: unique([
      ...guardRules,
      ...schoolRules,
      ...myRules,
      ...cleanRules
    ])
  };
};
