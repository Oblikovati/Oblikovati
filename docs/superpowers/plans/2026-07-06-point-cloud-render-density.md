# Point Cloud Render Density Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a session-level point-cloud render density slider that draws a stable random percentage of point-cloud samples.

**Architecture:** Keep the feature render-only by storing density on `app.Session` and applying it only in point-cloud render assembly before native point upload. Add a first-class ribbon slider model so the head can draw a percentage slider without abusing command buttons.

**Tech Stack:** Go, app session/ribbon model, head Dear ImGui wrapper, point-cloud GPU render path, focused Go tests.

---

### Task 1: Add session render-density state

**Files:**
- Modify: `app/session.go`
- Test: `app/point_cloud_gpu_test.go`

- [ ] Add `pointCloudRenderDensity float32` to `Session`.
- [ ] Initialize it to `100` in `newSession`.
- [ ] Add `PointCloudRenderDensity()` and `SetPointCloudRenderDensity(float32)` with `0..100` clamping.
- [ ] Test default and clamping with `go test ./app -run TestPointCloudRenderDensity -v`.

### Task 2: Apply density to point-cloud render samples

**Files:**
- Modify: `app/point_cloud_gpu.go`
- Modify: `app/point_cloud_render.go`
- Test: `app/point_cloud_gpu_test.go`

- [ ] Add a deterministic hash-threshold helper over cloud identity and model-space point values.
- [ ] Use it before appending GPU point vertices.
- [ ] Use it in the legacy marker item path for consistency.
- [ ] Include the density value in `PointCloudDisplayKey`.
- [ ] Test density count, stability, zero-density behavior, and key invalidation.

### Task 3: Add ribbon slider model and head rendering

**Files:**
- Modify: `app/ribbon.go`
- Modify: `app/ribbon_reorg_test.go`
- Modify: `app/point_cloud_ribbon_test.go`
- Modify: `head/ui/chrome_ribbon.go`

- [ ] Add `RibbonSlider` to `RibbonPanel`.
- [ ] Add a Surfaces & Mesh `Render Density` panel for part ribbons.
- [ ] Render slider panels with `native.SliderPercent`.
- [ ] Push slider edits to `Session.SetPointCloudRenderDensity`.
- [ ] Test the panel exists and exposes the default value.

### Task 4: Validate and launch

**Files:**
- No source edits expected.

- [ ] Run `gofmt` on changed Go files.
- [ ] Run targeted app tests.
- [ ] Rebuild the head.
- [ ] Launch the GUI on workspace 5.
