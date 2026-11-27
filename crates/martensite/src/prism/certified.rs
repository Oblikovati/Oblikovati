// SPDX-License-Identifier: GPL-2.0-only
//! Released region and extrusion operations, retaining measured tolerance and source lineage.
use super::*;
use martensite_tkbo::profile_extrusion::{
    ExtrudeOptions, HistoryRole, HistorySource, OperationResult, Resolution, ResultEntity,
    SketchProfile, ToleranceCap, build_planar_regions, extrude,
};
use oblikovati_model::feature::sketch_prism::SketchPrismSelection;

/// A validated solid and its measured maximum vertex, edge and face tolerances.
pub struct CertifiedPrismSnapshot {
    pub body: PrismProbeSnapshot,
    pub achieved_tolerance: PrismAchievedTolerance,
    /// Complete lower-dimensional kernel lineage, without native handles.
    pub subentities: PrismSubentityHistory,
    pub(super) subentity_shapes: Vec<martensite_tkbrep::topods::Shape>,
    /// Authored point identities represented by each input profile vertex identity.
    pub vertex_sources: std::collections::BTreeMap<u64, Vec<u64>>,
}
/// Maximum measured tolerances on the kernel operation result, in model units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PrismAchievedTolerance {
    pub vertex: f64,
    pub edge: f64,
    pub face: f64,
}

/// Build a region and prism with the caller's explicit resolution and authored boundary.
/// Example: `retain_certified_profile_prism(&profile, &selection, vector, resolution)?`.
pub fn retain_certified_profile_prism(
    profile: &SketchProfile,
    selection: &SketchPrismSelection,
    vector: [f64; 3],
    resolution: Resolution,
) -> Result<CertifiedPrismSnapshot, EvaluationFailure> {
    retain_certified_tapered_prism(profile, selection, vector, resolution, None)
}

pub(crate) fn retain_certified_tapered_prism(
    profile: &SketchProfile,
    selection: &SketchPrismSelection,
    vector: [f64; 3],
    resolution: Resolution,
    taper: Option<PrismTaper>,
) -> Result<CertifiedPrismSnapshot, EvaluationFailure> {
    let ToleranceCap::Refuse(maximum) = resolution.cap else {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!(
                "extrusion tolerance cap {:?}: expected an explicit refusal limit",
                resolution.cap
            ),
        ));
    };
    let regions = build_planar_regions(profile, resolution)
        .map_err(|cause| Phase::Profile.operation(Entity::Body, cause))?;
    let region = crate::resolve_region_by_boundary(&regions, selection).map_err(|cause| {
        Phase::Profile.refused(Kind::InvalidReference, Entity::Body, cause.to_string())
    })?;
    let result = extrude(region, vector, &ExtrudeOptions { resolution })
        .map_err(|cause| Phase::Prism.operation(Entity::Body, cause))?;
    let mut retained = retain_result(&result, taper, maximum)?;
    retained.vertex_sources = profile
        .vertices
        .iter()
        .map(|vertex| (vertex.id.0, vec![vertex.id.0]))
        .collect();
    Ok(retained)
}

fn retain_result(
    result: &OperationResult,
    taper: Option<PrismTaper>,
    maximum: f64,
) -> Result<CertifiedPrismSnapshot, EvaluationFailure> {
    if !result.findings().is_empty() {
        return Err(Phase::Validation.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!(
                "extrusion findings {:?}: expected no unreported defects",
                result.findings()
            ),
        ));
    }
    let lineage = result
        .faces()
        .iter()
        .enumerate()
        .map(|(index, face)| Ok((face_role(result, index)?, face.clone().into())))
        .collect::<Result<Vec<_>, EvaluationFailure>>()?;
    let mut subentities = super::provenance::capture(result)?;
    let sources = super::provenance::source_shapes(result, &subentities)?;
    let (body, subentity_shapes, achieved_tolerance) = if let Some(taper) = taper {
        let tapered = taper.apply(result.solid(), &lineage, &sources)?;
        for (record, shape) in subentities
            .edges
            .iter_mut()
            .chain(&mut subentities.vertices)
            .zip(&tapered.subentities)
        {
            record.point = super::provenance::representative(shape, "taper topology")?;
        }
        let achieved =
            crate::checked_shape::measured_tolerances(&tapered.shape, maximum, "extrusion taper")?;
        let (body, subentity_shapes) = snapshot::capture_prism_with_subentities(
            &tapered.shape,
            &tapered.faces,
            &tapered.subentities,
        )?;
        (body, subentity_shapes, achieved)
    } else {
        let (body, subentity_shapes) =
            snapshot::capture_prism_with_subentities(result.solid(), &lineage, &sources)?;
        let achieved = result.achieved_tolerance();
        (
            body,
            subentity_shapes,
            PrismAchievedTolerance {
                vertex: achieved.vertex,
                edge: achieved.edge,
                face: achieved.face,
            },
        )
    };
    Ok(CertifiedPrismSnapshot {
        body,
        achieved_tolerance,
        subentities,
        subentity_shapes,
        vertex_sources: Default::default(),
    })
}

fn face_role(result: &OperationResult, index: usize) -> Result<PrismFaceRole, EvaluationFailure> {
    let records: Vec<_> = result
        .history()
        .records
        .iter()
        .filter(|record| record.result == ResultEntity::Face(index))
        .copied()
        .collect();
    let record = match records.as_slice() {
        [record] => Some(*record),
        _ => None,
    };
    match record.map(|entry| (entry.source, entry.role)) {
        Some((HistorySource::Region(_), HistoryRole::StartCap)) => Ok(PrismFaceRole::Start),
        Some((HistorySource::Region(_), HistoryRole::EndCap)) => Ok(PrismFaceRole::End),
        Some((HistorySource::Curve(curve), HistoryRole::SideFace)) => Ok(PrismFaceRole::ProfileEdge(curve.0)),
        _ => Err(Phase::History.refused(Kind::IncompleteHistory, Entity::FaceIndex(index),
            format!("extrusion face {index} history {records:?}: expected one cap or source-curve side face"))),
    }
}

#[cfg(test)]
#[path = "../../tests/unit/certified_prism.rs"]
mod tests;
