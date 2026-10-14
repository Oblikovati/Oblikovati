// SPDX-License-Identifier: GPL-2.0-only
use super::{
    fields::{distance, field},
    *,
};
use crate::sketch::chain_layout::{line_chain_indices, require_chain_vertices};
impl SketchPlacementRecipe {
    /// Describe a complete chain with shared vertices and final-segment fields.
    /// Example: `SketchPlacementRecipe::line_chain(&[[0.,0.],[4.,0.],[4.,3.]], false)?`.
    pub fn line_chain(points: &[[f64; 2]], closed: bool) -> Result<Self, SketchError> {
        require_chain_vertices(points, closed)?;
        require_chain_lengths(points, closed)?;
        let recipe = Self {
            points: points.to_vec(),
            members: super::geometry::chain_members(points.len(), closed),
            relations: vec![],
            fields: chain_tail_fields(points, closed),
            circular_checks: vec![],
        };
        recipe.validate()?;
        Ok(recipe)
    }
}
fn require_chain_lengths(points: &[[f64; 2]], closed: bool) -> Result<(), SketchError> {
    for indices in line_chain_indices(points.len(), closed) {
        let edge = indices.map(|index| points[index]);
        let length = distance(edge);
        if !length.is_finite() || length == 0. {
            return Err(SketchError::Definition(format!(
                "line-chain edge {indices:?} at {edge:?}, length {length}: expected a finite nonzero length"
            )));
        }
    }
    Ok(())
}
fn chain_tail_fields(points: &[[f64; 2]], closed: bool) -> Vec<SketchPlacementField> {
    use SketchPlacementBinding::{Distance, SteeringAngle};
    let last = points.len() - 1;
    let indices = if closed { [last, 0] } else { [last - 1, last] };
    let witness = indices.map(|index| points[index]);
    let [a, b] = witness;
    let angle = (b[1] - a[1]).atan2(b[0] - a[0]);
    // Go aa8297d1 (#2032): dynamic input describes the cursor segment, not the first segment.
    vec![
        field("Length", distance(witness), witness, Distance(indices)),
        field("Angle", angle, witness, SteeringAngle),
    ]
}
