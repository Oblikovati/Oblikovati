// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/pointcloud"
)

// Attached point clouds over the wire (M17-F06, #645): attach a scan file to the active part,
// query and place the resulting cloud, and budget its display. The scan bytes are embedded in
// the document resource table (ADR-0031) on attach, so the cloud round-trips in the .obk.

// registerPointCloudHandlers wires the pointClouds.* methods.
func (r *Router) registerPointCloudHandlers() {
	r.readOnly(wire.MethodPointCloudsAttach, typed(attachPointCloud))
	r.readOnly(wire.MethodPointCloudsList, partQuery(listPointClouds))
	r.readOnly(wire.MethodPointCloudsGet, typedPart(getPointCloud))
	r.mutating(wire.MethodPointCloudsDelete, "Delete Point Cloud", typedPart(deletePointCloud))
	r.readOnly(wire.MethodPointCloudsSetVisible, typedPart(setPointCloudVisible))
	r.mutating(wire.MethodPointCloudsSetTransform, "Move Point Cloud", typedPart(setPointCloudTransform))
	r.mutating(wire.MethodPointCloudsSetScale, "Scale Point Cloud", typedPart(setPointCloudScale))
	r.readOnly(wire.MethodPointCloudsSetDensity, typedPart(setPointCloudDensity))
	r.mutating(wire.MethodPointCloudsSetDisplayMode, "Point Cloud Display Mode", typedPart(setPointCloudDisplayMode))
	r.readOnly(wire.MethodPointCloudsToModelSpace, typedPart(pointCloudToModelSpace))
	r.readOnly(wire.MethodPointCloudsFromModelSpace, typedPart(pointCloudFromModelSpace))
	r.mutating(wire.MethodPointCloudsAddCrop, "Crop Point Cloud", typedPart(addPointCloudCrop))
	r.readOnly(wire.MethodPointCloudsListCrops, typedPart(listPointCloudCrops))
	r.mutating(wire.MethodPointCloudsDeleteCrop, "Crop Point Cloud", typedPart(deletePointCloudCrop))
	r.readOnly(wire.MethodPointCloudsSetCropActive, typedPart(setPointCloudCropActive))
	r.readOnly(wire.MethodPointCloudsFitPlane, typed(fitPointCloudPlane))
	r.readOnly(wire.MethodPointCloudsNearestPoint, typedPart(nearestPointCloudPoint))
}

// attachPointCloud reads the scan file, embeds its bytes as a resource, decodes its points, and
// attaches the cloud to the active part under a unique name.
func attachPointCloud(s *app.Session, in wire.AttachPointCloudArgs) (wire.PointCloudInfo, error) {
	// Per-record decode warnings (#1646) are not yet surfaced over the wire: PointCloudInfo has
	// no warnings slot (an api/wire DTO addition in the Apache-2.0 sibling repo).
	pc, _, err := s.AttachPointCloud(in.Name, in.FullFileName)
	if err != nil {
		return wire.PointCloudInfo{}, fmt.Errorf("pointClouds.attach: %w", err)
	}
	return pointCloudInfo(pc), nil
}

// listPointClouds enumerates the active part's clouds.
func listPointClouds(_ *app.Session, part *compdef.PartComponentDefinition) (wire.ListPointCloudsResult, error) {
	clouds := projectAll(part.PointClouds(), func(_ int, pc *pointcloud.PointCloud) wire.PointCloudInfo {
		return pointCloudInfo(pc)
	})
	return wire.ListPointCloudsResult{PointClouds: clouds}, nil
}

// getPointCloud returns one named cloud's state.
func getPointCloud(_ *app.Session, part *compdef.PartComponentDefinition, in wire.PointCloudNameArgs) (wire.PointCloudInfo, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsGet)
	if err != nil {
		return wire.PointCloudInfo{}, err
	}
	return pointCloudInfo(pc), nil
}

// deletePointCloud removes a named cloud from the active part.
func deletePointCloud(_ *app.Session, part *compdef.PartComponentDefinition, in wire.PointCloudNameArgs) (wire.DeletePointCloudResult, error) {
	deleted := part.PointClouds().Remove(in.Name)
	return wire.DeletePointCloudResult{Name: in.Name, Deleted: deleted}, nil
}

// setPointCloudVisible shows or hides a named cloud.
func setPointCloudVisible(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetPointCloudVisibleArgs) (wire.PointCloudInfo, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsSetVisible)
	if err != nil {
		return wire.PointCloudInfo{}, err
	}
	pc.SetVisible(in.Visible)
	return pointCloudInfo(pc), nil
}

