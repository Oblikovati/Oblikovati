// SPDX-License-Identifier: GPL-2.0-only
//! Create and inspect extended topology-aware Work Plane definitions (#3918).

use super::*;
use oblikovati_model::feature::work_plane::{
    WorkPlaneEdgeReference, WorkPlaneExtendedDefinition, WorkPlaneFaceReference,
    WorkPlaneReference, WorkPlaneSurfaceReference, WorkPlaneTopologyRef, WorkPlaneVertexReference,
};
use std::boxed::Box as StdBox;

/// Normalize user-facing angular expressions such as `"45 degrees"` to `"45 deg"`.
pub fn normalize_angle_expression(raw: &str) -> String {
    let trimmed = raw.trim();
    if let Some(stem) = trimmed.strip_suffix("degrees") {
        return format!("{}deg", stem);
    }
    if let Some(stem) = trimmed.strip_suffix("degree") {
        return format!("{}deg", stem);
    }
    if let Some(stem) = trimmed.strip_suffix('°') {
        return format!("{} deg", stem.trim_end());
    }
    trimmed.to_owned()
}

pub(in crate::datums) fn constructor_kind(definition: &WorkPlaneExtendedDefinition) -> String {
    use WorkPlaneExtendedDefinition::*;
    match definition {
        MidplaneTwoPlanes { .. } => WORK_PLANE_TWO_PLANES.0.into_owned(),
        ThreePoints { .. } => WORK_PLANE_THREE_POINTS.0.into_owned(),
        ParallelThroughPoint { .. } => WORK_PLANE_PLANE_AND_POINT.0.into_owned(),
        NormalToAxisThroughPoint { .. } => WORK_PLANE_NORMAL_TO_CURVE.0.into_owned(),
        AngleToPlaneAroundEdge { .. } => WORK_PLANE_LINE_PLANE_ANGLE.0.into_owned(),
        TwoCoplanarEdges { .. } => WORK_PLANE_TWO_LINES.0.into_owned(),
        MidplaneOfTorus { .. } => WORK_PLANE_TORUS_MID_PLANE.0.into_owned(),
        TangentThroughEdge { .. } => WORK_PLANE_LINE_AND_TANGENT.0.into_owned(),
        TangentThroughPoint { .. } => WORK_PLANE_POINT_AND_TANGENT.0.into_owned(),
        TangentParallelToPlane { .. } => WORK_PLANE_PLANE_AND_TANGENT.0.into_owned(),
    }
}

pub(in crate::datums) fn slots(definition: &WorkPlaneExtendedDefinition) -> Vec<WorkPlaneRefSlot> {
    use WorkPlaneExtendedDefinition::*;
    match definition {
        MidplaneTwoPlanes { .. } => {
            reference_only::slots(&[("First plane", "plane"), ("Second plane", "plane")])
        }
        ThreePoints { .. } => three_points::slots(),
        ParallelThroughPoint { .. } => parallel::slots(),
        NormalToAxisThroughPoint { .. } => {
            axis_point::AxisPointPlaneConstructor::NormalToAxis.slots()
        }
        AngleToPlaneAroundEdge { .. } => line_angle::slots(),
        TwoCoplanarEdges { .. } => two_axes::slots(),
        MidplaneOfTorus { .. } => reference_only::slots(&[("Torus", "face")]),
        TangentThroughEdge { .. } => {
            reference_only::slots(&[("Surface", "face"), ("Edge", "axis")])
        }
        TangentThroughPoint { .. } => {
            reference_only::slots(&[("Surface", "face"), ("Point", "point")])
        }
        TangentParallelToPlane { .. } => {
            reference_only::slots(&[("Surface", "face"), ("Plane", "plane")])
        }
    }
}

pub(in crate::datums) fn create_midplane(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let refs = request.refs.as_deref().unwrap_or_default();
    let [first, second] = refs else {
        return Err(Error(format!(
            "midplane references {refs:?}: expected exactly two plane references"
        )));
    };
    let first = plane_or_face_reference(snapshot, first)?;
    let second = plane_or_face_reference(snapshot, second)?;
    Ok(PreparedGeometry::Definition(
        WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
            WorkPlaneExtendedDefinition::MidplaneTwoPlanes { first, second },
        )),
    ))
}

