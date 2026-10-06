module.exports.parse = ({ content, name, url }, { yaml, axios, notify }) => {
  const fs = require('fs');
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
    return { ...content, rules: unique([...myRules, ...originalRules]) };
  }

  const guardRules = [
    'PROCESS-NAME,nju-connect.exe,DIRECT',
    'DOMAIN,vpn.nju.edu.cn,DIRECT'
  ];

  return {
    ...content,
    proxies: [
      ...originalProxies.filter((item) => item.name !== 'NJU-VPN'),
      proxy
    ],
    rules: unique([
      ...guardRules,
      ...schoolRules,
      ...myRules,
      ...originalRules.filter((rule) => !isNjuRule(rule))
    ])
  };
};