// setPointCloudTransform sets a named cloud's placement.
func setPointCloudTransform(s *app.Session, part *compdef.PartComponentDefinition, in wire.SetPointCloudTransformArgs) (wire.PointCloudInfo, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsSetTransform)
	if err != nil {
		return wire.PointCloudInfo{}, err
	}
	pc.SetTransform(math.Matrix4FromCells(in.Transform.Cells))
	s.RecomputeAfterPointCloudMove() // datums built on the cloud follow it (#645)
	return pointCloudInfo(pc), nil
}

// setPointCloudScale sets a named cloud's uniform scale, rejecting a non-positive factor.
func setPointCloudScale(s *app.Session, part *compdef.PartComponentDefinition, in wire.SetPointCloudScaleArgs) (wire.PointCloudInfo, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsSetScale)
	if err != nil {
		return wire.PointCloudInfo{}, err
	}
	if !pc.SetScale(in.Scale) {
		return wire.PointCloudInfo{}, fmt.Errorf("pointClouds.setScale: scale must be positive, got %v", in.Scale)
	}
	s.RecomputeAfterPointCloudMove() // datums built on the cloud follow it (#645)
	return pointCloudInfo(pc), nil
}

// setPointCloudDensity sets a named cloud's display budget.
func setPointCloudDensity(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetPointCloudDensityArgs) (wire.PointCloudInfo, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsSetDensity)
	if err != nil {
		return wire.PointCloudInfo{}, err
	}
	pc.SetMaximumPointCount(in.MaximumPointCount)
	return pointCloudInfo(pc), nil
}

// setPointCloudDisplayMode sets a named cloud's rendering mode.
func setPointCloudDisplayMode(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetPointCloudDisplayModeArgs) (wire.PointCloudInfo, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsSetDisplayMode)
	if err != nil {
		return wire.PointCloudInfo{}, err
	}
	if !pc.SetDisplayMode(in.DisplayMode) {
		return wire.PointCloudInfo{}, fmt.Errorf("pointClouds.setDisplayMode: invalid mode %q; expected one of %v", in.DisplayMode, types.AllPointCloudDisplayModes())
	}
	return pointCloudInfo(pc), nil
}

// pointCloudToModelSpace maps a cloud-local point into model space.
func pointCloudToModelSpace(_ *app.Session, part *compdef.PartComponentDefinition, in wire.PointCloudSpaceArgs) (wire.PointCloudSpaceResult, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsToModelSpace)
	if err != nil {
		return wire.PointCloudSpaceResult{}, err
	}
	m := pc.ToModelSpace(point3Of(in.Point))
	return wire.PointCloudSpaceResult{Point: pointOf(m), OK: true}, nil
}

// pointCloudFromModelSpace maps a model-space point into a cloud's local space.
func pointCloudFromModelSpace(_ *app.Session, part *compdef.PartComponentDefinition, in wire.PointCloudSpaceArgs) (wire.PointCloudSpaceResult, error) {
	pc, err := cloudByName(part, in.Name, wire.MethodPointCloudsFromModelSpace)
	if err != nil {
		return wire.PointCloudSpaceResult{}, err
	}
	p, ok := pc.FromModelSpace(point3Of(in.Point))
	return wire.PointCloudSpaceResult{Point: pointOf(p), OK: ok}, nil
}

// cloudByName resolves a cloud by name on the part, naming the method on a miss.
func cloudByName(part *compdef.PartComponentDefinition, name, method string) (*pointcloud.PointCloud, error) {
	pc, ok := part.PointClouds().ByName(name)
	if !ok {
		return nil, fmt.Errorf("%s: no point cloud named %q", method, name)
	}
	return pc, nil
}

// pointCloudInfo maps a cloud to its wire shape.
func pointCloudInfo(pc *pointcloud.PointCloud) wire.PointCloudInfo {
	return wire.PointCloudInfo{
		Name:                pc.Name(),
		Source:              pc.SourceFullFileName(),
		Visible:             pc.Visible(),
		DisplayMode:         pc.DisplayMode(),
		Scale:               pc.Scale(),
		Transform:           types.Matrix{Cells: pc.Transform().Cells()},
		TotalPointCount:     pc.TotalPointCount(),
		DisplayedPointCount: pc.DisplayedPointCount(),
		MaximumPointCount:   pc.MaximumPointCount(),
	}
}

