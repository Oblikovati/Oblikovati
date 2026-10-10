// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchArc2D, SketchCircle2D, SketchConstraint, SketchError, SketchLine2D,
    SketchPoint2D,
};
/// First Rust planar recipe schema; this is independent of the enclosing document schema.
pub const SKETCH_RECIPE_VERSION: u32 = 1;
/// Owned authored values; no numerical workspace, derived topology or system equations.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRecipe {
    pub version: u32,
    pub points: Vec<SketchPoint2D>,
    pub lines: Vec<SketchLine2D>,
    pub circles: Vec<SketchCircle2D>,
    pub arcs: Vec<SketchArc2D>,
    pub constraints: Vec<SketchConstraint>,
}
impl Sketch2D {
    /// Capture authored values in stable order; example: `let recipe = sketch.recipe();`.
    pub fn recipe(&self) -> SketchRecipe {
        SketchRecipe {
            version: SKETCH_RECIPE_VERSION,
            points: self.points.clone(),
            lines: self.lines.clone(),
            circles: self.circles.clone(),
            arcs: self.arcs.clone(),
            constraints: self.constraints.clone(),
        }
    }
    /// Restore into a fresh owner after validation; example: `let sketch = Sketch2D::from_recipe(&recipe)?;`.
    pub fn from_recipe(recipe: &SketchRecipe) -> Result<Self, SketchError> {
        if recipe.version != SKETCH_RECIPE_VERSION {
            return Err(SketchError::Definition(format!(
                "sketch recipe version {}: expected {SKETCH_RECIPE_VERSION}",
                recipe.version
            )));
        }
        let mut sketch = Self::default();
        sketch.restore_geometry(recipe)?;
        for constraint in &recipe.constraints {
            sketch.add_constraint(constraint.id, constraint.relation.clone())?;
        }
        Ok(sketch)
    }
    fn restore_geometry(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        for point in &recipe.points {
            self.add_point(point.id, point.position)?;
        }
        for line in &recipe.lines {
            self.add_line(line.id, line.endpoints)?;
        }
        self.restore_circular_geometry(recipe)
    }
    fn restore_circular_geometry(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        for circle in &recipe.circles {
            self.add_circle(circle.id, circle.center, circle.radius)?;
        }
        for arc in &recipe.arcs {
            self.add_arc(
                arc.id,
                arc.center,
                [arc.start, arc.end],
                arc.counter_clockwise,
            )?;
        }
        Ok(())
    }
}
