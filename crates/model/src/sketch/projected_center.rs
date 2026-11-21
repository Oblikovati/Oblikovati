// SPDX-License-Identifier: GPL-2.0-only
//! Verify retained projected solver coordinates and source identities.
use super::{SketchPoint2D, SketchSolveOutcome, solve::same_equation_evidence};
use crate::parameter::ParameterCollection;
impl SketchSolveOutcome {
    pub(super) fn verified_projected_center(
        &self,
        parameters: &ParameterCollection<u64>,
        anchor: SketchPoint2D,
    ) -> Result<(Vec<f64>, [usize; 2]), String> {
        if self.grounded_inputs.len() != 1
            || self.grounded_inputs[0].id != anchor.id
            || self.grounded_inputs[0].position.map(f64::to_bits)
                != anchor.position.map(f64::to_bits)
        {
            return Err(format!(
                "grounded inputs {:?}: expected the unchanged origin anchor {anchor:?}",
                self.grounded_inputs
            ));
        }
        let point = self
            .candidate
            .points()
            .iter()
            .position(|point| point.id == anchor.id)
            .ok_or_else(|| format!("anchor {:?}: expected an owned point", anchor.id))?;
        let columns = [point * 2, point * 2 + 1];
        let values = self.candidate.variable_values();
        if columns
            .iter()
            .enumerate()
            .any(|(axis, &column)| values[column].to_bits() != anchor.position[axis].to_bits())
        {
            return Err(format!(
                "projected anchor {anchor:?}: expected the accepted candidate to retain its exact grounded inputs"
            ));
        }
        if self.active_slots.len() != values.len()
            || self
                .active_slots
                .iter()
                .enumerate()
                .any(|(column, slot)| match slot {
                    Some(index) => self
                        .numerical
                        .values
                        .get(*index)
                        .is_none_or(|value| value.to_bits() != values[column].to_bits()),
                    None => !columns.contains(&column),
                })
            || columns
                .iter()
                .any(|&column| self.active_slots[column].is_some())
        {
            return Err("grounded solve layout: expected the unchanged two origin coordinates and active values".into());
        }
        let mut active: Vec<_> = self.active_slots.iter().flatten().copied().collect();
        active.sort_unstable();
        if active != (0..self.numerical.values.len()).collect::<Vec<_>>() {
            return Err(
                "grounded solve layout: expected a bijective active-variable mapping".into(),
            );
        }
        let prepared =
            super::compile::prepare(&self.candidate, parameters).map_err(|e| e.to_string())?;
        if prepared.origins != self.origins {
            return Err("projected solve: expected unchanged relation identities".into());
        }
        for (index, source) in prepared.sources.iter().enumerate() {
            verify_projected_relation(self, source, index, &values, anchor.id)?;
        }
        Ok((values, columns))
    }
}
fn verify_projected_relation(
    outcome: &SketchSolveOutcome,
    source: &super::residual::PreparedConstraint,
    index: usize,
    values: &[f64],
    anchor: super::SketchEntityId,
) -> Result<(), String> {
    let operands = &outcome.operands[index];
    if operands.len() != source.variables.len()
        || operands
            .iter()
            .zip(&source.variables)
            .any(|(operand, &column)| {
                let point = outcome
                    .candidate
                    .points()
                    .get(column / 2)
                    .map(|point| (point.id, column % 2));
                operand.column != column
                    || operand.point != point
                    || operand.externally_grounded != point.is_some_and(|(id, _)| id == anchor)
                    || operand.value.to_bits() != values[column].to_bits()
            })
    {
        return Err(format!(
            "projected relation {:?}: expected unchanged source operands",
            outcome.origins[index]
        ));
    }
    let current =
        super::equation_evidence::rows(&outcome.candidate, outcome.origins[index], source);
    if !same_equation_evidence(&outcome.equations[index], &current) {
        return Err(format!(
            "projected relation {:?}: expected unchanged equation targets",
            outcome.origins[index]
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId},
        units::UnitsOfMeasure,
    };
    #[test]
    fn forged_elimination_mapping_and_anchor_values_are_refused() {
        let mut sketch = Sketch2D::default();
        let anchor = SketchPoint2D {
            id: SketchEntityId(1),
            position: [0., 0.],
        };
        sketch.add_point(anchor.id, anchor.position).unwrap();
        sketch.add_point(SketchEntityId(2), [1., 0.]).unwrap();
        sketch
            .add_constraint(
                SketchConstraintId(1),
                PlanarConstraint::Coincident {
                    points: [anchor.id, SketchEntityId(2)],
                },
            )
            .unwrap();
        let parameters = ParameterCollection::new(UnitsOfMeasure::default());
        let outcome = sketch
            .solve_with_grounded_points(&parameters, &[anchor], Default::default(), || false)
            .unwrap();
        assert!(
            outcome
                .verified_projected_center(&parameters, anchor)
                .is_ok()
        );
        let mut forged = outcome.clone();
        forged.candidate.set_position(anchor.id, [1., 0.]).unwrap();
        assert!(
            forged
                .verified_projected_center(&parameters, anchor)
                .unwrap_err()
                .contains("grounded inputs")
        );
        let mut forged = outcome.clone();
        forged.active_slots[2] = forged.active_slots[3];
        assert!(
            forged
                .verified_projected_center(&parameters, anchor)
                .is_err()
        );
    }
}
