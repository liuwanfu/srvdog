const state = {
  history: [],
  realtime: [],
  summary: null,
  viewerId: crypto.randomUUID(),
};

const cardsEl = document.getElementById("cards");
const dockerTableBody = document.querySelector("#docker-table tbody");
const retentionEl = document.getElementById("retention");
const windowEl = document.getElementById("window");
const chartUtils = window.SrvdogChartUtils;

function bytes(value) {
  if (!value) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let size = value;
  let index = 0;
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024;
    index += 1;
  }
  return `${size.toFixed(size >= 10 ? 0 : 1)} ${units[index]}`;
}

function rate(value) {
  return `${bytes(value)}/s`;
}

function percent(used, total) {
  if (!total) return 0;
  return (used / total) * 100;
}

async function getJSON(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    throw new Error(await response.text());
  }
  return response.json();
}

function renderCards(summary) {
  const sample = summary.sample;
  const entries = [
    ["Mode", summary.mode.toUpperCase()],
    ["CPU", `${sample.cpu_percent.toFixed(1)}%`],
    ["Load", `${sample.load_1.toFixed(2)} / ${sample.load_5.toFixed(2)} / ${sample.load_15.toFixed(2)}`],
    ["Memory", `${bytes(sample.mem_used_bytes)} / ${bytes(sample.mem_total_bytes)}`],
    ["Swap", `${bytes(sample.swap_used_bytes)} / ${bytes(sample.swap_total_bytes)}`],
    ["Disk", `${bytes(sample.disk_used_bytes)} / ${bytes(sample.disk_total_bytes)}`],
    ["Network realtime", `↓ ${rate(sample.net_rx_bps)} · ↑ ${rate(sample.net_tx_bps)}`],
    ["Network avg", `↓ ${rate(sample.net_rx_avg_bps)} · ↑ ${rate(sample.net_tx_avg_bps)}`],
    ["Interface", sample.primary_interface || "n/a"],
    ["Updated", summary.updated_at ? new Date(summary.updated_at).toLocaleString() : "-"],
  ];

  cardsEl.innerHTML = entries.map(([label, value]) => `
    <article class="card">
      <span>${label}</span>
      <strong>${value}</strong>
    </article>
  `).join("");
}

function renderDocker(summary) {
  dockerTableBody.innerHTML = "";
  for (const container of summary.docker || []) {
    const row = document.createElement("tr");
    row.innerHTML = `
      <td>${container.name}</td>
      <td>${container.image}</td>
      <td>${container.status}</td>
      <td>${container.health || "-"}</td>
    `;
    dockerTableBody.appendChild(row);
  }
  if ((summary.docker || []).length === 0) {
    const row = document.createElement("tr");
    row.innerHTML = `<td colspan="4">${summary.docker_error || "No containers"}</td>`;
    dockerTableBody.appendChild(row);
  }
}

function mergeSeries() {
  return [...state.history, ...state.realtime]
    .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
}

function drawLine(canvasId, values, colors) {
  const canvas = document.getElementById(canvasId);
  const ctx = canvas.getContext("2d");
  const width = canvas.width;
  const height = canvas.height;
  const layout = chartUtils.createChartLayout(width, height);
  const axis = resolveAxis(values, canvasId);

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = "#081019";
  ctx.fillRect(0, 0, width, height);

  drawYAxis(ctx, width, height, layout, axis);

  values.forEach((series, index) => {
    if (series.length < 2) {
      return;
    }
    ctx.strokeStyle = colors[index];
    ctx.lineWidth = 2;
    ctx.beginPath();
    series.forEach((item, i) => {
      const point = chartUtils.projectPoint({
        index: i,
        count: series.length,
        value: item.value,
        max: axis.max,
        layout,
      });
      if (i === 0) {
        ctx.moveTo(point.x, point.y);
      } else {
        ctx.lineTo(point.x, point.y);
      }
    });
    ctx.stroke();
  });
}

