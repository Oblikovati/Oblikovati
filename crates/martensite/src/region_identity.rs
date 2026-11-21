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
    let outer = source_uses(&region.outer().uses);
    let holes = region
        .holes()
        .iter()
        .map(|hole| source_uses(&hole.uses))
        .collect();
    equivalent_region_boundary(&SketchPrismSelection { outer, holes }, selection)
}

fn source_uses(uses: &[LoopUse]) -> Vec<SketchDirectedCurve> {
    uses.iter()
        .map(|item| SketchDirectedCurve {
            curve: oblikovati_model::sketch::SketchEntityId(item.curve.0),
            reversed: item.reversed,
        })
        .collect()
}

/// Compare complete source-boundary identity independent of cyclic start and traversal direction.
/// Example: `equivalent_region_boundary(&saved, &current)` before binding a transient region.
pub fn equivalent_region_boundary(
    actual: &SketchPrismSelection,
    expected: &SketchPrismSelection,
) -> bool {
    if !same_loop(&actual.outer, &expected.outer) || actual.holes.len() != expected.holes.len() {
        return false;
    }
    let mut used = vec![false; expected.holes.len()];
    for hole in &actual.holes {
        let Some(index) = expected
            .holes
            .iter()
            .enumerate()
            .position(|(index, candidate)| !used[index] && same_loop(hole, candidate))
        else {
            return false;
        };
        used[index] = true;
    }
    true
}

fn same_loop(actual: &[SketchDirectedCurve], expected: &[SketchDirectedCurve]) -> bool {
    if actual.is_empty() || actual.len() != expected.len() {
        return false;
    }
    (0..actual.len()).any(|start| {
        same_traversal(actual, expected, start, false)
            || same_traversal(actual, expected, start, true)
    })
}

fn same_traversal(
    actual: &[SketchDirectedCurve],
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
        edge.curve == wanted.curve && (edge.reversed ^ reverse) == wanted.reversed
    })
}

#[cfg(test)]
#[path = "../tests/unit/region_identity.rs"]
mod tests;
