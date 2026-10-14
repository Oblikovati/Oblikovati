// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{
    CircumcircleOptions, SketchCircleConstruction, SketchPlacementRecipe,
};
#[path = "circular_recipes.rs"]
mod circular_recipes;
/// A recipe choice in the shared mouse placement pipeline; dimensions are in working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacementTool {
    StraightSlot {
        width: f64,
    },
    Circle,
    /// Caller sets the residual bound in document working units, independent of view scale.
    ThreePointCircle {
        options: CircumcircleOptions,
    },
    Rectangle,
    CenterRectangle,
    CenterPointArc,
    ThreePointArc {
        options: CircumcircleOptions,
    },
    ThreePointRectangle,
}
impl SketchPlacementTool {
    pub(crate) fn validate(self) -> Result<(), String> {
        if let Self::StraightSlot { width } = self
            && (!width.is_finite() || width <= 0.)
        {
            return Err(format!(
                "slot width {width}: expected a positive finite working-unit length"
            ));
        }
        if let Self::ThreePointCircle { options } | Self::ThreePointArc { options } = self
            && (!options.maximum_point_residual.is_finite() || options.maximum_point_residual < 0.)
        {
            return Err(format!(
                "circular construction accuracy {options:?}: expected a finite nonnegative residual bound"
            ));
        }
        Ok(())
    }
    pub(crate) fn name(self) -> &'static str {
        match self {
            Self::StraightSlot { .. } => "Slot",
            Self::Circle => "Circle",
            Self::CenterPointArc => "Centre-point arc",
            Self::ThreePointArc { .. } => "Three-point arc",
            Self::ThreePointCircle { .. } => "Three-point circle",
            Self::Rectangle => "Rectangle",
            Self::CenterRectangle => "Center rectangle",
            Self::ThreePointRectangle => "Three-point rectangle",
        }
    }
    pub(crate) fn point_count(self) -> usize {
        match self {
            Self::StraightSlot { .. } | Self::Circle | Self::Rectangle | Self::CenterRectangle => 2,
            Self::ThreePointRectangle
            | Self::ThreePointCircle { .. }
            | Self::ThreePointArc { .. }
            | Self::CenterPointArc => 3,
        }
    }
    pub(crate) fn recipe(
        self,
        accepted: &[[f64; 2]],
        cursor: [f64; 2],
        construction: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementRecipe, String> {
        let expected = self.point_count() - 1;
        if accepted.len() != expected {
            return Err(format!(
                "{} accepted points {}: expected {expected}",
                self.name(),
                accepted.len()
            ));
        }
        self.resolved_recipe(accepted, cursor, construction)
    }
    fn resolved_recipe(
        self,
        accepted: &[[f64; 2]],
        cursor: [f64; 2],
        service: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementRecipe, String> {
        use circular_recipes::{center_recipe, circular_recipe, radial_recipe};
        use oblikovati_model::sketch::SketchPlacementRecipe as Recipe;
        let first = accepted[0];
        let next = accepted.get(1).copied().unwrap_or(cursor);
        let result = match self {
            Self::CenterPointArc => center_recipe([first, next, cursor], service),
            Self::ThreePointArc { options } => {
                circular_recipe([first, cursor, next], true, options, service)
            }
            Self::ThreePointCircle { options } => {
                circular_recipe([first, next, cursor], false, options, service)
            }
            Self::Circle => radial_recipe([first, cursor]),
            Self::StraightSlot { width } => Recipe::straight_slot([first, cursor], width),
            Self::Rectangle => Recipe::rectangle([first, cursor]),
            Self::CenterRectangle => Recipe::center_rectangle(first, cursor),
            Self::ThreePointRectangle => Recipe::three_point_rectangle([first, next], cursor),
        };
        result.map_err(|cause| cause.to_string())
    }
    pub(crate) fn prompt(self, placed: usize) -> String {
        let labels = self.point_labels();
        let next = labels[placed.min(labels.len() - 1)];
        format!("{}: select {next}. Escape cancels.", self.name())
    }
    fn point_labels(self) -> &'static [&'static str] {
        match self {
            Self::StraightSlot { .. } => &["the first centre point", "the second centre point"],
            Self::Circle => &["the centre point", "a circumference point"],
            Self::ThreePointCircle { .. } => &[
                "the first circumference point",
                "the second circumference point",
                "the third circumference point",
            ],
            Self::CenterPointArc => &["the centre point", "the start point", "the end point"],
            Self::ThreePointArc { .. } => {
                &["the start point", "the end point", "a point on the arc"]
            }
            Self::Rectangle => &["the first corner", "the opposite corner"],
            Self::CenterRectangle => &["the centre point", "a corner"],
            Self::ThreePointRectangle => &[
                "the first base point",
                "the second base point",
                "the height point",
            ],
        }
    }
}

#[cfg(test)]
#[path = "tool_tests.rs"]
mod tests;

#[cfg(test)]
#[path = "arc_tool_tests.rs"]
mod arc_tests;
