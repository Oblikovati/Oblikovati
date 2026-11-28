// SPDX-License-Identifier: GPL-2.0-only
//! B-rep surface sewing via `BRepBuilderAPISewing` with edge-condition classification (#3918).
use super::thicken_offset_chain::{collect_shapes, ensure_positive_solid};
use super::*;
use martensite_tkbrep::{
    brep::BRepTool,
    brep_adaptor::BRepAdaptorCurve,
    topexp,
    topods::{Builder, Edge, Shape, Shell, face, shell},
    toptools::{IndexedDataMapOfShapeListOfShape, IndexedMapOfShape},
};
use martensite_tkernel::message::ProgressRange;
use martensite_tkg3d::{adaptor3d::Adaptor3dCurve, top_abs::ShapeEnum};
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::{
    brep_builder_api::{BRepBuilderAPIMakeSolid, BRepBuilderAPISewing},
    brep_gprop::BRepGProp,
};
use oblikovati_model::feature::{FeatureId, stitch::ResolvedStitch};

/// One remaining free boundary edge after B-rep surface sewing.
#[derive(Clone, Debug, PartialEq)]
pub struct StitchFreeEdgeReport {
    pub index: usize,
    pub length_mm: f64,
    pub max_gap_mm: Option<f64>,
    pub polyline: Vec<[f64; 3]>,
}

/// Edge-condition classification produced by `BRepBuilderAPISewing`.
#[derive(Clone, Debug, PartialEq)]
pub struct StitchAnalysisReport {
    pub stitched_edge_count: usize,
    pub free_edges: Vec<StitchFreeEdgeReport>,
    pub near_tangent_edges: Vec<Vec<[f64; 3]>>,
    pub stitched_polylines: Vec<Vec<[f64; 3]>>,
    pub closed_shell: bool,
}

/// Analyze edge conditions and sew selected surface bodies with `BRepBuilderAPISewing`.
/// Example: `analyze_stitch_surfaces(&[srf1, srf2], history, guid, 0.05)?`.
pub fn analyze_stitch_surfaces(
    surfaces: &[&NamedSolid],
    history: &DocumentEditHistory,
    document: DocumentGuid,
    tolerance_cm: f64,
) -> Result<StitchAnalysisReport, String> {
    let shapes = collect_input_shapes(surfaces, history, document)?;
    let (_, report) = sew_surface_shapes(&shapes, tolerance_cm)?;
    Ok(report)
}

/// Sew selected surface bodies and build a named surface shell (or closed solid when requested).
pub(in crate::prism::named) fn evaluate_stitch_result(
    surfaces: &[&NamedSolid],
    history: &DocumentEditHistory,
    document: DocumentGuid,
    feature: FeatureId,
    input: &ResolvedStitch,
) -> Result<NamedSolid, String> {
    if surfaces.len() < 2 {
        return Err(format!(
            "stitch surfaces count {}: expected at least two surface bodies",
            surfaces.len()
        ));
    }
    let resolution = surfaces.first().and_then(|s| body_resolution(s));
    let shapes = collect_input_shapes(surfaces, history, document)?;
    let (sewed, report) = sew_surface_shapes(&shapes, input.maximum_tolerance_cm)?;

    if !input.maintain_as_surface
        && report.closed_shell
        && let Some(solid) = try_make_closed_solid(&sewed)
    {
        let topology =
            BlendNamedTopology::from_derived(history, document, feature, solid, resolution)?;
        return Ok(NamedSolid::Derived(Box::new(topology)));
    }

    let shell_shape = normalize_to_shell(&sewed)?;
    let topology =
        BlendNamedTopology::from_stitch_shell(history, document, feature, shell_shape, resolution)?;
    Ok(NamedSolid::Derived(Box::new(topology)))
}

fn body_resolution(body: &NamedSolid) -> Option<PrismResolution> {
    match body {
        NamedSolid::Prism(prism) => prism.certification.as_ref().map(|c| c.resolution),
        NamedSolid::Derived(derived) => derived.resolution(),
    }
}

