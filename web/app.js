const state = {
  history: [],
  realtime: [],
  summary: null,
  viewerId: crypto.randomUUID(),
  activeTab: "overview",
  clash: {
    status: null,
    logs: null,
  },
};

const cardsEl = document.getElementById("cards");
const dockerTableBody = document.querySelector("#docker-table tbody");
const retentionEl = document.getElementById("retention");
const windowEl = document.getElementById("window");
const overviewView = document.getElementById("overview-view");
const clashView = document.getElementById("clash-view");
const clashNoticeEl = document.getElementById("clash-notice");
const clashStatusEl = document.getElementById("clash-status");
const clashConfigEditorEl = document.getElementById("clash-config-editor");
const clashScriptEditorEl = document.getElementById("clash-script-editor");
const clashConfigSourceEl = document.getElementById("clash-config-source");
const clashScriptSourceEl = document.getElementById("clash-script-source");
const clashOperationsLogEl = document.getElementById("clash-operations-log");
const clashGeodataLogEl = document.getElementById("clash-geodata-log");
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

function formatMode(mode) {
  if (mode === "high") return "高频";
  if (mode === "low") return "低频";
  return mode;
}

function formatDocumentSource(source) {
  if (source === "draft") return "草稿";
  if (source === "published" || !source) return "已发布";
  return source;
}

function percent(used, total) {
  if (!total) return 0;
  return (used / total) * 100;
}

async function request(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    throw new Error(await response.text());
  }
  return response;
}

async function getJSON(url, options = {}) {
  const response = await request(url, options);
  return response.json();
}

