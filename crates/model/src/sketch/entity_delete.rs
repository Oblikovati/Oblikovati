// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraint, SketchEntityId, SketchError, entity_geometry};
use std::collections::{BTreeMap, BTreeSet};
mod retain;
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
        let retired = self.deletion_closure(selected);
        let mut candidate = self.clone();
        candidate.retain_surviving_geometry(&retired);
        let constraints = candidate.remove_dangling_relations()?;
        let entities = self.retired_entity_ids(&retired);
        let validated = Self::from_recipe(&candidate.recipe())?;
        *self = validated;
        Ok(SketchEntityDeletion {
            entities,
            constraints,
        })
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
    fn deletion_closure(&self, selected: &[SketchEntityId]) -> BTreeSet<SketchEntityId> {
        let edges: BTreeMap<SketchEntityId, Vec<SketchEntityId>> = entity_geometry::catalogue(self)
            .into_iter()
            .map(|(id, shape)| (id, shape.ownership_dependencies()))
            .collect();
        let mut retired: BTreeSet<SketchEntityId> = selected.iter().copied().collect();
        extend_dependent_entities(&edges, &mut retired);
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
        retired
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
            .map(|(id, dimensional)| self.remove_authored_relation(id, dimensional))
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
