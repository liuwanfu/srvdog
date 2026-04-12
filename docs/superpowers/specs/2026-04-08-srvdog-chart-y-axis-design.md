# srvdog Chart Y-Axis Design

**Date:** 2026-04-08

**Goal:** Add visible y-axis numeric labels to the existing lightweight canvas charts without introducing a charting library or changing any backend data APIs.

## Requirements

- CPU, Memory, and Disk charts keep their current percent-based titles and gain fixed numeric y-axis ticks.
- Network chart keeps the current `KB/s` unit and gains numeric y-axis ticks derived from the current visible data range.
- The canvas renderer remains lightweight and self-contained.
- Existing summary polling, history loading, and realtime polling behavior does not change.

## Chosen Approach

Keep the current canvas renderer and add a small chart helper module for axis calculations.

- Extract pure chart helpers into `web/chart-utils.js`.
- Serve that helper as a static asset alongside the existing frontend files.
- Update `web/app.js` so each chart:
  - reserves left-side space for labels
  - draws horizontal grid lines only across the plot area
  - draws right-aligned y-axis labels
  - maps line coordinates into the reduced plot area

## Axis Rules

### Percent charts

- Tick values are fixed at `0`, `25`, `50`, `75`, `100`.
- Labels are numeric only because the chart titles already contain `%`.

### Network chart

- Tick values are computed from the current max visible `KB/s` value across RX and TX series.
- The upper bound is rounded up to a "nice" value so labels are readable.
- Labels include `KB/s`.

## Testing

- Add a Go HTTP test that proves the new helper script is served by the embedded web server.
- Add Node unit tests for:
  - fixed percent tick generation
  - dynamic `KB/s` tick generation
  - coordinate projection into the chart plot area
- Perform a manual browser verification pass after implementation.
