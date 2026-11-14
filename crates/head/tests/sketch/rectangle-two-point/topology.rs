// SPDX-License-Identifier: GPL-2.0-only
//! Independent structural check for the two-point rectangle's loop.
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use std::collections::{BTreeMap, BTreeSet};

pub(super) fn is_axis_aligned_closed(geometry: &Sketch2D) -> bool {
    if geometry.lines().len() != 4 {
        return false;
    }
    // Go aa8297d1 model/sketch/composite.go shares corner identities to form one profile.
    let mut degrees = BTreeMap::<SketchEntityId, usize>::new();
    let mut edges = BTreeSet::new();
    let mut horizontal_edges = 0;
    for line in geometry.lines() {
        let [a, b] = line.endpoints;
        let Some(horizontal) = horizontal_axis(geometry, a, b) else {
            return false;
        };
        if a == b || !edges.insert(ordered_edge(a, b)) {
            return false;
        }
        horizontal_edges += usize::from(horizontal);
        *degrees.entry(a).or_default() += 1;
        *degrees.entry(b).or_default() += 1;
    }
    horizontal_edges == 2 && degrees.len() == 4 && degrees.values().all(|degree| *degree == 2)
}
fn horizontal_axis(geometry: &Sketch2D, a: SketchEntityId, b: SketchEntityId) -> Option<bool> {
    let (Ok(a), Ok(b)) = (geometry.point(a), geometry.point(b)) else {
        return None;
    };
    let horizontal = a.position[1] == b.position[1];
    let vertical = a.position[0] == b.position[0];
    (horizontal != vertical).then_some(horizontal)
}

fn ordered_edge(a: SketchEntityId, b: SketchEntityId) -> [SketchEntityId; 2] {
    if a < b { [a, b] } else { [b, a] }
}
#[cfg(test)]
#[path = "topology_tests.rs"]
mod tests;