async function sendJSON(url, method, payload = {}) {
  return getJSON(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

function renderCards(summary) {
  const sample = summary.sample;
  const entries = [
    ["模式", formatMode(summary.mode)],
    ["CPU", `${sample.cpu_percent.toFixed(1)}%`],
    ["负载", `${sample.load_1.toFixed(2)} / ${sample.load_5.toFixed(2)} / ${sample.load_15.toFixed(2)}`],
    ["内存", `${bytes(sample.mem_used_bytes)} / ${bytes(sample.mem_total_bytes)}`],
    ["交换分区", `${bytes(sample.swap_used_bytes)} / ${bytes(sample.swap_total_bytes)}`],
    ["磁盘", `${bytes(sample.disk_used_bytes)} / ${bytes(sample.disk_total_bytes)}`],
    ["网络实时", `↓ ${rate(sample.net_rx_bps)} · ↑ ${rate(sample.net_tx_bps)}`],
    ["网络平均", `↓ ${rate(sample.net_rx_avg_bps)} · ↑ ${rate(sample.net_tx_avg_bps)}`],
    ["网络接口", sample.primary_interface || "未知"],
    ["更新时间", summary.updated_at ? new Date(summary.updated_at).toLocaleString() : "-"],
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
    row.innerHTML = `<td colspan="4">${summary.docker_error || "暂无容器"}</td>`;
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
  await request("/api/heartbeat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id: state.viewerId }),
  });
}

async function setRetention(days) {
  await request("/api/settings/retention", {
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
  if (!window.confirm("确定要清空所有采样历史吗？")) {
    return;
  }
  await request("/api/history/clear", { method: "POST" });
  state.history = [];
  state.realtime = [];
  renderCharts();
}

function setNotice(message, isError = false) {
  clashNoticeEl.textContent = message;
  clashNoticeEl.classList.remove("hidden", "error");
  if (isError) {
    clashNoticeEl.classList.add("error");
  }
}

function clearNotice() {
  clashNoticeEl.textContent = "";
  clashNoticeEl.classList.add("hidden");
  clashNoticeEl.classList.remove("error");
}

function renderClashStatus(status) {
  const items = [
    ["当前 Token", status.token || "-"],
    ["订阅 URL", status.subscription_url || "-"],
    ["配置路径", status.config_path || "-"],
    ["脚本路径", status.script_path || "-"],
    ["GeoIP URL", status.geoip_url || "-"],
    ["GeoSite URL", status.geosite_url || "-"],
  ];
  clashStatusEl.innerHTML = items.map(([label, value]) => `
    <div class="status-item">
      <span>${label}</span>
      <code>${value}</code>
    </div>
  `).join("");
}

function renderClashLogs(logs) {
  clashOperationsLogEl.textContent = (logs.operations || []).join("\n") || "暂无操作记录。";
  clashGeodataLogEl.textContent = (logs.geodata || []).join("\n") || "暂无 Geo 数据日志。";
}

async function refreshClashStatus() {
  state.clash.status = await getJSON("/api/clash/status");
  renderClashStatus(state.clash.status);
}

async function refreshClashConfig() {
  const doc = await getJSON("/api/clash/config");
  clashConfigEditorEl.value = doc.content || "";
  clashConfigSourceEl.textContent = formatDocumentSource(doc.source);
}

async function refreshClashScript() {
  const doc = await getJSON("/api/clash/script");
  clashScriptEditorEl.value = doc.content || "";
  clashScriptSourceEl.textContent = formatDocumentSource(doc.source);
}

async function refreshClashLogs() {
  state.clash.logs = await getJSON("/api/clash/logs?limit=80");
  renderClashLogs(state.clash.logs);
}

async function refreshClashAll() {
  clearNotice();
  await Promise.all([
    refreshClashStatus(),
    refreshClashConfig(),
    refreshClashScript(),
    refreshClashLogs(),
  ]);
}

async function saveClashConfig() {
  await sendJSON("/api/clash/config", "PUT", { content: clashConfigEditorEl.value });
  setNotice("已保存配置草稿。");
  await refreshClashConfig();
}

async function validateClashConfig() {
  await request("/api/clash/config/validate", { method: "POST" });
  setNotice("配置校验通过。");
}

async function publishClashConfig() {
  await request("/api/clash/config/publish", { method: "POST" });
  setNotice("配置已发布。");
  await refreshClashAll();
}

async function saveClashScript() {
  await sendJSON("/api/clash/script", "PUT", { content: clashScriptEditorEl.value });
  setNotice("已保存脚本草稿。");
  await refreshClashScript();
}

async function validateClashScript() {
  await request("/api/clash/script/validate", { method: "POST" });
  setNotice("脚本校验通过。");
}

async function publishClashScript() {
  await request("/api/clash/script/publish", { method: "POST" });
  setNotice("脚本已发布。");
  await refreshClashAll();
}

async function updateClashGeodata() {
  await request("/api/clash/geodata/update", { method: "POST" });
  setNotice("已触发 Geo 数据更新。");
  await refreshClashLogs();
}

async function rotateClashToken() {
  if (!window.confirm("确定要立即轮换 Token 吗？旧 Token 会立刻失效。")) {
    return;
  }
  const status = await getJSON("/api/clash/token/rotate", { method: "POST" });
  setNotice(`Token 已轮换为 ${status.token}。`);
  await refreshClashAll();
}

async function copySubscriptionURL() {
  const value = state.clash.status?.subscription_url || "";
  if (!value) {
    throw new Error("订阅 URL 为空。");
  }
  await navigator.clipboard.writeText(value);
  setNotice("订阅 URL 已复制。");
}

function switchTab(tab) {
  state.activeTab = tab;
  for (const button of document.querySelectorAll(".tab-button")) {
    button.classList.toggle("active", button.dataset.tab === tab);
  }
  overviewView.classList.toggle("hidden", tab !== "overview");
  clashView.classList.toggle("hidden", tab !== "clash");
  if (tab === "clash") {
    refreshClashAll().catch(handleClashError);
  }
}

function handleClashError(error) {
  console.error(error);
  setNotice(error.message, true);
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

for (const button of document.querySelectorAll(".tab-button")) {
  button.addEventListener("click", () => switchTab(button.dataset.tab));
}

document.getElementById("clash-refresh").addEventListener("click", () => refreshClashAll().catch(handleClashError));
document.getElementById("clash-copy-subscription").addEventListener("click", () => copySubscriptionURL().catch(handleClashError));
document.getElementById("clash-update-geodata").addEventListener("click", () => updateClashGeodata().catch(handleClashError));
document.getElementById("clash-rotate-token").addEventListener("click", () => rotateClashToken().catch(handleClashError));
document.getElementById("clash-config-reload").addEventListener("click", () => refreshClashConfig().catch(handleClashError));
document.getElementById("clash-config-save").addEventListener("click", () => saveClashConfig().catch(handleClashError));
document.getElementById("clash-config-validate").addEventListener("click", () => validateClashConfig().catch(handleClashError));
document.getElementById("clash-config-publish").addEventListener("click", () => publishClashConfig().catch(handleClashError));
document.getElementById("clash-script-reload").addEventListener("click", () => refreshClashScript().catch(handleClashError));
document.getElementById("clash-script-save").addEventListener("click", () => saveClashScript().catch(handleClashError));
document.getElementById("clash-script-validate").addEventListener("click", () => validateClashScript().catch(handleClashError));
document.getElementById("clash-script-publish").addEventListener("click", () => publishClashScript().catch(handleClashError));
document.getElementById("clash-logs-refresh").addEventListener("click", () => refreshClashLogs().catch(handleClashError));

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
