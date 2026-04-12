(function initChartUtils(root, factory) {
  const api = factory();
  if (typeof module === "object" && module.exports) {
    module.exports = api;
  }
  root.SrvdogChartUtils = api;
})(typeof globalThis !== "undefined" ? globalThis : this, function chartUtilsFactory() {
  const DEFAULT_LAYOUT = Object.freeze({
    left: 72,
    right: 16,
    top: 12,
    bottom: 16,
  });

  function buildPercentYAxis() {
    return {
      max: 100,
      ticks: [0, 25, 50, 75, 100].map((value) => ({
        value,
        label: String(value),
      })),
    };
  }

  function buildRateYAxis(maxValue) {
    const upper = computeNiceUpperBound(maxValue, 4);
    const step = upper / 4;
    const ticks = [];
    for (let index = 0; index <= 4; index += 1) {
      const value = step * index;
      ticks.push({
        value,
        label: `${formatRateValue(value, step)} KB/s`,
      });
    }
    return { max: upper, ticks };
  }

  function createChartLayout(width, height, overrides = {}) {
    const left = overrides.left ?? DEFAULT_LAYOUT.left;
    const right = overrides.right ?? DEFAULT_LAYOUT.right;
    const top = overrides.top ?? DEFAULT_LAYOUT.top;
    const bottom = overrides.bottom ?? DEFAULT_LAYOUT.bottom;

    return {
      left,
      right,
      top,
      bottom,
      plotWidth: Math.max(width - left - right, 1),
      plotHeight: Math.max(height - top - bottom, 1),
    };
  }

  function projectPoint({ index, count, value, max, layout }) {
    const safeCount = Math.max(count - 1, 1);
    const safeMax = max > 0 ? max : 1;
    const x = layout.left + (layout.plotWidth / safeCount) * index;
    const ratio = clamp(value / safeMax, 0, 1);
    const y = layout.top + (1 - ratio) * layout.plotHeight;
    return { x, y };
  }

  function computeNiceUpperBound(maxValue, tickCount) {
    if (!(maxValue > 0)) {
      return tickCount;
    }
    const rawStep = maxValue / tickCount;
    const step = computeNiceStep(rawStep);
    return step * tickCount;
  }

  function computeNiceStep(rawStep) {
    const magnitude = 10 ** Math.floor(Math.log10(rawStep));
    const normalized = rawStep / magnitude;
    if (normalized <= 1) return magnitude;
    if (normalized <= 2) return 2 * magnitude;
    if (normalized <= 5) return 5 * magnitude;
    return 10 * magnitude;
  }

  function formatRateValue(value, step) {
    if (step < 1) {
      return value.toFixed(1).replace(/\.0$/, "");
    }
    return String(Math.round(value));
  }

  function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
  }

  return {
    buildPercentYAxis,
    buildRateYAxis,
    createChartLayout,
    projectPoint,
  };
});
