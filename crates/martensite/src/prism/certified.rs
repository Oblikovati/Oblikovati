// SPDX-License-Identifier: GPL-2.0-only
//! Released region and extrusion operations, retaining measured tolerance and source lineage.
use super::*;
use martensite_tkbo::profile_extrusion::{
    ExtrudeOptions, HistoryRole, HistorySource, OperationResult, PlanarRegion, PlanarRegionSet,
    Resolution, ResultEntity, SketchProfile, ToleranceCap, build_planar_regions, extrude,
};
use oblikovati_model::feature::sketch_prism::SketchPrismSelection;
use std::collections::BTreeSet;

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

/// Extrude distinct selected regions of one planar sketch using a shared arrangement.
/// Returns one certified body per supplied boundary, in selection order. Boundary
/// lineage resolves each region and the entire batch must succeed before anything
/// is returned. Example: `retain_certified_profile_prisms(&profile, &selections, vector, resolution)?`.
pub fn retain_certified_profile_prisms(
    profile: &SketchProfile,
    selections: &[SketchPrismSelection],
    vector: [f64; 3],
    resolution: Resolution,
) -> Result<Vec<CertifiedPrismSnapshot>, EvaluationFailure> {
    retain_certified_tapered_prisms(profile, selections.iter(), vector, resolution, None)
}

pub(crate) fn retain_certified_tapered_prisms<'a>(
    profile: &SketchProfile,
    selections: impl Iterator<Item = &'a SketchPrismSelection>,
    vector: [f64; 3],
    resolution: Resolution,
    taper: Option<PrismTaper>,
) -> Result<Vec<CertifiedPrismSnapshot>, EvaluationFailure> {
    let mut selections = selections.peekable();
    if selections.peek().is_none() {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            "extrusion profiles: expected at least one selected region".into(),
        ));
    }
    let arrangement = CertifiedPrismRegions::new(profile, resolution)?;
    let mut seen = BTreeSet::new();
    let selected = selections
        .map(|selection| {
            let region = arrangement.resolve(selection)?;
            if !seen.insert(region.id()) {
                return Err(Phase::Profile.refused(
                    Kind::InvalidReference,
                    Entity::Body,
                    format!("region boundary {selection:?}: expected a distinct extrusion profile"),
                ));
            }
            Ok(region)
        })
        .collect::<Result<Vec<_>, _>>()?;
    selected
        .into_iter()
        .map(|region| arrangement.retain(region, vector, taper))
        .collect()
}

pub(crate) fn retain_certified_tapered_prism(
    profile: &SketchProfile,
    selection: &SketchPrismSelection,
    vector: [f64; 3],
    resolution: Resolution,
    taper: Option<PrismTaper>,
) -> Result<CertifiedPrismSnapshot, EvaluationFailure> {
    let arrangement = CertifiedPrismRegions::new(profile, resolution)?;
    arrangement.retain(arrangement.resolve(selection)?, vector, taper)
}

struct CertifiedPrismRegions<'a> {
    profile: &'a SketchProfile,
    regions: PlanarRegionSet,
    resolution: Resolution,
    maximum: f64,
}

impl<'a> CertifiedPrismRegions<'a> {
    fn new(profile: &'a SketchProfile, resolution: Resolution) -> Result<Self, EvaluationFailure> {
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
        Ok(Self {
            profile,
            regions,
            resolution,
            maximum,
        })
    }

    fn resolve(
        &self,
        selection: &SketchPrismSelection,
    ) -> Result<&PlanarRegion, EvaluationFailure> {
        crate::resolve_region_by_boundary(&self.regions, selection).map_err(|cause| {
            Phase::Profile.refused(Kind::InvalidReference, Entity::Body, cause.to_string())
        })
    }

    fn retain(
        &self,
        region: &PlanarRegion,
        vector: [f64; 3],
        taper: Option<PrismTaper>,
    ) -> Result<CertifiedPrismSnapshot, EvaluationFailure> {
        let result = extrude(
            region,
            vector,
            &ExtrudeOptions {
                resolution: self.resolution,
            },
        )
        .map_err(|cause| Phase::Prism.operation(Entity::Body, cause))?;
        let mut retained = retain_result(&result, taper, self.maximum)?;
        retained.vertex_sources = self
            .profile
            .vertices
            .iter()
            .map(|vertex| (vertex.id.0, vec![vertex.id.0]))
            .collect();
        Ok(retained)
    }
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
