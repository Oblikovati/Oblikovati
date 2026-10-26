// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeSet;
pub(super) fn validate_definition(
    definition: &PrismProbeDefinition,
) -> Result<(), EvaluationFailure> {
    let vector = definition.displacement;
    if vector.iter().any(|value| !value.is_finite()) || vector[2] == 0. {
        return Err(Phase::Input.refused(Kind::InvalidInput, Entity::Body, format!(
            "prism displacement {vector:?}: expected finite components with nonzero profile-normal displacement"
        )));
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
) -> Result<(), EvaluationFailure> {
    match profile {
        PrismProbeLoop::Polygon {
            edge_keys,
            vertices,
        } => {
            if vertices.len() < 3 || vertices.len() != edge_keys.len() {
                return Err(Phase::Input.refused(Kind::InvalidInput, Entity::Loop(index), format!(
                    "prism polygon loop {index} vertices={}, edge keys={}: expected at least three vertices and one key per segment",
                    vertices.len(), edge_keys.len()
                )));
            }
            require_finite_vertices(vertices, index)?;
        }
        PrismProbeLoop::Circle { center, radius, .. } => {
            if !center.iter().all(|value| value.is_finite()) || !radius.is_finite() || *radius <= 0.
            {
                return Err(Phase::Input.refused(Kind::InvalidInput, Entity::Loop(index), format!(
                    "prism circle loop {index} center {center:?}, radius {radius:?}: expected finite center and positive finite radius"
                )));
            }
        }
    }
    validate_keys(&profile.edge_keys(), keys)
}
fn validate_keys(edge_keys: &[u64], keys: &mut BTreeSet<u64>) -> Result<(), EvaluationFailure> {
    for key in edge_keys {
        if *key == 0 || !keys.insert(*key) {
            return Err(Phase::Input.refused(
                Kind::InvalidInput,
                Entity::Edge(*key),
                format!(
                    "prism edge key {key}: expected unique nonzero authored keys across all loops"
                ),
            ));
        }
    }
    Ok(())
}

fn require_finite_vertices(vertices: &[[f64; 2]], index: usize) -> Result<(), EvaluationFailure> {
    if vertices.iter().flatten().all(|value| value.is_finite()) {
        return Ok(());
    }
    Err(Phase::Input.refused(
        Kind::InvalidInput,
        Entity::Loop(index),
        format!(
            "prism loop {index} {:?}: expected finite XY coordinates",
            vertices
        ),
    ))
}
