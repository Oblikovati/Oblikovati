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
    /// List the stable owners of entity-owned structural relations.
    /// Example: `sketch.corner_owners()` before checking each connected component.
    pub fn corner_owners(&self) -> Vec<SketchEntityId> {
        self.owned_relations
            .values()
            .copied()
            .collect::<std::collections::BTreeSet<_>>()
            .into_iter()
            .collect()
    }
    /// Find dimension sources connected to an entity-owned corner through sketch relations.
    /// Example: `sketch.corner_related_parameters(owner)?` before accepting an edit.
    pub fn corner_related_parameters(
        &self,
        owner: SketchEntityId,
    ) -> Result<Vec<u64>, SketchError> {
        let reached = self.corner_related_columns(owner)?;
        let mut parameters = Vec::new();
        for row in &self.constraints {
            if self.reference_dimensions.contains(&row.id) {
                continue;
            }
            let connected = self.owned_relations.get(&row.id) == Some(&owner)
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
        self.append_corner_pattern_parameters(&reached, &mut parameters)?;
        Ok(parameters)
    }
    /// Find point identities connected to a corner's solver variables.
    /// Example: `sketch.corner_related_points(owner)?` before checking projected anchors.
    pub fn corner_related_points(
        &self,
        owner: SketchEntityId,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let reached = self.corner_related_columns(owner)?;
        let mut points = Vec::new();
        for point in &self.points {
            let columns = self
                .point_variables(&[point.id])
                .into_iter()
                .map(|variable| self.variable_column(variable))
                .collect::<Result<Vec<_>, _>>()?;
            if columns.iter().any(|column| reached.contains(column)) {
                points.push(point.id);
            }
        }
        Ok(points)
    }
    pub(super) fn corner_related_columns(
        &self,
        owner: SketchEntityId,
    ) -> Result<std::collections::BTreeSet<usize>, SketchError> {
        let mut reached = std::collections::BTreeSet::new();
        for row in &self.constraints {
            if self.owned_relations.get(&row.id) == Some(&owner) {
                reached.extend(self.relation_columns(&row.relation)?);
            }
        }
        loop {
            let count = reached.len();
            for row in &self.constraints {
                if self.reference_dimensions.contains(&row.id)
                    || matches!(row.relation, super::PlanarConstraint::Ground { .. })
                {
                    continue;
                }
                let columns = self.relation_columns(&row.relation)?;
                if columns.iter().any(|column| reached.contains(column)) {
                    reached.extend(columns);
                }
            }
            for columns in self.corner_implicit_columns()? {
                if columns.iter().any(|column| reached.contains(column)) {
                    reached.extend(columns);
                }
            }
            if reached.len() == count {
                break;
            }
        }
        Ok(reached)
    }
    fn corner_implicit_columns(&self) -> Result<Vec<Vec<usize>>, SketchError> {
        let mut sources = Vec::new();
        for arc in &self.arcs {
            sources.push(self.arc_columns(arc)?);
        }
        for link in &self.pattern_point_links {
            sources.push(super::compile::pattern_point_columns(self, link)?);
        }
        for link in &self.pattern_radius_links {
            sources.push(super::compile::pattern_radius_columns(self, link)?);
        }
        Ok(sources)
    }
    fn append_corner_pattern_parameters(
        &self,
        reached: &std::collections::BTreeSet<usize>,
        parameters: &mut Vec<u64>,
    ) -> Result<(), SketchError> {
        for offset in &self.pattern_offsets {
            let columns = self
                .point_variables(&[offset.member])
                .into_iter()
                .map(|variable| self.variable_column(variable))
                .collect::<Result<Vec<_>, _>>()?;
            if !columns.iter().any(|column| reached.contains(column)) {
                continue;
            }
            for term in &offset.terms {
                if !parameters.contains(&term.parameter) {
                    parameters.push(term.parameter);
                }
            }
        }
        Ok(())
    }
    pub(super) fn relation_columns(
        &self,
        relation: &super::PlanarConstraint,
    ) -> Result<Vec<usize>, SketchError> {
        match relation {
            super::PlanarConstraint::Radius { curve, .. }
            | super::PlanarConstraint::Diameter { curve, .. } => self.radius_columns(*curve),
            super::PlanarConstraint::EqualRadius { curves } => {
                let mut columns = self.radius_columns(curves[0])?;
                columns.extend(self.radius_columns(curves[1])?);
                Ok(columns)
            }
            _ => super::circular_compile::relation_projection(self, relation, None)
                .map(|(_, columns)| columns),
        }
    }
    fn radius_columns(&self, curve: SketchEntityId) -> Result<Vec<usize>, SketchError> {
        use super::circular_frame::CircularRadius;
        let frame = self.circular_frame(curve)?;
        let variables = match frame.radius {
            CircularRadius::Independent(id) => vec![super::variables::SketchVariable::Radius(id)],
            CircularRadius::Derived(start) => self.point_variables(&[frame.center, start]),
        };
        variables
            .into_iter()
            .map(|variable| self.variable_column(variable))
            .collect()
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
            (1..=18 | 21..=super::SKETCH_RECIPE_VERSION, None) => Ok(()),
            (19..=super::SKETCH_RECIPE_VERSION, Some(relations)) if !relations.is_empty() => {
                for relation in relations {
                    self.own_relation(relation.owner, relation.constraint)?;
                }
                Ok(())
            }
            shape => Err(SketchError::Definition(format!(
                "structural relation recipe {shape:?}: expected no owners through version 18 or a nonempty owner list in versions 19 or later"
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
