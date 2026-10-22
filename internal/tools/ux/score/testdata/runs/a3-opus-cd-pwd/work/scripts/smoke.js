// Runs the rendered page's inline script against a tiny DOM stub.
const fs = require("fs");
const html = fs.readFileSync(process.argv[2], "utf8");
const src = html.match(/<script>([\s\S]*?)<\/script>/)[1];
const counts = {};
function node(tag) {
  counts[tag] = (counts[tag] || 0) + 1;
  return { tag, attrs: {}, children: [], style: {}, classList: { add() {}, remove() {} },
    setAttribute(k, v) { if (typeof v === "number" && !isFinite(v)) throw new Error(tag + " " + k + "=" + v); this.attrs[k] = v; },
    appendChild(c) { this.children.push(c); return c; }, append(...c) { this.children.push(...c); },
    insertBefore(c) { this.children.push(c); }, addEventListener() {},
    getBoundingClientRect() { return { width: 880 }; }, set textContent(v) { this.text = v; } };
}
const byId = {}; const q = {};
global.document = {
  createElementNS: (_, t) => node(t), createElement: node,
  getElementById: id => byId[id] || (byId[id] = node("#" + id)),
  querySelector: s => q[s] || (q[s] = Object.assign(node(s), { textContent: "T" })),
};
eval(src);
const paths = [];
(function walk(n) { if (n.tag === "path") paths.push(n.attrs.d); (n.children || []).forEach(walk); })(byId.plot);
console.log("summary:", byId.summary.text);
console.log("bars:", paths.length, "| NaN in paths:", paths.some(d => /NaN|Infinity/.test(d)));
