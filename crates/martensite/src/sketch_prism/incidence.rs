// SPDX-License-Identifier: GPL-2.0-only
//! Authored point coincidence, without coordinate-based welding or geometry changes.
use super::*;
use oblikovati_model::sketch::{PlanarConstraint, Sketch2D, SketchEntityId, SketchPoint2D};
use std::collections::BTreeMap;

pub(super) struct CoincidentPoints {
    roots: BTreeMap<SketchEntityId, SketchEntityId>,
    groups: BTreeMap<SketchEntityId, Vec<SketchPoint2D>>,
}
impl CoincidentPoints {
    pub(super) fn new(sketch: &Sketch2D) -> Self {
        let mut parents: BTreeMap<_, _> = sketch
            .points()
            .iter()
            .map(|point| (point.id, point.id))
            .collect();
        for constraint in sketch.constraints() {
            if let PlanarConstraint::Coincident { points: [a, b] } = constraint.relation {
                let first = root(&parents, a);
                let second = root(&parents, b);
                parents.insert(first.max(second), first.min(second));
            }
        }
        let roots: BTreeMap<_, _> = parents
            .keys()
            .map(|id| (*id, root(&parents, *id)))
            .collect();
        let mut groups: BTreeMap<SketchEntityId, Vec<SketchPoint2D>> = BTreeMap::new();
        for point in sketch.points() {
            groups.entry(roots[&point.id]).or_default().push(*point);
        }
        for points in groups.values_mut() {
            points.sort_by_key(|point| point.id);
        }
        Self { roots, groups }
    }
    pub(super) fn vertex(&self, id: SketchEntityId) -> Result<SketchPoint2D, EvaluationFailure> {
        let root = self.roots[&id];
        let points = &self.groups[&root];
        let source = points
            .iter()
            .find(|point| point.id == root)
            .expect("root is an authored point");
        for point in points {
            if point.position.map(f64::to_bits) != source.position.map(f64::to_bits) {
                return Err(Phase::Input.refused(Kind::InvalidGeometry, Entity::SketchEntity(point.id.0), format!(
                    "coincident points {:?} at {:?} and {:?} at {:?}: expected identical solved coordinates to preserve both curves",
                    root, source.position, point.id, point.position
                )));
            }
        }
        Ok(*source)
    }
    pub(super) fn sources(&self, id: SketchEntityId) -> Vec<u64> {
        self.groups[&id].iter().map(|point| point.id.0).collect()
    }
}
fn root(
    parents: &BTreeMap<SketchEntityId, SketchEntityId>,
    mut id: SketchEntityId,
) -> SketchEntityId {
    while parents[&id] != id {
        id = parents[&id];
    }
    id
}

#[cfg(test)]
#[path = "../../tests/unit/sketch_incidence.rs"]
mod tests;
