// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
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
	r.readOnly(wire.MethodPointCloudsAttach, attachPointCloud)
	r.readOnly(wire.MethodPointCloudsList, listPointClouds)
	r.readOnly(wire.MethodPointCloudsGet, getPointCloud)
	r.readOnly(wire.MethodPointCloudsDelete, deletePointCloud)
	r.readOnly(wire.MethodPointCloudsSetVisible, setPointCloudVisible)
	r.readOnly(wire.MethodPointCloudsSetTransform, setPointCloudTransform)
	r.readOnly(wire.MethodPointCloudsSetScale, setPointCloudScale)
	r.readOnly(wire.MethodPointCloudsSetDensity, setPointCloudDensity)
	r.readOnly(wire.MethodPointCloudsToModelSpace, pointCloudToModelSpace)
	r.readOnly(wire.MethodPointCloudsFromModelSpace, pointCloudFromModelSpace)
	r.readOnly(wire.MethodPointCloudsAddCrop, addPointCloudCrop)
	r.readOnly(wire.MethodPointCloudsListCrops, listPointCloudCrops)
	r.readOnly(wire.MethodPointCloudsDeleteCrop, deletePointCloudCrop)
	r.readOnly(wire.MethodPointCloudsSetCropActive, setPointCloudCropActive)
	r.readOnly(wire.MethodPointCloudsFitPlane, fitPointCloudPlane)
	r.readOnly(wire.MethodPointCloudsNearestPoint, nearestPointCloudPoint)
}

// attachPointCloud reads the scan file, embeds its bytes as a resource, decodes its points, and
// attaches the cloud to the active part under a unique name.
func attachPointCloud(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AttachPointCloudArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := s.AttachPointCloud(in.Name, in.FullFileName)
	if err != nil {
		return nil, fmt.Errorf("pointClouds.attach: %w", err)
	}
	return json.Marshal(pointCloudInfo(pc))
}

// listPointClouds enumerates the active part's clouds.
func listPointClouds(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	clouds := part.PointClouds()
	out := make([]wire.PointCloudInfo, 0, clouds.Count())
	for i := 0; i < clouds.Count(); i++ {
		out = append(out, pointCloudInfo(clouds.Item(i)))
	}
	return json.Marshal(wire.ListPointCloudsResult{PointClouds: out})
}

// getPointCloud returns one named cloud's state.
func getPointCloud(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	pc, _, err := resolvePointCloud(s, raw, wire.MethodPointCloudsGet)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pointCloudInfo(pc))
}

// deletePointCloud removes a named cloud from the active part.
func deletePointCloud(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.PointCloudNameArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	deleted := part.PointClouds().Remove(in.Name)
	return json.Marshal(wire.DeletePointCloudResult{Name: in.Name, Deleted: deleted})
}

// setPointCloudVisible shows or hides a named cloud.
func setPointCloudVisible(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetPointCloudVisibleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Name, wire.MethodPointCloudsSetVisible)
	if err != nil {
		return nil, err
	}
	pc.SetVisible(in.Visible)
	return json.Marshal(pointCloudInfo(pc))
}

// setPointCloudTransform sets a named cloud's placement.
func setPointCloudTransform(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetPointCloudTransformArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Name, wire.MethodPointCloudsSetTransform)
	if err != nil {
		return nil, err
	}
	pc.SetTransform(math.Matrix4FromCells(in.Transform.Cells))
	s.RecomputeAfterPointCloudMove() // datums built on the cloud follow it (#645)
	return json.Marshal(pointCloudInfo(pc))
}

// setPointCloudScale sets a named cloud's uniform scale, rejecting a non-positive factor.
func setPointCloudScale(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetPointCloudScaleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Name, wire.MethodPointCloudsSetScale)
	if err != nil {
		return nil, err
	}
	if !pc.SetScale(in.Scale) {
		return nil, fmt.Errorf("pointClouds.setScale: scale must be positive, got %v", in.Scale)
	}
	s.RecomputeAfterPointCloudMove() // datums built on the cloud follow it (#645)
	return json.Marshal(pointCloudInfo(pc))
}

// setPointCloudDensity sets a named cloud's display budget.
func setPointCloudDensity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetPointCloudDensityArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Name, wire.MethodPointCloudsSetDensity)
	if err != nil {
		return nil, err
	}
	pc.SetMaximumPointCount(in.MaximumPointCount)
	return json.Marshal(pointCloudInfo(pc))
}

// pointCloudToModelSpace maps a cloud-local point into model space.
func pointCloudToModelSpace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	in, pc, err := pointCloudSpaceRequest(s, raw, wire.MethodPointCloudsToModelSpace)
	if err != nil {
		return nil, err
	}
	m := pc.ToModelSpace(point3Of(in.Point))
	return json.Marshal(wire.PointCloudSpaceResult{Point: pointOf(m), OK: true})
}

