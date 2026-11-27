// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraintId, SketchError, SketchPoint2D};
use crate::parameter::ParameterCollection;
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
use std::collections::BTreeSet;

impl Sketch2D {
    /// Solve a corner's connected equations while retaining unrelated authored geometry.
    /// Example: `sketch.solve_corner_component(owner, &parameters, &anchors, options)?`.
    pub fn solve_corner_component(
        &self,
        owner: super::SketchEntityId,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        options: SolveOptions,
    ) -> Result<Sketch2D, SketchError> {
        let columns = self.corner_related_columns(owner)?;
        if columns.is_empty() {
            return Err(SketchError::Definition(
                "corner component has no variables: expected an owned corner relation".into(),
            ));
        }
        let equations = self.corner_equations(owner, &columns)?;
        let outcome =
            equations.solve_with_grounded_points(parameters, anchors, options, || false)?;
        if outcome.numerical().termination != SolveTermination::Converged {
            return Err(SketchError::Definition(format!(
                "corner component solve termination {:?}: expected convergence",
                outcome.numerical().termination
            )));
        }
        let original = self.variable_values();
        let mut solved = outcome.candidate().variable_values();
        for (column, value) in original.into_iter().enumerate() {
            if !columns.contains(&column) {
                solved[column] = value;
            }
        }
        self.candidate_values(&solved)
    }
    fn corner_equations(
        &self,
        owner: super::SketchEntityId,
        columns: &BTreeSet<usize>,
    ) -> Result<Sketch2D, SketchError> {
        let mut candidate = self.clone();
        let relations = self.corner_relation_ids(owner, columns)?;
        candidate
            .constraints
            .retain(|row| relations.contains(&row.id));
        candidate
            .reference_dimensions
            .retain(|id| relations.contains(id));
        candidate
            .dimension_text_points
            .retain(|id, _| relations.contains(id));
        candidate
            .dimension_drive_limits
            .retain(|id, _| relations.contains(id));
        candidate
            .owned_relations
            .retain(|id, _| relations.contains(id));
        candidate.retain_corner_implicit_equations(columns)?;
        Ok(candidate)
    }
    fn corner_relation_ids(
        &self,
        owner: super::SketchEntityId,
        columns: &BTreeSet<usize>,
    ) -> Result<BTreeSet<SketchConstraintId>, SketchError> {
        let mut result = BTreeSet::new();
        for row in &self.constraints {
            if self.owned_relations.get(&row.id) == Some(&owner)
                || self
                    .relation_columns(&row.relation)?
                    .iter()
                    .any(|column| columns.contains(column))
            {
                result.insert(row.id);
            }
        }
        Ok(result)
    }
    fn retain_corner_implicit_equations(
        &mut self,
        columns: &BTreeSet<usize>,
    ) -> Result<(), SketchError> {
        let arcs = self
            .arcs
            .iter()
            .map(|arc| self.arc_columns(arc))
            .collect::<Result<Vec<_>, _>>()?;
        retain_connected(&mut self.arcs, arcs, columns);
        let points = self
            .pattern_point_links
            .iter()
            .map(|link| super::compile::pattern_point_columns(self, link))
            .collect::<Result<Vec<_>, _>>()?;
        retain_connected(&mut self.pattern_point_links, points, columns);
        let radii = self
            .pattern_radius_links
            .iter()
            .map(|link| super::compile::pattern_radius_columns(self, link))
            .collect::<Result<Vec<_>, _>>()?;
        retain_connected(&mut self.pattern_radius_links, radii, columns);
        self.pattern_offsets.retain(|offset| {
            self.pattern_point_links
                .iter()
                .any(|link| link.member == offset.member)
        });
        // #4026: retained point links carry the solver equations and their offset sources.
        // Pattern records validate the full sketch and can depend on a disconnected sick source.
        self.rectangular_patterns.clear();
        Ok(())
    }
}
fn retain_connected<T>(
    values: &mut Vec<T>,
    source_columns: Vec<Vec<usize>>,
    selected: &BTreeSet<usize>,
) {
    let mut rows = source_columns.into_iter();
    values.retain(|_| {
        rows.next()
            .is_some_and(|columns| columns.iter().any(|column| selected.contains(column)))
    });
}
