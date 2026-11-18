// SPDX-License-Identifier: GPL-2.0-only
//! Detached cylindrical cut and exact Boolean provenance for Hole (#4093).
use super::*;
use martensite_tkbo::brep_algo_api::BRepAlgoAPICut;
use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::gp::{Axis2, Direction3, Point3};
use martensite_tkprim::brep_prim_api::{BRepPrimAPIMakeCylinder, BRepPrimAPIMakeOneAxis};
use martensite_tktopalgo::brep_gprop::BRepGProp;
use oblikovati_model::identity::DerivedReferenceName;
mod tolerances;

pub(in crate::prism::named) struct HoleRequest<'a> {
    pub feature: FeatureId,
    pub face: &'a [u8],
    pub axis: [[f64; 3]; 2],
    pub diameter: f64,
    pub depth: f64,
}

impl NamedPrismSnapshot {
    pub(in crate::prism::named) fn hole_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &HoleRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let source = self.blend_sources(history, document)?;
        let body = self.body_reference_key(history, document)?.to_vec();
        build_hole_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            request,
        )
    }
}

pub(super) fn build_hole_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    request: &HoleRequest<'_>,
) -> Result<BlendCandidate, String> {
    let HoleRequest {
        feature,
        face: face_key,
        axis,
        diameter,
        depth,
    } = *request;
    if !diameter.is_finite() || diameter <= 0. || !depth.is_finite() || depth <= 0. {
        return Err(format!(
            "hole diameter {diameter}, depth {depth}: expected finite positive lengths"
        ));
    }
    let mut copier = crate::checked_shape::copy_shape(input.body, "hole input")
        .map_err(|error| error.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let target = copier.shape().map_err(|error| error.to_string())?.clone();
    let mut maker = cylinder(axis, diameter, depth)?;
    let tool = maker.shape().map_err(|error| error.to_string())?.clone();
    let tool_faces = tool_faces(&mut maker, document, feature, face_key)?;
    let cut = BRepAlgoAPICut::from_shapes(&target, &tool)
        .map_err(|error| format!("hole cut: expected completed Boolean operation: {error}"))?;
    let (shape, achieved_tolerance) = checked_cut(&target, &cut, input.resolution)?;
    let mut records = cut_history(&cut, copied, false)?;
    records.extend(cut_history(&cut, tool_faces, true)?);
    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: Some(achieved_tolerance),
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        document,
        scope: history.evaluation_scope(),
    })
}

fn cylinder(
    axis: [[f64; 3]; 2],
    diameter: f64,
    depth: f64,
) -> Result<BRepPrimAPIMakeCylinder, String> {
    let [origin, vector] = axis;
    let direction =
        Direction3::new(vector[0], vector[1], vector[2]).map_err(|error| error.to_string())?;
    let frame = Axis2::new(Point3::new(origin[0], origin[1], origin[2]), direction)
        .map_err(|error| error.to_string())?;
    Ok(BRepPrimAPIMakeCylinder::with_axes(
        frame,
        diameter / 2.,
        depth,
    ))
}

fn tool_faces(
    maker: &mut BRepPrimAPIMakeCylinder,
    document: DocumentGuid,
    feature: FeatureId,
    parent: &[u8],
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let axis = maker.one_axis();
    let faces = [
        ("hole-wall", axis.lateral_face()),
        ("hole-end", axis.top_face()),
        ("hole-start", axis.bottom_face()),
    ];
    faces
        .into_iter()
        .map(|(role, face)| {
            let name = DerivedReferenceName::new(document, feature.0, 1, role, &[parent.to_vec()])?
                .encode()?;
            Ok((
                name,
                face.map_err(|error| format!("hole {role}: expected a tool face: {error}"))?
                    .into(),
            ))
        })
        .collect()
}

fn checked_cut(
    target: &Shape,
    cut: &BRepAlgoAPICut,
    resolution: Option<PrismResolution>,
) -> Result<(Shape, PrismAchievedTolerance), String> {
    if !cut.is_done() || cut.has_errors() || cut.has_warnings() {
        return Err(format!(
            "hole cut: expected a completed Boolean without errors or warnings (done={}, errors={}, warnings={})",
            cut.is_done(),
            cut.has_errors(),
            cut.has_warnings()
        ));
    }
    let result = single_solid(cut.shape())?;
    crate::checked_shape::require_geometry(&result, "hole result")
        .map_err(|error| error.to_string())?;
    validate_result_faces(&result)?;
    let achieved = tolerances::check(target, &result, resolution)?;
    let before = crate::checked_shape::analytic_properties(target, "hole input")
        .map_err(|error| error.to_string())?;
    let after = crate::checked_shape::analytic_properties(&result, "hole result")
        .map_err(|error| error.to_string())?;
    if after.volume <= 0. || after.volume >= before.volume {
        return Err(format!(
            "hole volume {} from {}: expected a nonempty reduced body",
            after.volume, before.volume
        ));
    }
    Ok((result, achieved))
}

fn validate_result_faces(result: &Shape) -> Result<(), String> {
    let faces = entities(result, ShapeEnum::Face)?;
    if faces.is_empty() {
        return Err("hole result faces []: expected nonempty solid boundaries".into());
    }
    for face in faces {
        let mut properties = GProps::new();
        BRepGProp::surface_properties(&face, &mut properties, false, false)
            .map_err(|error| format!("hole result face: expected analytic area: {error}"))?;
        let area = properties.mass();
        let loops = entities(&face, ShapeEnum::Wire)?.len();
        if !area.is_finite() || area <= 0. || loops == 0 {
            return Err(format!(
                "hole result face area {area}, loops {loops}: expected positive finite area and closed loops"
            ));
        }
    }
    Ok(())
}

fn single_solid(result: &Shape) -> Result<Shape, String> {
    let solids = entities(result, ShapeEnum::Solid)?;
    if solids.len() != 1 {
        return Err(format!(
            "hole result has {} solids: expected exactly one",
            solids.len()
        ));
    }
    let solid = solids[0].clone();
    for kind in [ShapeEnum::Face, ShapeEnum::Edge, ShapeEnum::Vertex] {
        let all = entities(result, kind)?;
        let retained = entities(&solid, kind)?;
        if all.len() != retained.len()
            || all
                .iter()
                .any(|item| !retained.iter().any(|other| item.is_same(other)))
        {
            return Err(format!(
                "hole result {kind:?}: expected all result entities in the single solid"
            ));
        }
    }
    Ok(solid)
}
fn entities(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, String> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut map).map_err(|error| error.to_string())?;
    (1..=map.extent())
        .map(|index| {
            map.find_key(index)
                .cloned()
                .map_err(|error| error.to_string())
        })
        .collect()
}

fn cut_history(
    cut: &BRepAlgoAPICut,
    sources: Vec<(Vec<u8>, Shape)>,
    transient: bool,
) -> Result<Vec<BlendHistoryRecord>, String> {
    sources
        .into_iter()
        .map(|(name, source)| {
            Ok(BlendHistoryRecord {
                generated: cut.generated(&source).map_err(|error| error.to_string())?,
                modified: cut.modified(&source).map_err(|error| error.to_string())?,
                deleted: cut.is_deleted(&source).map_err(|error| error.to_string())?,
                source,
                name,
                transient,
            })
        })
        .collect()
}
