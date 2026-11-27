// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSnapshot, regions::resolved_regions};
use crate::PrismResolutionPolicy;
use martensite_tkbrep::{
    brep::BRepTool,
    brep_adaptor::BRepAdaptorCurve,
    brep_tools::{WireExplorer, outer_wire},
    topexp,
    topods::{self, Shape},
};
use martensite_tkg3d::{
    adaptor3d::Adaptor3dCurve,
    top_abs::{Orientation, ShapeEnum},
};
use martensite_tkmath::geom_abs::{CurveType, JoinType};
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIMakeOffset;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::sketch::SketchPlane;

/// Offset the certified outer boundary and author a polyline witness for each result wire.
/// Example: `offset_sketch_profile(snapshot, id, policy, 0, 0.1, 8)?`.
pub fn offset_sketch_profile(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    policy: PrismResolutionPolicy,
    profile_index: usize,
    distance: f64,
    arc_segments: u32,
) -> Result<Vec<Vec<[f64; 2]>>, String> {
    if !distance.is_finite() || arc_segments == 0 {
        return Err(format!(
            "profile offset {distance} with {arc_segments} spans: expected a finite distance and positive span count"
        ));
    }
    let resolved = snapshot
        .require_converged_resolved_sketch(sketch)
        .map_err(|cause| cause.to_string())?;
    let plane = resolved.plane;
    let regions = resolved_regions(resolved, policy).map_err(|cause| cause.to_string())?;
    let region = regions
        .as_ref()
        .and_then(|regions| regions.regions().get(profile_index))
        .ok_or_else(|| format!("profile {profile_index}: expected a closed sketch region"))?;
    let source = outer_wire(region.face()).map_err(|cause| cause.to_string())?;
    if distance == 0. {
        return polylines(source.as_ref(), plane, arc_segments);
    }
    let mut offset = BRepOffsetAPIMakeOffset::from_wire(&source, JoinType::Arc, false);
    offset
        .perform(distance, 0.)
        .map_err(|cause| cause.to_string())?;
    if !offset.is_done() {
        return Err(format!(
            "profile {profile_index} offset {distance}: expected a completed closed contour"
        ));
    }
    let shape = offset.shape().map_err(|cause| cause.to_string())?;
    polylines(shape, plane, arc_segments)
}

fn polylines(
    shape: &Shape,
    plane: SketchPlane,
    arc_segments: u32,
) -> Result<Vec<Vec<[f64; 2]>>, String> {
    let mut explorer = topexp::Explorer::new(shape, ShapeEnum::Wire, ShapeEnum::Shape)
        .map_err(|cause| cause.to_string())?;
    let mut loops = Vec::new();
    while explorer.more() {
        let wire = topods::wire(explorer.current()).map_err(|cause| cause.to_string())?;
        if !BRepTool::is_closed(&wire).map_err(|cause| cause.to_string())? {
            return Err(
                "sketch profile offset produced an open wire: expected a closed contour".into(),
            );
        }
        let mut edges = WireExplorer::with_wire(&wire).map_err(|cause| cause.to_string())?;
        let mut points = Vec::new();
        while edges.more() {
            sample_edge(&edges, plane, arc_segments, &mut points)?;
            edges.next().map_err(|cause| cause.to_string())?;
        }
        if points.len() < 3 {
            return Err(format!(
                "offset wire has {} samples: expected a closed contour with at least three vertices",
                points.len()
            ));
        }
        loops.push(points);
        explorer.next().map_err(|cause| cause.to_string())?;
    }
    if loops.is_empty() {
        return Err("offset profile collapsed: expected at least one closed contour".into());
    }
    Ok(loops)
}

fn sample_edge(
    explorer: &WireExplorer,
    plane: SketchPlane,
    arc_segments: u32,
    points: &mut Vec<[f64; 2]>,
) -> Result<(), String> {
    let edge = topods::edge(explorer.current()).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())?;
    let spans = if curve.get_type().map_err(|cause| cause.to_string())? == CurveType::Line {
        1
    } else if BRepTool::is_closed(&edge).map_err(|cause| cause.to_string())? {
        arc_segments.max(3)
    } else {
        arc_segments
    };
    let mut ends = [
        curve.first_parameter().map_err(|cause| cause.to_string())?,
        curve.last_parameter().map_err(|cause| cause.to_string())?,
    ];
    if edge.orientation() == Orientation::Reversed {
        ends.reverse();
    }
    for index in 0..spans {
        let parameter = ends[0] + (ends[1] - ends[0]) * f64::from(index) / f64::from(spans);
        let point = curve.value(parameter).map_err(|cause| cause.to_string())?;
        points.push(
            plane
                .to_sketch(*point.coordinates())
                .map_err(|cause| cause.to_string())?,
        );
    }
    Ok(())
}