fn collect_input_shapes(
    surfaces: &[&NamedSolid],
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<Vec<Shape>, String> {
    let mut shapes = Vec::with_capacity(surfaces.len());
    for surface in surfaces {
        let native = surface.native_shape(history, document)?;
        let copied = crate::checked_shape::copy_shape(native, "stitch surface input")
            .map_err(|cause| cause.to_string())?
            .shape()
            .map_err(|cause| cause.to_string())?
            .clone();
        shapes.push(copied);
    }
    Ok(shapes)
}

/// Execute `BRepBuilderAPISewing` on isolated surface copies and classify stitched/free/near-tangent edges.
pub(crate) fn sew_surface_shapes(
    shapes: &[Shape],
    tolerance_cm: f64,
) -> Result<(Shape, StitchAnalysisReport), String> {
    if !tolerance_cm.is_finite() || tolerance_cm <= 0.0 {
        return Err(format!(
            "stitch tolerance {tolerance_cm}: expected a finite positive tolerance in cm"
        ));
    }
    let mut sewing = BRepBuilderAPISewing::new(tolerance_cm.max(1.0e-7), true, true, true, false);
    for shape in shapes {
        sewing
            .add(shape)
            .map_err(|cause| format!("stitch add surface: {cause}"))?;
    }
    sewing
        .perform(&ProgressRange::default())
        .map_err(|cause| format!("stitch sewing perform: {cause}"))?;
    let sewed = sewing.sewed_shape().clone();
    if sewed.is_null() {
        return Err("stitch sewing produced a null shape".into());
    }
    let faces = collect_shapes(&sewed, ShapeEnum::Face)?;
    if faces.is_empty() {
        return Err("stitch sewing produced zero faces".into());
    }
    let report = classify_sewed_edges(&sewing, &sewed)?;
    Ok((sewed, report))
}

fn classify_sewed_edges(
    sewing: &BRepBuilderAPISewing,
    sewed: &Shape,
) -> Result<StitchAnalysisReport, String> {
    let mut edge_faces = IndexedDataMapOfShapeListOfShape::default();
    topexp::map_shapes_and_ancestors(sewed, ShapeEnum::Edge, ShapeEnum::Face, &mut edge_faces)
        .map_err(|cause| cause.to_string())?;

    let mut stitched_polylines = Vec::new();
    let mut near_tangent_edges = Vec::new();
    let mut raw_free_edges: Vec<(Edge, f64, Vec<[f64; 3]>)> = Vec::new();

    for i in 1..=edge_faces.extent() {
        let Ok(edge_shape) = edge_faces.find_key(i) else {
            continue;
        };
        let Ok(edge) = martensite_tkbrep::topods::edge(edge_shape) else {
            continue;
        };
        if BRepTool::degenerated(&edge).unwrap_or(true) {
            continue;
        }
        let Ok(faces) = edge_faces.find_from_index(i) else {
            continue;
        };
        let distinct_faces = count_distinct_shapes(faces);
        let polyline = sample_edge_polyline(&edge);
        if distinct_faces >= 2 {
            if is_near_tangent_pair(&edge, faces) {
                near_tangent_edges.push(polyline.clone());
            }
            stitched_polylines.push(polyline);
        } else if distinct_faces == 1 {
            // Skip cylindrical/periodic seam edges (appear twice on one face).
            let is_seam = faces
                .first()
                .and_then(|f| face(f).ok())
                .and_then(|f| BRepTool::is_closed_on_face(&edge, &f).ok())
                .unwrap_or(false);
            if !is_seam {
                let len_mm = edge_length_mm(&edge);
                raw_free_edges.push((edge, len_mm, polyline));
            }
        }
    }

    let stitched_edge_count = stitched_polylines
        .len()
        .max(usize::try_from(sewing.nb_contigous_edges().max(0)).unwrap_or(0));

    let free_edges = build_free_edge_reports(&raw_free_edges);
    let closed_shell = free_edges.is_empty()
        && matches!(sewed.shape_type(), Ok(ShapeEnum::Shell | ShapeEnum::Solid));
    Ok(StitchAnalysisReport {
        stitched_edge_count,
        free_edges,
        near_tangent_edges,
        stitched_polylines,
        closed_shell,
    })
}

fn count_distinct_shapes(list: &martensite_tkernel::ncollection::List<Shape>) -> usize {
    let mut map = IndexedMapOfShape::default();
    for shape in list.iter() {
        map.add(shape.clone());
    }
    map.extent()
}

fn edge_length_mm(edge: &Edge) -> f64 {
    let mut props = GProps::default();
    if BRepGProp::linear_properties(edge.as_ref(), &mut props, false, false).is_ok() {
        let len = props.mass();
        if len.is_finite() && len >= 0.0 {
            return len * 10.0;
        }
    }
    0.0
}

fn sample_edge_polyline(edge: &Edge) -> Vec<[f64; 3]> {
    let Ok(curve) = BRepAdaptorCurve::with_edge(edge) else {
        return Vec::new();
    };
    let (Ok(first), Ok(last)) = (curve.first_parameter(), curve.last_parameter()) else {
        return Vec::new();
    };
    if !first.is_finite() || !last.is_finite() {
        return Vec::new();
    }
    let steps = 16usize;
    let mut pts = Vec::with_capacity(steps + 1);
    for idx in 0..=steps {
        let t = first + (last - first) * (idx as f64 / steps as f64);
        if let Ok(p) = curve.value(t) {
            pts.push([p.x(), p.y(), p.z()]);
        }
    }
    pts
}

fn build_free_edge_reports(raw: &[(Edge, f64, Vec<[f64; 3]>)]) -> Vec<StitchFreeEdgeReport> {
    raw.iter()
        .enumerate()
        .map(|(idx, (_, length_mm, polyline))| {
            let max_gap_mm = nearest_other_free_edge_gap_mm(idx, polyline, raw);
            StitchFreeEdgeReport {
                index: idx + 1,
                length_mm: *length_mm,
                max_gap_mm,
                polyline: polyline.clone(),
            }
        })
        .collect()
}

fn nearest_other_free_edge_gap_mm(
    current: usize,
    polyline: &[[f64; 3]],
    all: &[(Edge, f64, Vec<[f64; 3]>)],
) -> Option<f64> {
    let mid = polyline.get(polyline.len() / 2)?;
    let mut best = f64::INFINITY;
    for (idx, (_, _, other)) in all.iter().enumerate() {
        if idx == current || other.is_empty() {
            continue;
        }
        if let Some(other_mid) = other.get(other.len() / 2) {
            let dx = mid[0] - other_mid[0];
            let dy = mid[1] - other_mid[1];
            let dz = mid[2] - other_mid[2];
            let dist = (dx * dx + dy * dy + dz * dz).sqrt();
            if dist < best {
                best = dist;
            }
        }
    }
    best.is_finite().then_some(best * 10.0)
}

fn is_near_tangent_pair(edge: &Edge, faces: &martensite_tkernel::ncollection::List<Shape>) -> bool {
    let mut unique = IndexedMapOfShape::default();
    for shape in faces.iter() {
        unique.add(shape.clone());
    }
    if unique.extent() < 2 {
        return false;
    }
    let (Ok(f1), Ok(f2)) = (
        unique.find_key(1).and_then(face),
        unique.find_key(2).and_then(face),
    ) else {
        return false;
    };
    let (Some(n1), Some(n2)) = (face_mid_normal(edge, &f1), face_mid_normal(edge, &f2)) else {
        return false;
    };
    let dot = (n1[0] * n2[0] + n1[1] * n2[1] + n1[2] * n2[2]).clamp(-1.0, 1.0);
    let angle_deg = dot.abs().acos().to_degrees();
    // Near-tangent: smoothly aligned within 15 degrees (but not 90-degree sharp corners).
    angle_deg > 0.1 && angle_deg <= 15.0
}

fn face_mid_normal(edge: &Edge, face: &martensite_tkbrep::topods::Face) -> Option<[f64; 3]> {
    let pcurve = BRepTool::curve_on_face(edge, face).ok()?;
    let mid_t = 0.5 * (pcurve.first + pcurve.last);
    let uv = pcurve.pcurve.get()?.value(mid_t).ok()?;
    let surf = BRepTool::surface(face).ok()?;
    let geom = surf.get()?;
    let d1 = geom.d1(uv.x(), uv.y()).ok()?;
    let (du, dv) = (d1.d1u, d1.d1v);
    let n = [
        du.y() * dv.z() - du.z() * dv.y(),
        du.z() * dv.x() - du.x() * dv.z(),
        du.x() * dv.y() - du.y() * dv.x(),
    ];
    let mag = (n[0] * n[0] + n[1] * n[1] + n[2] * n[2]).sqrt();
    (mag > 1.0e-12).then_some([n[0] / mag, n[1] / mag, n[2] / mag])
}

fn try_make_closed_solid(sewed: &Shape) -> Option<Shape> {
    if matches!(sewed.shape_type(), Ok(ShapeEnum::Solid)) {
        return ensure_positive_solid(sewed.clone(), "stitch solid").ok();
    }
    let sh = shell(sewed).ok()?;
    let maker = BRepBuilderAPIMakeSolid::from_shell(&sh).ok()?;
    let solid = maker.solid().ok()?.into_shape();
    ensure_positive_solid(solid, "stitch solid").ok()
}

fn normalize_to_shell(sewed: &Shape) -> Result<Shape, String> {
    if matches!(sewed.shape_type(), Ok(ShapeEnum::Shell)) {
        return Ok(sewed.clone());
    }
    let faces = collect_shapes(sewed, ShapeEnum::Face)?;
    let mut out = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut out);
    let mut shape = out.into_shape();
    for face in &faces {
        builder
            .add(&mut shape, face)
            .map_err(|cause| cause.to_string())?;
    }
    Ok(shape)
}