// addPointCloudCrop adds an active crop over the requested box on a named cloud.
func addPointCloudCrop(_ *app.Session, part *compdef.PartComponentDefinition, in wire.AddPointCloudCropArgs) (wire.PointCloudCropInfo, error) {
	pc, err := cloudByName(part, in.Cloud, wire.MethodPointCloudsAddCrop)
	if err != nil {
		return wire.PointCloudCropInfo{}, err
	}
	crop := pc.AddCrop(math.NewBox(point3Of(in.Min), point3Of(in.Max)))
	return cropInfo(in.Cloud, crop), nil
}

// fitPointCloudPlane fits a least-squares work plane to the named cloud's displayed points and
// returns the created work plane's name with the fitted origin (centroid) and unit normal.
func fitPointCloudPlane(s *app.Session, in wire.FitPointCloudPlaneArgs) (wire.FitPointCloudPlaneResult, error) {
	wp, plane, err := s.CreatePointCloudPlane(in.Cloud)
	if err != nil {
		return wire.FitPointCloudPlaneResult{}, err
	}
	n := plane.Normal()
	return wire.FitPointCloudPlaneResult{
		WorkPlane: wp.Name(),
		Origin:    pointOf(plane.Origin),
		Normal:    pointOf(math.P3(n.X, n.Y, n.Z)),
	}, nil
}

// nearestPointCloudPoint snaps the query point onto the named cloud, returning its nearest scan
// point in model space and the distance to it.
func nearestPointCloudPoint(_ *app.Session, part *compdef.PartComponentDefinition, in wire.NearestPointArgs) (wire.NearestPointResult, error) {
	pc, err := cloudByName(part, in.Cloud, wire.MethodPointCloudsNearestPoint)
	if err != nil {
		return wire.NearestPointResult{}, err
	}
	query := point3Of(in.Point)
	nearest, found := pc.NearestModelPoint(query)
	return wire.NearestPointResult{
		Point:    pointOf(nearest),
		Distance: float64(query.DistanceTo(nearest)),
		Found:    found,
	}, nil
}

// listPointCloudCrops enumerates a named cloud's crops.
func listPointCloudCrops(_ *app.Session, part *compdef.PartComponentDefinition, in wire.ListPointCloudCropsArgs) (wire.ListPointCloudCropsResult, error) {
	pc, err := cloudByName(part, in.Cloud, wire.MethodPointCloudsListCrops)
	if err != nil {
		return wire.ListPointCloudCropsResult{}, err
	}
	crops := projectAll(pc.Crops(), func(_ int, c *pointcloud.PointCloudCrop) wire.PointCloudCropInfo {
		return cropInfo(in.Cloud, c)
	})
	return wire.ListPointCloudCropsResult{Crops: crops}, nil
}

// deletePointCloudCrop removes a named crop from a cloud.
func deletePointCloudCrop(_ *app.Session, part *compdef.PartComponentDefinition, in wire.PointCloudCropArgs) (wire.DeletePointCloudCropResult, error) {
	pc, err := cloudByName(part, in.Cloud, wire.MethodPointCloudsDeleteCrop)
	if err != nil {
		return wire.DeletePointCloudCropResult{}, err
	}
	return wire.DeletePointCloudCropResult{Crop: in.Crop, Deleted: pc.Crops().Remove(in.Crop)}, nil
}

// setPointCloudCropActive toggles whether a named crop limits display.
func setPointCloudCropActive(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SetPointCloudCropActiveArgs) (wire.PointCloudCropInfo, error) {
	pc, err := cloudByName(part, in.Cloud, wire.MethodPointCloudsSetCropActive)
	if err != nil {
		return wire.PointCloudCropInfo{}, err
	}
	crop, ok := pc.Crops().ByName(in.Crop)
	if !ok {
		return wire.PointCloudCropInfo{}, fmt.Errorf("pointClouds.setCropActive: cloud %q has no crop %q", in.Cloud, in.Crop)
	}
	crop.SetActive(in.Active)
	return cropInfo(in.Cloud, crop), nil
}

// cropInfo maps a crop to its wire shape under the owning cloud's name.
func cropInfo(cloud string, c *pointcloud.PointCloudCrop) wire.PointCloudCropInfo {
	return wire.PointCloudCropInfo{
		Cloud: cloud, Crop: c.Name(), Active: c.Active(),
		Min: pointOf(c.Box().Min), Max: pointOf(c.Box().Max),
	}
}

// point3Of / pointOf bridge the wire point type and math.Point3.
func point3Of(p types.Point) math.Point3 {
	return math.P3(math.Scalar(p.X), math.Scalar(p.Y), math.Scalar(p.Z))
}

func pointOf(p math.Point3) types.Point {
	return types.Point{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)}
}
