# Point Cloud Display Controls Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use inline task execution in this session. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Combine point-cloud display mode, render density, and intensity ramp controls into one ribbon panel, with RGB as the default mode.

**Architecture:** Keep display mode as the existing per-cloud command selector. Keep render density and intensity ramp as global session viewport state, injected into the selector panel after the ribbon command model is built. Route the ramp through both CPU marker and retained GPU point-cloud render paths, and include it in the GPU display key.

**Tech Stack:** Go app model and tests, cgo Dear ImGui head wrapper, existing `types.PointCloudDisplayMode` API enum.

---

### Task 1: Session Defaults And Ramp State

**Files:**
- Modify: `app/session.go`
- Test: `app/point_cloud_gpu_test.go`
- Test: `model/pointcloud/pointcloud_test.go`

- [ ] Change new point clouds to default to `types.PointCloudDisplayModeRGB`.
- [ ] Add `PointCloudIntensityRamp() (low, high [4]float32)` and `SetPointCloudIntensityRamp(low, high [4]float32)` on `app.Session`.
- [ ] Initialize the ramp to red low `[1,0,0,1]` and yellow high `[1,1,0,1]`.
- [ ] Add tests for the RGB default and session ramp getters/setters.

### Task 2: Ribbon Model

**Files:**
- Modify: `app/commands_standard.go`
- Modify: `app/ribbon.go`
- Test: `app/point_cloud_ribbon_test.go`
- Test: `app/ribbon_reorg_test.go`

- [ ] Rename the display mode command category from `Display Mode` to `Point Cloud Display`.
- [ ] Attach the render-density slider to that same panel instead of injecting a separate `Render Density` panel.
- [ ] Add intensity ramp data to the panel only when the selected cloud display mode is `Intensity`.
- [ ] Update panel-order and point-cloud ribbon tests for the combined panel.

### Task 3: Ribbon Rendering

**Files:**
- Modify: `head/ui/chrome_ribbon.go`
- Test: `head/ui/point_cloud_ribbon_shot_test.go`

- [ ] Draw selector panels with optional controls stacked below the combo.
- [ ] Draw the render-density slider below the point-cloud display selector.
- [ ] Draw low/high color editors below density only when the panel exposes the intensity ramp.
- [ ] Keep the panel label centered under the full control group.

### Task 4: Intensity Render Mapping

**Files:**
- Modify: `app/point_cloud_render.go`
- Modify: `app/point_cloud_gpu.go`
- Test: `app/point_cloud_render_test.go`
- Test: `app/point_cloud_gpu_test.go`

- [ ] Pass the session intensity ramp into CPU marker color assembly.
- [ ] Pass the session intensity ramp into GPU vertex assembly.
- [ ] Replace grayscale intensity mapping with interpolation between low and high ramp colors.
- [ ] Include both ramp colors in `PointCloudDisplayKey()`.
- [ ] Update render and GPU tests to expect red/yellow intensity output.

### Task 5: Verification

**Commands:**
- `go test ./model/pointcloud ./app -run 'Test.*PointCloud.*'`
- `go test ./app -run 'TestSurfacesMeshTabPanelOrder|TestPointCloudRibbonButtonsWiredAndEnable|TestPointCloudGPUVerticesColorsByDisplayMode|TestPointCloudItemsColorsByDisplayMode|TestPointCloudDisplayKeyChangesOnIntensityRamp|TestPointCloudIntensityRampDefaultAndSet'`
- `cd head && go test ./ui -run TestInWindowPointCloudPanelButtons`

- [ ] Run focused tests after implementation.
- [ ] If cgo/head tests cannot run in this environment, report that explicitly.
