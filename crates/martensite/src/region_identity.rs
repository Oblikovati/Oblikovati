// SPDX-License-Identifier: GPL-2.0-only
//! Resolve saved boundary lineage against the kernel's current region arrangement.
use martensite_tkbo::profile_extrusion::{LoopUse, PlanarRegion, PlanarRegionSet, RegionId};
use oblikovati_model::feature::sketch_prism::{SketchDirectedCurve, SketchPrismSelection};
use std::{error::Error, fmt};

/// A saved region boundary did not resolve to exactly one current region.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum RegionSelectionError {
    Missing {
        selection: SketchPrismSelection,
    },
    Ambiguous {
        selection: SketchPrismSelection,
        regions: Vec<RegionId>,
    },
}

impl fmt::Display for RegionSelectionError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Missing { selection } => write!(
                output,
                "region boundary {selection:?}: expected one current region, found none"
            ),
            Self::Ambiguous { selection, regions } => write!(
                output,
                "region boundary {selection:?}: expected one current region, found {regions:?}"
            ),
        }
    }
}
impl Error for RegionSelectionError {}

/// Resolve authored boundary lineage without persisting an arrangement index.
/// Example: `resolve_region_by_boundary(&regions, &selection)?` after each recompute.
pub fn resolve_region_by_boundary<'a>(
    regions: &'a PlanarRegionSet,
    selection: &SketchPrismSelection,
) -> Result<&'a PlanarRegion, RegionSelectionError> {
    let matches: Vec<_> = regions
        .regions()
        .iter()
        .filter(|region| matches_region(region, selection))
        .collect();
    match matches.as_slice() {
        [region] => Ok(*region),
        [] => Err(RegionSelectionError::Missing {
            selection: selection.clone(),
        }),
        _ => Err(RegionSelectionError::Ambiguous {
            selection: selection.clone(),
            regions: matches.iter().map(|region| region.id()).collect(),
        }),
    }
}

fn matches_region(region: &PlanarRegion, selection: &SketchPrismSelection) -> bool {
    if !same_loop(&region.outer().uses, &selection.outer)
        || region.holes().len() != selection.holes.len()
    {
        return false;
    }
    let mut used = vec![false; selection.holes.len()];
    for hole in region.holes() {
        let Some(index) = selection
            .holes
            .iter()
            .enumerate()
            .position(|(index, candidate)| !used[index] && same_loop(&hole.uses, candidate))
        else {
            return false;
        };
        used[index] = true;
    }
    true
}

fn same_loop(actual: &[LoopUse], expected: &[SketchDirectedCurve]) -> bool {
    if actual.is_empty() || actual.len() != expected.len() {
        return false;
    }
    (0..actual.len()).any(|start| {
        same_traversal(actual, expected, start, false)
            || same_traversal(actual, expected, start, true)
    })
}

fn same_traversal(
    actual: &[LoopUse],
    expected: &[SketchDirectedCurve],
    start: usize,
    reverse: bool,
) -> bool {
    expected.iter().enumerate().all(|(offset, wanted)| {
        let index = if reverse {
            (start + actual.len() - offset) % actual.len()
        } else {
            (start + offset) % actual.len()
        };
        let edge = actual[index];
        edge.curve.0 == wanted.curve.0 && (edge.reversed ^ reverse) == wanted.reversed
    })
}

#[cfg(test)]
#[path = "../tests/unit/region_identity.rs"]
mod tests;
