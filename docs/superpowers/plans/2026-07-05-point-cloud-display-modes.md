# Point Cloud RGB and Intensity Display Modes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add explicit point-cloud display modes for default cyan, scan RGB, and scan intensity, with a ribbon dropdown to switch the selected cloud's mode.

**Architecture:** Extend the Apache-2.0 API contract first, then teach the host point-cloud model to retain color channels and display mode metadata. Rendering stays in the existing cross-marker pipeline; the new work only threads per-point colors into the draw items and exposes a combo selector in the point-cloud ribbon panel.

**Tech Stack:** Go, `oblikovati.org/api` contract/wire/client, host `app`/`model`/`renderer`, ribbon command registry, headless tests.

---

### Task 1: Extend the public point-cloud API

**Files:**
- Modify: `../Oblikovati.API/types/point_cloud_display_mode.go`
- Modify: `../Oblikovati.API/wire/point_clouds.go`
- Modify: `../Oblikovati.API/wire/methods.go`
- Modify: `../Oblikovati.API/client/point_clouds.go`
- Test: `../Oblikovati.API/client/point_clouds_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPointCloudsSetDisplayModeMarshals(t *testing.T) {
	ft := &fakeTransport{}
	c := New(ft)
	_, err := c.PointClouds().SetDisplayMode("Scan", types.PointCloudDisplayModeRGB)
	if err != nil {
		t.Fatalf("SetDisplayMode: %v", err)
	}
	if ft.method != wire.MethodPointCloudsSetDisplayMode {
		t.Fatalf("method = %q, want %q", ft.method, wire.MethodPointCloudsSetDisplayMode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./client -run TestPointCloudsSetDisplayModeMarshals -v`
Expected: fail because the enum/method/client entry do not exist yet.

- [ ] **Step 3: Write minimal implementation**

```go
type PointCloudDisplayMode int32

const (
	PointCloudDisplayModeDefault PointCloudDisplayMode = 1
	PointCloudDisplayModeRGB PointCloudDisplayMode = 2
	PointCloudDisplayModeIntensity PointCloudDisplayMode = 3
)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./...`
Expected: pass for the API module additions.

- [ ] **Step 5: Commit**

```bash
git add ../Oblikovati.API/types/point_cloud_display_mode.go ../Oblikovati.API/wire/point_clouds.go ../Oblikovati.API/wire/methods.go ../Oblikovati.API/client/point_clouds.go ../Oblikovati.API/client/point_clouds_test.go
git commit -m "feat(api): add point cloud display mode contract"
```

### Task 2: Store scan channels and display mode in the host model

**Files:**
- Modify: `model/pointcloud/pointcloud.go`
- Modify: `model/pointcloud/reader.go`
- Modify: `model/pointcloud/ply.go`
- Modify: `model/pointcloud/las.go`
- Modify: `model/pointcloud/e57.go`
- Modify: `model/doc/pointcloud_records.go`
- Modify: `model/compdef/pointcloud_persist.go`
- Test: `model/pointcloud/pointcloud_test.go`
- Test: `model/pointcloud/ply_test.go`
- Test: `model/pointcloud/las_test.go`
- Test: `model/pointcloud/e57_test.go`
- Test: `model/compdef/pointcloud_persist_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestASCIIReaderParsesRGBAndIntensity(t *testing.T) { /* ... */ }
func TestPointCloudDisplayModePersists(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./model/pointcloud ./model/compdef -run 'TestASCIIReaderParsesRGBAndIntensity|TestPointCloudDisplayModePersists' -v`
Expected: fail because point samples only carry XYZ and the record has no mode field.

- [ ] **Step 3: Write minimal implementation**

```go
type Sample struct {
	Point math.Point3
	RGB   [3]float32
	HasRGB bool
	Intensity float64
	HasIntensity bool
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./model/pointcloud ./model/compdef`
Expected: pass with channel-aware readers and point-cloud mode persistence.

- [ ] **Step 5: Commit**

```bash
git add model/pointcloud model/doc/pointcloud_records.go model/compdef/pointcloud_persist.go
git commit -m "feat(pointcloud): preserve scan channels and display mode"
```

### Task 3: Render colored point-cloud markers

**Files:**
- Modify: `renderer/point_markers.go`
- Modify: `app/point_cloud_render.go`
- Test: `renderer/point_markers_test.go`
- Test: `app/point_cloud_render_test.go`
- Test: `app/point_cloud_render_perf_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestPointMarkersSupportsPerVertexColors(t *testing.T) { /* ... */ }
func TestPointCloudItemsUsesSelectedDisplayMode(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./renderer ./app -run 'TestPointMarkersSupportsPerVertexColors|TestPointCloudItemsUsesSelectedDisplayMode' -v`
Expected: fail because the marker helper still only accepts one broadcast color.

- [ ] **Step 3: Write minimal implementation**

```go
func PointMarkersColored(points []math.Point3, colors [][4]float32, size float64, objectID uint64) *DrawItem
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./renderer ./app`
Expected: pass and preserve the current cyan fallback path.

- [ ] **Step 5: Commit**

```bash
git add renderer/point_markers.go app/point_cloud_render.go
git commit -m "feat(renderer): support colored point-cloud markers"
```

### Task 4: Add the point-cloud ribbon selector

**Files:**
- Modify: `app/commands_standard.go`
- Modify: `app/ribbon.go`
- Modify: `app/point_cloud_ribbon_test.go`
- Modify: `app/session.go` if a point-cloud mode field is required for selection state
- Modify: `addin/router/point_clouds.go`
- Modify: `addin/router/point_clouds_test.go`
- Modify: `../Oblikovati.API/wire/point_clouds.go` if the mode is exposed over wire state

- [ ] **Step 1: Write the failing test**

```go
func TestPointCloudRibbonDisplayModeSelector(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app ./addin/router -run TestPointCloudRibbonDisplayModeSelector -v`
Expected: fail because the point-cloud panel has no selector yet.

- [ ] **Step 3: Write minimal implementation**

```go
NewCommand("PointCloud.DisplayRGB", "RGB", "Display Mode", func(s *Session) error { /* ... */ }).WithKind(ComboControl)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app ./addin/router`
Expected: pass and the selector appears near the other point-cloud buttons.

- [ ] **Step 5: Commit**

```bash
git add app/commands_standard.go app/ribbon.go app/point_cloud_ribbon_test.go addin/router/point_clouds.go addin/router/point_clouds_test.go
git commit -m "feat(ui): add point cloud display mode selector"
```

### Task 5: Run regression suite

**Files:**
- None

- [ ] **Step 1: Run the focused tests**

Run: `go test ./model/pointcloud ./model/compdef ./renderer ./app ./addin/router ./...`
Expected: pass.

- [ ] **Step 2: Validate no unrelated files changed**

Run: `git status --short`
Expected: only intended implementation files plus any user-owned pre-existing edits outside the scope.
