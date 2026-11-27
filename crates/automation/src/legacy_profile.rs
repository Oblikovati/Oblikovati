// SPDX-License-Identifier: GPL-2.0-only
//! Verify Go profile boundary witnesses against current certified Martensite regions.
use oblikovati_martensite::{
    PrismResolutionPolicy, certified_sketch_regions_with_policy, equivalent_region_boundary,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId, feature::sketch_prism::SketchPrismSelection,
    history::DocumentSnapshot,
};
use oblikovati_persistence::package::{
    LegacyCapStageVerifier, LegacyProfileResolver, LegacySketchCandidate,
};

/// Supply independently established source boundaries for a Go profile index.
pub trait LegacyProfileOrderWitness {
    /// Resolve one source profile or decline when its Go ordering is not established.
    /// Example: `witness.boundary(candidate, sketch, 0)?`.
    fn boundary(
        &self,
        candidate: &LegacySketchCandidate,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String>;
}

/// A source witness checked against one current qualified sketch and its exact regions.
pub struct WitnessedLegacyProfileResolver<'a> {
    base: &'a DocumentSnapshot,
    policy: PrismResolutionPolicy,
    qualifier: &'a dyn LegacyCapStageVerifier,
    witness: &'a dyn LegacyProfileOrderWitness,
}
impl<'a> WitnessedLegacyProfileResolver<'a> {
    /// Bind the source snapshot, tolerance policy and independent evidence providers.
    /// Example: `WitnessedLegacyProfileResolver::new(&base, policy, qualifier, witness)`.
    pub fn new(
        base: &'a DocumentSnapshot,
        policy: PrismResolutionPolicy,
        qualifier: &'a dyn LegacyCapStageVerifier,
        witness: &'a dyn LegacyProfileOrderWitness,
    ) -> Self {
        Self {
            base,
            policy,
            qualifier,
            witness,
        }
    }
}
impl LegacyProfileResolver for WitnessedLegacyProfileResolver<'_> {
    fn resolve(
        &self,
        candidate: &LegacySketchCandidate,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String> {
        let snapshot = candidate
            .prepare_qualified_profile(self.base, sketch, self.qualifier)
            .map_err(|e| e.to_string())?;
        let expected = self.witness.boundary(candidate, sketch, index)?;
        let regions = certified_sketch_regions_with_policy(&snapshot, sketch, self.policy)
            .map_err(|e| e.to_string())?;
        let geometry = snapshot
            .sketches
            .get(sketch)
            .map_err(|e| e.to_string())?
            .geometry();
        select_matching_region(&regions, geometry, &expected, sketch, index)
    }
}

fn select_matching_region(
    regions: &[oblikovati_martensite::SketchRegionBoundary],
    geometry: &oblikovati_model::sketch::Sketch2D,
    expected: &SketchPrismSelection,
    sketch: DocumentSketchId,
    index: usize,
) -> Result<SketchPrismSelection, String> {
    let matches = regions
        .iter()
        .map(|region| region.line_selection(geometry).map_err(|e| e.to_string()))
        .collect::<Result<Vec<_>, _>>()?
        .into_iter()
        .filter(|region| equivalent_region_boundary(region, expected))
        .collect::<Vec<_>>();
    match matches.as_slice() {
        [region] => Ok(region.clone()),
        _ => Err(format!(
            "Go profile {index}, sketch {sketch:?}: expected exactly one certified region for witnessed boundary; found {}",
            matches.len()
        )),
    }
}

#[cfg(test)]
mod tests;
