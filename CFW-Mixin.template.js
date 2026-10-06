// CFW 0.20.39 JavaScript Mixin. Generated data will be inserted on VPN login.
// If you already have a Mixin, keep its logic, add this helper, and pass its
// existing return value through addNJU(...). Do not replace your TUN/DNS setup.
function addNJU(content) {
  const nju = /* NJU_FRAGMENT */ {};
  const proxies = Array.isArray(content.proxies) ? content.proxies : [];
  const rules = Array.isArray(content.rules) ? content.rules : [];
  const extraRules = nju.rules || [];
  const extraRuleSet = new Set(extraRules);
  const isOldNJU = (rule) => {
    if (typeof rule !== "string") return false;
    const fields = rule.split(",").map((field) => field.trim());
    return (fields[0] === "IP-CIDR" || fields[0] === "IP-CIDR6") && fields[2] === "NJU-VPN";
  };
  return {
    ...content,
    proxies: [...proxies.filter((proxy) => proxy.name !== "NJU-VPN"), ...(nju.proxies || [])],
    rules: [...extraRules, ...rules.filter((rule) => !extraRuleSet.has(rule) && !isOldNJU(rule))],
  };
}

module.exports.parse = ({ content }) => addNJU(content);
