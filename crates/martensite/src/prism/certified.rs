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
    if !matches!(resolution.cap, ToleranceCap::Refuse(_)) {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!(
                "extrusion tolerance cap {:?}: expected an explicit refusal limit",
                resolution.cap
            ),
        ));
    }
    let regions = build_planar_regions(profile, resolution)
        .map_err(|cause| Phase::Profile.operation(Entity::Body, cause))?;
    let region = crate::resolve_region_by_boundary(&regions, selection).map_err(|cause| {
        Phase::Profile.refused(Kind::InvalidReference, Entity::Body, cause.to_string())
    })?;
    let result = extrude(region, vector, &ExtrudeOptions { resolution })
        .map_err(|cause| Phase::Prism.operation(Entity::Body, cause))?;
    retain_result(&result)
}

fn retain_result(result: &OperationResult) -> Result<CertifiedPrismSnapshot, EvaluationFailure> {
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
    let body = snapshot::capture_prism(result.solid(), &lineage)?;
    let achieved = result.achieved_tolerance();
    Ok(CertifiedPrismSnapshot {
        body,
        achieved_tolerance: PrismAchievedTolerance {
            vertex: achieved.vertex,
            edge: achieved.edge,
            face: achieved.face,
        },
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
