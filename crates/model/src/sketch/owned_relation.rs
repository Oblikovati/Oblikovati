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
            (19, Some(relations)) if !relations.is_empty() => {
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
