// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraint, SketchEntityId, SketchError, entity_geometry};
use std::collections::{BTreeMap, BTreeSet};
mod retain;
mod selection;
pub use selection::SketchDeletionSelection;
/// Retired identities and complete relations for the document's parameter/history cascade.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct SketchEntityDeletion {
    pub entities: Vec<SketchEntityId>,
    pub constraints: Vec<SketchConstraint>,
}
impl Sketch2D {
    /// Delete a checked selection and its dependent geometry atomically; example: `let retired = sketch.delete_entities(&[parent])?;`.
    pub fn delete_entities(
        &mut self,
        selected: &[SketchEntityId],
    ) -> Result<SketchEntityDeletion, SketchError> {
        if selected.is_empty() {
            return Ok(SketchEntityDeletion::default());
        }
        self.validate_deletion_selection(selected)?;
        let retired = self.deletion_closure(selected)?;
        self.require_pattern_members_survive(&retired)?;
        let mut candidate = self.clone();
        candidate.retain_surviving_geometry(&retired);
        let mut constraints = candidate.remove_owned_relations(&retired)?;
        constraints.extend(candidate.remove_dangling_relations()?);
        let entities = self.retired_entity_ids(&retired);
        let validated = Self::from_recipe(&candidate.recipe())?;
        *self = validated;
        Ok(SketchEntityDeletion {
            entities,
            constraints,
        })
    }
    fn require_pattern_members_survive(
        &self,
        retired: &BTreeSet<SketchEntityId>,
    ) -> Result<(), SketchError> {
        for pattern in &self.rectangular_patterns {
            if pattern
                .definition
                .seed
                .iter()
                .chain(pattern.members.iter().flat_map(|member| &member.entities))
                .any(|id| retired.contains(id))
            {
                return Err(SketchError::Definition(format!(
                    "delete entities {retired:?}: expected the owning pattern {:?} to be removed first",
                    pattern.id
                )));
            }
        }
        Ok(())
    }
    fn retired_entity_ids(&self, retired: &BTreeSet<SketchEntityId>) -> Vec<SketchEntityId> {
        self.owned_entity_ids()
            .into_iter()
            .filter(|id| retired.contains(id))
            .collect()
    }
    fn validate_deletion_selection(&self, selected: &[SketchEntityId]) -> Result<(), SketchError> {
        for id in selected {
            self.entity_kind(*id)?;
        }
        for constraint in &self.constraints {
            self.validate_constraint_relation(&constraint.relation)?;
        }
        Ok(())
    }
    fn deletion_closure(
        &self,
        selected: &[SketchEntityId],
    ) -> Result<BTreeSet<SketchEntityId>, SketchError> {
        let mut edges: BTreeMap<SketchEntityId, Vec<SketchEntityId>> =
            entity_geometry::catalogue(self)
                .into_iter()
                .map(|(id, shape)| (id, shape.ownership_dependencies()))
                .collect();
        for handle in self.spline_handles() {
            edges.entry(handle.end()).or_default().push(handle.parent());
        }
        let mut retired: BTreeSet<SketchEntityId> = selected.iter().copied().collect();
        self.extend_owned_dependencies(&edges, &mut retired)?;
        let live: BTreeSet<SketchEntityId> = edges
            .iter()
            .filter(|(id, _)| !retired.contains(id))
            .flat_map(|(_, dependencies)| dependencies.iter().copied())
            .collect();
        retired.extend(
            self.defining_points
                .iter()
                .filter(|id| !live.contains(id))
                .copied(),
        );
        Ok(retired)
    }
    fn extend_owned_dependencies(
        &self,
        edges: &BTreeMap<SketchEntityId, Vec<SketchEntityId>>,
        retired: &mut BTreeSet<SketchEntityId>,
    ) -> Result<(), SketchError> {
        loop {
            let previous = retired.len();
            extend_dependent_entities(edges, retired);
            let mut candidate = self.clone();
            candidate.retain_surviving_geometry(retired);
            for (constraint, owner) in &self.owned_relations {
                let relation = self
                    .constraints
                    .iter()
                    .find(|row| row.id == *constraint)
                    .ok_or_else(|| {
                        SketchError::Definition(format!(
                            "owned relation {constraint:?}: expected an existing relation"
                        ))
                    })?;
                if candidate
                    .validate_constraint_relation(&relation.relation)
                    .is_err()
                {
                    retired.insert(*owner);
                }
            }
            if retired.len() == previous {
                return Ok(());
            }
        }
    }
    fn remove_owned_relations(
        &mut self,
        retired: &BTreeSet<SketchEntityId>,
    ) -> Result<Vec<SketchConstraint>, SketchError> {
        let ids: Vec<_> = self
            .owned_relations
            .iter()
            .filter(|(_, owner)| retired.contains(owner))
            .map(|(id, _)| *id)
            .collect();
        let mut removed = Vec::new();
        for id in ids {
            self.owned_relations.remove(&id);
            let relation = self
                .constraints
                .iter()
                .find(|row| row.id == id)
                .ok_or_else(|| {
                    SketchError::Definition(format!(
                        "owned relation {id:?}: expected an existing relation"
                    ))
                })?;
            let dimensional =
                super::dependencies::dimension_parameter(&relation.relation).is_some();
            removed.push(self.remove_authored_relation(id, dimensional)?);
        }
        Ok(removed)
    }
    fn remove_dangling_relations(&mut self) -> Result<Vec<SketchConstraint>, SketchError> {
        // The seed was validated and geometry values did not change. Only missing ownership can invalidate a relation here.
        let ids: Vec<_> = self
            .constraints
            .iter()
            .filter(|constraint| {
                self.validate_constraint_relation(&constraint.relation)
                    .is_err()
            })
            .map(|constraint| {
                (
                    constraint.id,
                    super::dependencies::dimension_parameter(&constraint.relation).is_some(),
                )
            })
            .collect();
        ids.into_iter()
            .map(|(id, dimensional)| {
                self.owned_relations.remove(&id);
                self.remove_authored_relation(id, dimensional)
            })
            .collect()
    }
}
fn extend_dependent_entities(
    edges: &BTreeMap<SketchEntityId, Vec<SketchEntityId>>,
    retired: &mut BTreeSet<SketchEntityId>,
) {
    loop {
        let previous = retired.len();
        let dependents: Vec<SketchEntityId> = edges
            .iter()
            .filter(|(_, dependencies)| {
                dependencies
                    .iter()
                    .any(|dependency| retired.contains(dependency))
            })
            .map(|(id, _)| *id)
            .collect();
        retired.extend(dependents);
        if retired.len() == previous {
            return;
        }
    }
}