function resolveAxis(values, canvasId) {
  if (canvasId === "network-chart") {
    const max = Math.max(
      0,
      ...values.flatMap((series) => series.map((item) => item.value)),
    );
    return chartUtils.buildRateYAxis(max);
  }
  return chartUtils.buildPercentYAxis();
}

function drawYAxis(ctx, width, height, layout, axis) {
  ctx.save();
  ctx.font = '11px "Segoe UI", "PingFang SC", sans-serif';
  ctx.textAlign = "right";
  ctx.textBaseline = "middle";
  ctx.fillStyle = "rgba(231, 238, 247, 0.72)";
  ctx.strokeStyle = "rgba(255,255,255,0.12)";
  ctx.lineWidth = 1;

  axis.ticks.forEach((tick) => {
    const point = chartUtils.projectPoint({
      index: 0,
      count: 1,
      value: tick.value,
      max: axis.max,
      layout,
    });
    ctx.beginPath();
    ctx.moveTo(layout.left, point.y);
    ctx.lineTo(width - layout.right, point.y);
    ctx.stroke();
    ctx.fillText(tick.label, layout.left - 8, point.y);
  });

  ctx.restore();
}

function renderCharts() {
  const merged = mergeSeries();
  const cpu = merged.map((item) => ({ value: item.cpu_percent }));
  const memory = merged.map((item) => ({ value: percent(item.mem_used_bytes, item.mem_total_bytes) }));
  const disk = merged.map((item) => ({ value: percent(item.disk_used_bytes, item.disk_total_bytes) }));
  const netRx = merged.map((item) => ({ value: item.net_rx_bps / 1024 }));
  const netTx = merged.map((item) => ({ value: item.net_tx_bps / 1024 }));

  drawLine("cpu-chart", [cpu], ["#4dd0e1"]);
  drawLine("memory-chart", [memory], ["#7cb342"]);
  drawLine("disk-chart", [disk], ["#ffb300"]);
  drawLine("network-chart", [netRx, netTx], ["#29b6f6", "#ef5350"]);
}

async function refreshSummary() {
  state.summary = await getJSON("/api/summary");
  retentionEl.value = String(state.summary.retention_days);
  renderCards(state.summary);
  renderDocker(state.summary);
}

async function refreshHistory() {
  const payload = await getJSON(`/api/history?window=${encodeURIComponent(windowEl.value)}`);
  state.history = payload.samples || [];
}

async function refreshRealtime() {
  const payload = await getJSON("/api/realtime");
  state.realtime = payload.samples || [];
  renderCharts();
}

async function sendHeartbeat() {
  await fetch("/api/heartbeat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id: state.viewerId }),
  });
}

async function setRetention(days) {
  await fetch("/api/settings/retention", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ days }),
  });
  await refreshSummary();
}

function exportData(format) {
  const url = `/api/export?format=${encodeURIComponent(format)}&window=${encodeURIComponent(windowEl.value)}`;
  window.location.href = url;
}

async function clearHistory() {
  if (!window.confirm("Clear all stored sampling history?")) {
    return;
  }
  await fetch("/api/history/clear", { method: "POST" });
  state.history = [];
  state.realtime = [];
  renderCharts();
}

retentionEl.addEventListener("change", (event) => {
  setRetention(Number(event.target.value)).catch(alert);
});

windowEl.addEventListener("change", () => {
  refreshHistory().then(renderCharts).catch(alert);
});

document.getElementById("export-json").addEventListener("click", () => exportData("json"));
document.getElementById("export-csv").addEventListener("click", () => exportData("csv"));
document.getElementById("clear-history").addEventListener("click", () => clearHistory().catch(alert));

async function boot() {
  await Promise.all([refreshSummary(), refreshHistory()]);
  await sendHeartbeat();
  await refreshRealtime();
  setInterval(() => sendHeartbeat().catch(console.error), 10000);
  setInterval(() => refreshSummary().catch(console.error), 5000);
  setInterval(() => refreshRealtime().catch(console.error), 2000);
}

boot().catch((error) => {
  console.error(error);
  alert(error.message);
});
