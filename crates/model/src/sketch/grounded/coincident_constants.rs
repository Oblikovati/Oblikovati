// SPDX-License-Identifier: GPL-2.0-only
//! Freeze coordinates implied by coincident relations and consistent fixed sources.
use super::{Sketch2D, SketchError, SketchPoint2D};
use crate::sketch::{PlanarConstraint, SketchEntityId, frozen::CoordinateTargets};
use std::collections::{BTreeMap, BTreeSet};

pub(super) fn propagate(
    sketch: &Sketch2D,
    anchors: &[SketchPoint2D],
    targets: &CoordinateTargets,
    grounded: &BTreeSet<usize>,
) -> Result<Vec<(usize, f64)>, SketchError> {
    let points: BTreeMap<_, _> = sketch
        .points()
        .iter()
        .enumerate()
        .map(|(index, point)| (point.id, index))
        .collect();
    let mut parents: Vec<usize> = (0..points.len()).collect();
    for constraint in sketch.constraints() {
        if let PlanarConstraint::Coincident {
            points: [first, second],
        } = constraint.relation
        {
            join(&points, &mut parents, first, second)?;
        }
    }
    let mut groups: BTreeMap<(usize, usize), Option<f64>> = BTreeMap::new();
    for (&column, &value) in targets {
        merge_target(&mut groups, (root(&parents, column / 2), column % 2), value);
    }
    for anchor in anchors {
        let point = point_index(&points, anchor.id)?;
        for axis in 0..2 {
            merge_target(
                &mut groups,
                (root(&parents, point), axis),
                Some(anchor.position[axis]),
            );
        }
    }
    let mut result = Vec::new();
    for column in 0..points.len() * 2 {
        if grounded.contains(&column) {
            continue;
        }
        let target = groups
            .get(&(root(&parents, column / 2), column % 2))
            .copied()
            .flatten()
            .or_else(|| targets.get(&column).copied().flatten());
        if let Some(value) = target {
            result.push((column, value));
        }
    }
    Ok(result)
}
fn join(
    points: &BTreeMap<SketchEntityId, usize>,
    parents: &mut [usize],
    first: SketchEntityId,
    second: SketchEntityId,
) -> Result<(), SketchError> {
    let first = root(parents, point_index(points, first)?);
    let second = root(parents, point_index(points, second)?);
    parents[first.max(second)] = first.min(second);
    Ok(())
}
fn point_index(
    points: &BTreeMap<SketchEntityId, usize>,
    id: SketchEntityId,
) -> Result<usize, SketchError> {
    points.get(&id).copied().ok_or_else(|| {
        SketchError::Definition(format!(
            "coincident point {id:?}: expected an owned sketch point"
        ))
    })
}
fn root(parents: &[usize], mut point: usize) -> usize {
    while parents[point] != point {
        point = parents[point];
    }
    point
}
fn merge_target(
    groups: &mut BTreeMap<(usize, usize), Option<f64>>,
    key: (usize, usize),
    value: Option<f64>,
) {
    groups
        .entry(key)
        .and_modify(|previous| {
            if previous.map(f64::to_bits) != value.map(f64::to_bits) {
                *previous = None;
            }
        })
        .or_insert(value);
}
#[cfg(test)]
#[path = "../../../tests/sketch/coincident_constants.rs"]
mod tests;
