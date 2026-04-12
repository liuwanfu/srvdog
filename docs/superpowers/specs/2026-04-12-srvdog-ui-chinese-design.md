# srvdog UI Chinese Design

**Date:** 2026-04-12

**Goal:** Update the srvdog web UI so user-facing text is shown in Chinese by default, while keeping product names, protocol names, file formats, and technical units in their original form.

## Requirements

- Translate user-facing labels, buttons, headings, status text, empty states, confirmation dialogs, and success/error notices in the existing web UI to Chinese.
- Keep proper nouns and technical terms unchanged where translation would reduce clarity, including:
  - `srvdog`
  - `Clash`
  - `Docker`
  - `JSON`
  - `CSV`
  - `YAML`
  - `GeoIP`
  - `GeoSite`
  - technical units such as `CPU`, `%`, `KB/s`
- Do not change backend API routes, payload fields, or server behavior.
- Preserve the current layout, interaction flow, and chart behavior.

## Chosen Approach

Translate the existing static frontend text in place.

- Update `web/index.html` for visible static labels and button text.
- Update `web/app.js` for runtime-generated labels, notices, confirmations, and empty-state text.
- Update the page language metadata from English to Chinese.

This is the best fit because the current UI is small, the text surface is easy to audit, and introducing a localization system would add unnecessary complexity for a single-language requirement.

## Translation Rules

### Overview area

- Translate dashboard controls such as retention, time window, export, and clear-history actions into natural Chinese.
- Translate summary card labels such as mode, load, memory, swap, disk, network, interface, and updated time.
- Keep chart titles readable while preserving technical units, for example:
  - `CPU %`
  - `内存 %`
  - `磁盘 %`
  - `网络 KB/s`
- Translate the Docker table headers except the proper noun `Docker` itself.

### Clash area

- Translate section headings, action buttons, and notices into Chinese.
- Keep `Clash`, `GeoIP`, `GeoSite`, `YAML`, and URL-related terms where they are clearer untranslated or partially untranslated.
- Translate operational messages such as save, publish, validate, rotate-token, and copy-subscription feedback into Chinese.

### Empty and fallback states

- Replace English fallbacks such as `No containers`, `No operations yet.`, `No geodata log yet.`, `published`, `n/a`, and similar strings with Chinese equivalents.
- Keep symbolic placeholders such as `-` unchanged.

## Non-Goals

- No multi-language toggle or i18n framework.
- No translation of API field names, DOM ids, or internal variable names.
- No redesign of layout, styles, or interaction flow.

## Testing

- Manually review the page to confirm the visible UI is Chinese by default.
- Verify that runtime-generated notices and empty states also appear in Chinese.
- Re-run existing frontend tests to confirm the UI text changes do not break unrelated JavaScript behavior.