// pointCloudFromModelSpace maps a model-space point into a cloud's local space.
func pointCloudFromModelSpace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	in, pc, err := pointCloudSpaceRequest(s, raw, wire.MethodPointCloudsFromModelSpace)
	if err != nil {
		return nil, err
	}
	p, ok := pc.FromModelSpace(point3Of(in.Point))
	return json.Marshal(wire.PointCloudSpaceResult{Point: pointOf(p), OK: ok})
}

// pointCloudSpaceRequest decodes a space-conversion request and resolves its cloud.
func pointCloudSpaceRequest(s *app.Session, raw json.RawMessage, method string) (wire.PointCloudSpaceArgs, *pointcloud.PointCloud, error) {
	var in wire.PointCloudSpaceArgs
	if err := decode(raw, &in); err != nil {
		return in, nil, err
	}
	pc, err := namedCloud(s, in.Name, method)
	return in, pc, err
}

// resolvePointCloud decodes a name-only request and resolves its cloud on the active part.
func resolvePointCloud(s *app.Session, raw json.RawMessage, method string) (*pointcloud.PointCloud, *compdef.PartComponentDefinition, error) {
	var in wire.PointCloudNameArgs
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, err
	}
	pc, ok := part.PointClouds().ByName(in.Name)
	if !ok {
		return nil, nil, fmt.Errorf("%s: no point cloud named %q", method, in.Name)
	}
	return pc, part, nil
}

// namedCloud resolves a cloud by name on the active part.
func namedCloud(s *app.Session, name, method string) (*pointcloud.PointCloud, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
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
		Scale:               pc.Scale(),
		Transform:           types.Matrix{Cells: pc.Transform().Cells()},
		TotalPointCount:     pc.TotalPointCount(),
		DisplayedPointCount: pc.DisplayedPointCount(),
		MaximumPointCount:   pc.MaximumPointCount(),
	}
}

// addPointCloudCrop adds an active crop over the requested box on a named cloud.
func addPointCloudCrop(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddPointCloudCropArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Cloud, wire.MethodPointCloudsAddCrop)
	if err != nil {
		return nil, err
	}
	crop := pc.AddCrop(math.NewBox(point3Of(in.Min), point3Of(in.Max)))
	return json.Marshal(cropInfo(in.Cloud, crop))
}

// fitPointCloudPlane fits a least-squares work plane to the named cloud's displayed points and
// returns the created work plane's name with the fitted origin (centroid) and unit normal.
func fitPointCloudPlane(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.FitPointCloudPlaneArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	wp, plane, err := s.CreatePointCloudPlane(in.Cloud)
	if err != nil {
		return nil, err
	}
	n := plane.Normal()
	return json.Marshal(wire.FitPointCloudPlaneResult{
		WorkPlane: wp.Name(),
		Origin:    pointOf(plane.Origin),
		Normal:    pointOf(math.P3(n.X, n.Y, n.Z)),
	})
}

// nearestPointCloudPoint snaps the query point onto the named cloud, returning its nearest scan
// point in model space and the distance to it.
func nearestPointCloudPoint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.NearestPointArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Cloud, wire.MethodPointCloudsNearestPoint)
	if err != nil {
		return nil, err
	}
	query := point3Of(in.Point)
	nearest, found := pc.NearestModelPoint(query)
	return json.Marshal(wire.NearestPointResult{
		Point:    pointOf(nearest),
		Distance: float64(query.DistanceTo(nearest)),
		Found:    found,
	})
}

// listPointCloudCrops enumerates a named cloud's crops.
func listPointCloudCrops(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.ListPointCloudCropsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Cloud, wire.MethodPointCloudsListCrops)
	if err != nil {
		return nil, err
	}
	crops := pc.Crops()
	out := make([]wire.PointCloudCropInfo, 0, crops.Count())
	for i := 0; i < crops.Count(); i++ {
		out = append(out, cropInfo(in.Cloud, crops.Item(i)))
	}
	return json.Marshal(wire.ListPointCloudCropsResult{Crops: out})
}

// deletePointCloudCrop removes a named crop from a cloud.
func deletePointCloudCrop(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.PointCloudCropArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Cloud, wire.MethodPointCloudsDeleteCrop)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.DeletePointCloudCropResult{Crop: in.Crop, Deleted: pc.Crops().Remove(in.Crop)})
}

// setPointCloudCropActive toggles whether a named crop limits display.
func setPointCloudCropActive(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetPointCloudCropActiveArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pc, err := namedCloud(s, in.Cloud, wire.MethodPointCloudsSetCropActive)
	if err != nil {
		return nil, err
	}
	crop, ok := pc.Crops().ByName(in.Crop)
	if !ok {
		return nil, fmt.Errorf("pointClouds.setCropActive: cloud %q has no crop %q", in.Cloud, in.Crop)
	}
	crop.SetActive(in.Active)
	return json.Marshal(cropInfo(in.Cloud, crop))
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
