const test = require("node:test");
const assert = require("node:assert/strict");

const {
  buildPercentYAxis,
  buildRateYAxis,
  createChartLayout,
  projectPoint,
} = require("./chart-utils.js");

test("buildPercentYAxis returns fixed 0-100 ticks", () => {
  const axis = buildPercentYAxis();
  assert.equal(axis.max, 100);
  assert.deepEqual(axis.ticks.map((tick) => tick.label), ["0", "25", "50", "75", "100"]);
});

test("buildRateYAxis rounds to readable KB/s ticks", () => {
  const axis = buildRateYAxis(137);
  assert.equal(axis.max, 200);
  assert.deepEqual(axis.ticks.map((tick) => tick.label), [
    "0 KB/s",
    "50 KB/s",
    "100 KB/s",
    "150 KB/s",
    "200 KB/s",
  ]);
});

test("projectPoint stays inside the computed plot area", () => {
  const layout = createChartLayout(600, 180);
  const first = projectPoint({ index: 0, count: 2, value: 100, max: 100, layout });
  const last = projectPoint({ index: 1, count: 2, value: 0, max: 100, layout });

  assert.equal(first.x, layout.left);
  assert.equal(first.y, layout.top);
  assert.equal(last.x, 600 - layout.right);
  assert.equal(last.y, 180 - layout.bottom);
});
