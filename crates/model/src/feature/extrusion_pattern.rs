// SPDX-License-Identifier: GPL-2.0-only
use super::{FeatureId, move_body::ResolvedBodyMotion};
use std::collections::BTreeSet;

/// Model-space placement addressed by signed offsets from the authored seed cell.
#[derive(Clone, Copy, Debug)]
pub struct ExtrusionPatternOccurrence {
    pub offset: [i64; 2],
    pub motion: ResolvedBodyMotion,
}

/// A resolved extrusion tool and its operation, borrowed from the authored source feature.
pub struct PatternPrismSource<'a> {
    pub source: FeatureId,
    pub prism: super::sketch_prism::ResolvedSketchPrism<'a>,
    pub operation: super::sketch_prism::PrismBoolean,
}

pub(super) fn validate_sources(sources: &[FeatureId]) -> Result<(), String> {
    let unique: BTreeSet<_> = sources.iter().copied().collect();
    if unique.is_empty() || unique.len() != sources.len() || unique.contains(&FeatureId(0)) {
        return Err(format!(
            "pattern sources {:?}: expected distinct feature identities",
            sources
        ));
    }
    Ok(())
}
