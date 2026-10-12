// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeSet;
pub(super) fn validate_definition(definition: &PrismProbeDefinition) -> Result<(), String> {
    let vector = definition.displacement;
    if vector.iter().any(|value| !value.is_finite()) || vector[2] == 0. {
        return Err(format!(
            "prism displacement {vector:?}: expected finite components with nonzero XY-normal displacement"
        ));
    }
    let mut keys = BTreeSet::new();
    for (index, profile) in std::iter::once(&definition.outer)
        .chain(&definition.holes)
        .enumerate()
    {
        validate_loop(profile, index, &mut keys)?;
    }
    Ok(())
}
fn validate_loop(
    profile: &PrismProbeLoop,
    index: usize,
    keys: &mut BTreeSet<u64>,
) -> Result<(), String> {
    if profile.vertices.len() < 3 || profile.vertices.len() != profile.edge_keys.len() {
        return Err(format!(
            "prism loop {index} vertices={}, edge keys={}: expected at least three vertices and one key per segment",
            profile.vertices.len(),
            profile.edge_keys.len()
        ));
    }
    require_finite_vertices(profile, index)?;
    validate_keys(&profile.edge_keys, keys)
}
fn validate_keys(edge_keys: &[u64], keys: &mut BTreeSet<u64>) -> Result<(), String> {
    for key in edge_keys {
        if *key == 0 || !keys.insert(*key) {
            return Err(format!(
                "prism edge key {key}: expected unique nonzero authored keys across all loops"
            ));
        }
    }
    Ok(())
}

fn require_finite_vertices(profile: &PrismProbeLoop, index: usize) -> Result<(), String> {
    if profile
        .vertices
        .iter()
        .flatten()
        .all(|value| value.is_finite())
    {
        return Ok(());
    }
    Err(format!(
        "prism loop {index} {:?}: expected finite XY coordinates",
        profile.vertices
    ))
}
