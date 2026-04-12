# srvdog Chart Y-Axis Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add numeric y-axis labels to the srvdog canvas charts while keeping the frontend lightweight and preserving existing polling behavior.

**Architecture:** Keep the current canvas-based renderer and introduce one small helper module for tick and layout calculations. Update the embedded web asset list and HTTP route table so the helper script is delivered with the existing page, then update the chart renderer to draw labels and lines within the new plot area.

**Tech Stack:** Go, embedded static files, plain HTML/CSS/JavaScript, Node test runner

---

### Task 1: Lock The New Static Asset And Axis Logic With Tests

**Files:**
- Modify: `internal/httpapi/server_test.go`
- Create: `web/chart-utils.test.js`

- [ ] **Step 1: Write the failing Go route test**

```go
func TestChartUtilsScriptServed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chart-utils.js", nil)
	rec := httptest.NewRecorder()

	srv := NewServer(Dependencies{
		StaticFS: fstest.MapFS{
			"index.html":      &fstest.MapFile{Data: []byte("ok")},
			"app.js":          &fstest.MapFile{Data: []byte("ok")},
			"chart-utils.js":  &fstest.MapFile{Data: []byte("chart utils")},
			"styles.css":      &fstest.MapFile{Data: []byte("ok")},
		},
	})
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
```

- [ ] **Step 2: Run the Go test to verify it fails**

Run: `& "C:\Program Files\Go\bin\go.exe" test ./internal/httpapi -run TestChartUtilsScriptServed`
Expected: FAIL because `/chart-utils.js` is not routed yet

- [ ] **Step 3: Write the failing Node chart helper tests**

```js
test('buildPercentYAxis returns fixed 0-100 ticks', () => {
  assert.deepEqual(buildPercentYAxis().ticks.map((tick) => tick.label), ['0', '25', '50', '75', '100']);
});

test('buildRateYAxis rounds to readable KB/s ticks', () => {
  assert.equal(buildRateYAxis(137).max, 200);
});

test('projectPoint stays inside the computed plot area', () => {
  const layout = createChartLayout(600, 180);
  const point = projectPoint({ index: 0, count: 2, value: 100, max: 100, layout });
  assert.equal(point.x, layout.left);
  assert.equal(point.y, layout.top);
});
```

- [ ] **Step 4: Run the Node test to verify it fails**

Run: `node --test web/chart-utils.test.js`
Expected: FAIL because `web/chart-utils.js` does not exist yet

### Task 2: Implement The Helper Module And Canvas Axis Rendering

**Files:**
- Create: `web/chart-utils.js`
- Modify: `web/app.js`
- Modify: `web/index.html`
- Modify: `web/embed.go`
- Modify: `internal/httpapi/server.go`

- [ ] **Step 1: Write the helper module with pure functions**

```js
function buildPercentYAxis() {
  return { max: 100, ticks: [0, 25, 50, 75, 100].map((value) => ({ value, label: String(value) })) };
}
```

- [ ] **Step 2: Update the page to load the helper before `app.js`**

```html
<script src="/chart-utils.js"></script>
<script src="/app.js"></script>
```

- [ ] **Step 3: Serve and embed the new helper asset**

```go
//go:embed index.html app.js chart-utils.js styles.css
var FS embed.FS

mux.Handle("GET /chart-utils.js", fileServer)
```

- [ ] **Step 4: Update the chart renderer to use the helper axis and layout data**

```js
const axis = window.SrvdogChartUtils.buildPercentYAxis();
const layout = window.SrvdogChartUtils.createChartLayout(width, height);
```

- [ ] **Step 5: Run the targeted tests to verify they pass**

Run: `& "C:\Program Files\Go\bin\go.exe" test ./internal/httpapi -run TestChartUtilsScriptServed`
Expected: PASS

Run: `node --test web/chart-utils.test.js`
Expected: PASS

### Task 3: Verify The End-To-End Dashboard Behavior

**Files:**
- Modify: `web/app.js`

- [ ] **Step 1: Run the relevant Go package tests**

Run: `& "C:\Program Files\Go\bin\go.exe" test ./...`
Expected: PASS

- [ ] **Step 2: Run the local server and inspect the chart UI**

```powershell
& "C:\Program Files\Go\bin\go.exe" run ./cmd/srvdog
```

- [ ] **Step 3: Confirm the UI behavior**

```text
CPU/Memory/Disk charts show 0-100 numeric y-axis labels.
Network chart shows numeric y-axis labels with KB/s.
Lines remain inside the plotting area and no API calls regress.
```
