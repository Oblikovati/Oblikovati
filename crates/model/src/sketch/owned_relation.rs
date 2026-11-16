// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraintId, SketchEntityId, SketchError, SketchRecipe};

/// Persistent lifecycle owner of one structural sketch relation.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchOwnedRelation {
    pub owner: SketchEntityId,
    pub constraint: SketchConstraintId,
}
impl Sketch2D {
    /// Check whether this sketch has entity-owned structural relations.
    /// Example: `if sketch.has_owned_relations() { /* validate its owner */ }`.
    pub fn has_owned_relations(&self) -> bool {
        !self.owned_relations.is_empty()
    }
    pub(crate) fn own_relation(
        &mut self,
        owner: SketchEntityId,
        constraint: SketchConstraintId,
    ) -> Result<(), SketchError> {
        self.entity_kind(owner)?;
        if !self.constraints.iter().any(|row| row.id == constraint)
            || self.owned_relations.contains_key(&constraint)
        {
            return Err(SketchError::Definition(format!(
                "structural relation {constraint:?}: expected an existing unowned constraint"
            )));
        }
        self.owned_relations.insert(constraint, owner);
        Ok(())
    }
    pub(super) fn capture_owned_relations(&self, recipe: &mut SketchRecipe) {
        if self.owned_relations.is_empty() {
            return;
        }
        recipe.owned_relations = Some(
            self.owned_relations
                .iter()
                .map(|(constraint, owner)| SketchOwnedRelation {
                    owner: *owner,
                    constraint: *constraint,
                })
                .collect(),
        );
        self.capture_point_ownership_metadata(recipe);
        recipe.version = 19;
    }
    pub(super) fn restore_owned_relations(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.owned_relations) {
            (1..=18, None) => Ok(()),
            (19..=20, Some(relations)) if !relations.is_empty() => {
                for relation in relations {
                    self.own_relation(relation.owner, relation.constraint)?;
                }
                Ok(())
            }
            shape => Err(SketchError::Definition(format!(
                "structural relation recipe {shape:?}: expected no owners through version 18 or a nonempty owner list in version 19"
            ))),
        }
    }
}
impl Sketch2D {
    pub(super) fn require_corner_connectivity_after_removing(
        &self,
        removed: &std::collections::BTreeSet<SketchConstraintId>,
        removed_entities: &[SketchEntityId],
    ) -> Result<(), SketchError> {
        let owners: std::collections::BTreeSet<_> =
            self.owned_relations.values().copied().collect();
        for owner in owners {
            if removed_entities.contains(&owner)
                || self.corner_source_is_removed(owner, removed_entities)
            {
                continue;
            }
            let points: Vec<_> = self
                .constraints
                .iter()
                .filter_map(|row| {
                    if self.owned_relations.get(&row.id) != Some(&owner) {
                        return None;
                    }
                    match row.relation {
                        super::PlanarConstraint::PointOnLine { point, .. } => Some(point),
                        _ => None,
                    }
                })
                .collect();
            if let [first, second] = points.as_slice()
                && !self.coincidence_path(*first, *second, removed)
            {
                return Err(SketchError::Definition(format!(
                    "corner owner {owner:?}, removed relations {removed:?}: expected its virtual corners to remain coincident"
                )));
            }
        }
        Ok(())
    }
    fn corner_source_is_removed(&self, owner: SketchEntityId, removed: &[SketchEntityId]) -> bool {
        self.constraints.iter().any(|row| {
            self.owned_relations.get(&row.id) == Some(&owner)
                && matches!(row.relation, super::PlanarConstraint::PointOnLine { line, .. } if removed.contains(&line))
        })
    }
    /// Recognize a recorded coincidence without applying a positional tolerance.
    /// Example: `sketch.points_coincident(first, second)?` before editing a shared corner.
    pub fn points_coincident(
        &self,
        first: SketchEntityId,
        second: SketchEntityId,
    ) -> Result<bool, SketchError> {
        let first_position = self.point(first)?.position;
        let second_position = self.point(second)?.position;
        Ok(first_position == second_position
            || self.coincidence_path(first, second, &std::collections::BTreeSet::new()))
    }
    pub(crate) fn coincidence_path(
        &self,
        first: SketchEntityId,
        second: SketchEntityId,
        removed: &std::collections::BTreeSet<SketchConstraintId>,
    ) -> bool {
        let mut seen = std::collections::BTreeSet::from([first]);
        loop {
            if seen.contains(&second) {
                return true;
            }
            let count = seen.len();
            for row in &self.constraints {
                if removed.contains(&row.id) {
                    continue;
                }
                if let super::PlanarConstraint::Coincident { points } = row.relation
                    && points.iter().any(|point| seen.contains(point))
                {
                    seen.extend(points);
                }
            }
            if seen.len() == count {
                return false;
            }
        }
    }
}
