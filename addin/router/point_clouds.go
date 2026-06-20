// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"os"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// Attached point clouds over the wire (M17-F06, #645): attach a scan file to the active part,
// query and place the resulting cloud, and budget its display. The scan bytes are embedded in
// the document resource table (ADR-0031) on attach, so the cloud round-trips in the .obk.

// registerPointCloudHandlers wires the pointClouds.* methods.
func (r *Router) registerPointCloudHandlers() {
	r.handlers[wire.MethodPointCloudsAttach] = attachPointCloud
	r.handlers[wire.MethodPointCloudsList] = listPointClouds
	r.handlers[wire.MethodPointCloudsGet] = getPointCloud
	r.handlers[wire.MethodPointCloudsDelete] = deletePointCloud
	r.handlers[wire.MethodPointCloudsSetVisible] = setPointCloudVisible
	r.handlers[wire.MethodPointCloudsSetTransform] = setPointCloudTransform
	r.handlers[wire.MethodPointCloudsSetScale] = setPointCloudScale
	r.handlers[wire.MethodPointCloudsSetDensity] = setPointCloudDensity
	r.handlers[wire.MethodPointCloudsToModelSpace] = pointCloudToModelSpace
	r.handlers[wire.MethodPointCloudsFromModelSpace] = pointCloudFromModelSpace
}

// attachPointCloud reads the scan file, embeds its bytes as a resource, decodes its points, and
// attaches the cloud to the active part under a unique name.
func attachPointCloud(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AttachPointCloudArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	data, points, err := loadScan(in.FullFileName)
	if err != nil {
		return nil, err
	}
	name := in.Name
	if name == "" {
		name = part.PointClouds().UniqueName("Cloud")
	}
	rid := part.AddResource(doc.Resource{Type: "PointCloudScan", Encoding: doc.EncodingUTF8, Value: data, Origin: in.FullFileName})
	pc, err := part.PointClouds().Add(name, in.FullFileName, rid, points)
	if err != nil {
		return nil, fmt.Errorf("pointClouds.attach: %w", err)
	}
	return json.Marshal(pointCloudInfo(pc))
}

// loadScan reads a scan file and decodes its cloud-local points, returning the raw bytes (for
// embedding as a resource) alongside the points.
func loadScan(fullFileName string) ([]byte, []math.Point3, error) {
	if fullFileName == "" {
		return nil, nil, fmt.Errorf("pointClouds.attach: fullFileName is required")
	}
	data, err := os.ReadFile(fullFileName)
	if err != nil {
		return nil, nil, fmt.Errorf("pointClouds.attach: read %q: %w", fullFileName, err)
	}
	points, err := pointcloud.ReadScan(fullFileName, data)
	if err != nil {
		return nil, nil, err
	}
	return data, points, nil
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

// point3Of / pointOf bridge the wire point type and math.Point3.
func point3Of(p types.Point) math.Point3 {
	return math.P3(math.Scalar(p.X), math.Scalar(p.Y), math.Scalar(p.Z))
}

func pointOf(p math.Point3) types.Point {
	return types.Point{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)}
}