pub(in crate::datums) fn has_topology_ref(args: &CreateWorkPlaneArgs) -> bool {
    let refs = args.refs.as_deref().unwrap_or_default();
    let is_topological = |value: &str| {
        !value.is_empty() && !value.starts_with("origin/") && !value.starts_with("datum/")
    };
    refs.iter().any(|value| is_topological(value))
        || matches!(
            args.kind.as_str(),
            k if k == WORK_PLANE_TORUS_MID_PLANE.0
                || k == WORK_PLANE_LINE_AND_TANGENT.0
                || k == WORK_PLANE_POINT_AND_TANGENT.0
                || k == WORK_PLANE_PLANE_AND_TANGENT.0
        )
}

pub(in crate::datums) fn prepare_topology_plane(
    args: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedDatum, Error> {
    let refs = args.refs.as_deref().unwrap_or_default();
    let definition = match args.kind.as_str() {
        k if k == WORK_PLANE_OFFSET.0 => {
            let [reference] = refs else {
                return Err(Error(format!(
                    "offset references {refs:?}: expected one plane or face reference"
                )));
            };
            let base = plane_or_face_reference(snapshot, reference)?;
            scalars::validate_offset(snapshot, &args.offset)?;
            PreparedGeometry::Offset {
                base,
                expression: args.offset.clone(),
            }
        }
        k if k == WORK_PLANE_TWO_PLANES.0 => create_midplane(args, snapshot)?,
        k if k == WORK_PLANE_THREE_POINTS.0 => {
            let [p0, p1, p2] = refs else {
                return Err(Error(format!(
                    "three-point references {refs:?}: expected three point or vertex references"
                )));
            };
            let points = [
                vertex_reference(snapshot, p0)?,
                vertex_reference(snapshot, p1)?,
                vertex_reference(snapshot, p2)?,
            ];
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::ThreePoints { points },
            )))
        }
        k if k == WORK_PLANE_PLANE_AND_POINT.0 => {
            let [plane, point] = refs else {
                return Err(Error(format!(
                    "parallel-through-point references {refs:?}: expected plane and point"
                )));
            };
            let base = plane_or_face_reference(snapshot, plane)?;
            let point = vertex_reference(snapshot, point)?;
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::ParallelThroughPoint { base, point },
            )))
        }
        k if k == WORK_PLANE_NORMAL_TO_CURVE.0 => {
            let [axis, point] = refs else {
                return Err(Error(format!(
                    "normal-to-axis references {refs:?}: expected axis/edge and point"
                )));
            };
            let axis = edge_reference(snapshot, axis)?;
            let point = vertex_reference(snapshot, point)?;
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::NormalToAxisThroughPoint { axis, point },
            )))
        }
        k if k == WORK_PLANE_LINE_PLANE_ANGLE.0 => {
            let [line, plane] = refs else {
                return Err(Error(format!(
                    "angle-around-edge references {refs:?}: expected edge/axis and plane"
                )));
            };
            let edge = edge_reference(snapshot, line)?;
            let base = plane_or_face_reference(snapshot, plane)?;
            let normalized = normalize_angle_expression(&args.angle);
            scalars::OwnedDatumScalar::Angle.validate(snapshot, &normalized)?;
            PreparedGeometry::ExtendedAngle {
                base,
                edge,
                expression: normalized,
            }
        }
        k if k == WORK_PLANE_TWO_LINES.0 => {
            let [first, second] = refs else {
                return Err(Error(format!(
                    "two-coplanar-edges references {refs:?}: expected two edges or axes"
                )));
            };
            let first = edge_reference(snapshot, first)?;
            let second = edge_reference(snapshot, second)?;
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::TwoCoplanarEdges { first, second },
            )))
        }
        k if k == WORK_PLANE_TORUS_MID_PLANE.0 => {
            let [face] = refs else {
                return Err(Error(format!(
                    "torus midplane references {refs:?}: expected one torus face"
                )));
            };
            let surface = surface_reference(face)?;
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::MidplaneOfTorus { surface },
            )))
        }
        k if k == WORK_PLANE_LINE_AND_TANGENT.0 => {
            let [surface, edge] = refs else {
                return Err(Error(format!(
                    "tangent-through-edge references {refs:?}: expected surface and edge"
                )));
            };
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::TangentThroughEdge {
                    surface: surface_reference(surface)?,
                    edge: edge_reference(snapshot, edge)?,
                },
            )))
        }
        k if k == WORK_PLANE_POINT_AND_TANGENT.0 => {
            let [surface, point] = refs else {
                return Err(Error(format!(
                    "tangent-through-point references {refs:?}: expected surface and point"
                )));
            };
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::TangentThroughPoint {
                    surface: surface_reference(surface)?,
                    point: vertex_reference(snapshot, point)?,
                },
            )))
        }
        k if k == WORK_PLANE_PLANE_AND_TANGENT.0 => {
            let [surface, plane] = refs else {
                return Err(Error(format!(
                    "tangent-parallel references {refs:?}: expected surface and plane"
                )));
            };
            PreparedGeometry::Definition(WorkGeometryDefinition::WorkPlaneExtended(StdBox::new(
                WorkPlaneExtendedDefinition::TangentParallelToPlane {
                    surface: surface_reference(surface)?,
                    plane: plane_or_face_reference(snapshot, plane)?,
                },
            )))
        }
        other => {
            return Err(Error(format!(
                "topology work plane constructor {other:?}: unsupported constructor"
            )));
        }
    };
    Ok(PreparedDatum {
        definition,
        collection: DatumCollection::Planes,
        construction: args.construction,
        visible: args.visible.as_deref().copied().unwrap_or(true),
    })
}

