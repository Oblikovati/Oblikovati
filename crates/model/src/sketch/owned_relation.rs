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
    /// Find dimension sources connected to an entity-owned corner through sketch relations.
    /// Example: `sketch.corner_related_parameters()?` before accepting an edit.
    pub fn corner_related_parameters(&self) -> Result<Vec<u64>, SketchError> {
        let mut reached = std::collections::BTreeSet::new();
        for row in &self.constraints {
            if self.owned_relations.contains_key(&row.id) {
                reached.extend(self.relation_columns(&row.relation)?);
            }
        }
        loop {
            let count = reached.len();
            for row in &self.constraints {
                let columns = self.relation_columns(&row.relation)?;
                if columns.iter().any(|column| reached.contains(column)) {
                    reached.extend(columns);
                }
            }
            for arc in &self.arcs {
                let columns = self.arc_columns(arc)?;
                if columns.iter().any(|column| reached.contains(column)) {
                    reached.extend(columns);
                }
            }
            if reached.len() == count {
                break;
            }
        }
        let mut parameters = Vec::new();
        for row in &self.constraints {
            let connected = self.owned_relations.contains_key(&row.id)
                || self
                    .relation_columns(&row.relation)?
                    .iter()
                    .any(|column| reached.contains(column));
            if connected
                && let Some(parameter) = super::dependencies::dimension_parameter(&row.relation)
                && !parameters.contains(&parameter)
            {
                parameters.push(parameter);
            }
        }
        Ok(parameters)
    }
    fn relation_columns(
        &self,
        relation: &super::PlanarConstraint,
    ) -> Result<Vec<usize>, SketchError> {
        let (_, columns) = super::circular_compile::relation_projection(self, relation, None)?;
        Ok(columns)
    }
    /// Check whether a constraint belongs to an entity.
    /// Example: `sketch.owns_relation(id)` before classifying a solve failure.
    pub fn owns_relation(&self, id: SketchConstraintId) -> bool {
        self.owned_relations.contains_key(&id)
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
                "structural relation recipe {shape:?}: expected no owners through version 18 or a nonempty owner list in versions 19 or 20"
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
