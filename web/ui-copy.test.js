const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const indexHTML = fs.readFileSync(path.join(__dirname, "index.html"), "utf8");
const appJS = fs.readFileSync(path.join(__dirname, "app.js"), "utf8");

test("index.html uses Chinese page metadata and controls", () => {
  assert.match(indexHTML, /<html lang="zh-CN">/);
  assert.match(indexHTML, /保留时长/);
  assert.match(indexHTML, /导出 JSON/);
  assert.match(indexHTML, /概览/);
});

test("app.js contains representative Chinese runtime copy", () => {
  assert.match(appJS, /当前 Token/);
  assert.match(appJS, /暂无容器/);
  assert.match(appJS, /已保存配置草稿。/);
  assert.match(appJS, /确定要立即轮换 Token 吗？/);
});