fn plane_or_face_reference(
    snapshot: &DocumentSnapshot,
    value: &str,
) -> Result<WorkPlaneReference, Error> {
    if value.starts_with("origin/") || value.starts_with("datum/") {
        return inputs::plane_reference(snapshot, value);
    }
    let key = crate::reference_bytes::hex_bytes(value)?;
    let face = WorkPlaneFaceReference::new(key).map_err(Error)?;
    Ok(WorkPlaneReference::Face(face))
}

fn vertex_reference(
    snapshot: &DocumentSnapshot,
    value: &str,
) -> Result<WorkPlaneVertexReference, Error> {
    if value == WORK_REF_CENTER {
        return Ok(WorkPlaneVertexReference::Origin);
    }
    if value.starts_with("datum/") {
        let id = reference::decode_reference(snapshot, value)?;
        let definition = snapshot
            .work_geometry
            .get(id)
            .map_err(api_error)?
            .definition();
        if !DatumCollection::Points.contains(definition) {
            return Err(Error(format!(
                "point reference {value:?}: expected a work point"
            )));
        }
        return Ok(WorkPlaneVertexReference::WorkPoint(id));
    }
    let key = crate::reference_bytes::hex_bytes(value)?;
    let topo = WorkPlaneTopologyRef::vertex(key).map_err(Error)?;
    Ok(WorkPlaneVertexReference::TopologyVertex(topo))
}

fn edge_reference(
    snapshot: &DocumentSnapshot,
    value: &str,
) -> Result<WorkPlaneEdgeReference, Error> {
    match value {
        WORK_REF_X_AXIS => return Ok(WorkPlaneEdgeReference::OriginX),
        WORK_REF_Y_AXIS => return Ok(WorkPlaneEdgeReference::OriginY),
        WORK_REF_Z_AXIS => return Ok(WorkPlaneEdgeReference::OriginZ),
        _ => {}
    }
    if value.starts_with("datum/") {
        let id = reference::decode_reference(snapshot, value)?;
        let definition = snapshot
            .work_geometry
            .get(id)
            .map_err(api_error)?
            .definition();
        if !DatumCollection::Axes.contains(definition) {
            return Err(Error(format!(
                "axis reference {value:?}: expected a work axis"
            )));
        }
        return Ok(WorkPlaneEdgeReference::WorkAxis(id));
    }
    let key = crate::reference_bytes::hex_bytes(value)?;
    let topo = WorkPlaneTopologyRef::edge(key).map_err(Error)?;
    Ok(WorkPlaneEdgeReference::TopologyEdge(topo))
}

fn surface_reference(value: &str) -> Result<WorkPlaneSurfaceReference, Error> {
    let key = crate::reference_bytes::hex_bytes(value)?;
    let face = WorkPlaneFaceReference::new(key).map_err(Error)?;
    Ok(WorkPlaneSurfaceReference { face })
}
