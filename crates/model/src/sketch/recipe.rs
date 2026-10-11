// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchArc2D, SketchCircle2D, SketchConstraint, SketchError, SketchLine2D,
    SketchPoint2D,
};
/// Latest Rust planar schema; version 2 records order and version 3 adds construction roles.
pub const SKETCH_RECIPE_VERSION: u32 = 3;
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
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub entity_order: Option<Vec<super::SketchEntityId>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub construction_curves: Option<Vec<super::SketchEntityId>>,
}
impl Sketch2D {
    /// Capture authored values in stable order; example: `let recipe = sketch.recipe();`.
    pub fn recipe(&self) -> SketchRecipe {
        let entity_order = self.saved_entity_order();
        let construction_curves = self.saved_construction_curves();
        SketchRecipe {
            version: planar_recipe_version(entity_order.is_some(), construction_curves.is_some()),
            points: self.points.clone(),
            lines: self.lines.clone(),
            circles: self.circles.clone(),
            arcs: self.arcs.clone(),
            constraints: self.constraints.clone(),
            entity_order,
            construction_curves,
        }
    }
    /// Restore into a fresh owner after validation; example: `let sketch = Sketch2D::from_recipe(&recipe)?;`.
    pub fn from_recipe(recipe: &SketchRecipe) -> Result<Self, SketchError> {
        if !matches!(recipe.version, 1..=SKETCH_RECIPE_VERSION) {
            return Err(SketchError::Definition(format!(
                "sketch recipe version {}: expected 1 through {SKETCH_RECIPE_VERSION}",
                recipe.version
            )));
        }
        let mut sketch = Self::default();
        sketch.restore_geometry(recipe)?;
        sketch.restore_entity_order(recipe)?;
        sketch.restore_construction_curves(recipe)?;
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

fn planar_recipe_version(ordered: bool, construction: bool) -> u32 {
    match (ordered, construction) {
        (_, true) => SKETCH_RECIPE_VERSION,
        (true, false) => 2,
        (false, false) => 1,
    }
}
