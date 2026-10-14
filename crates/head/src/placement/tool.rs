// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{
    CircumcircleOptions, SketchCircleConstruction, SketchPlacementRecipe,
};
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
        if let Self::ThreePointCircle { options } = self
            && (!options.maximum_point_residual.is_finite() || options.maximum_point_residual < 0.)
        {
            return Err(format!(
                "circle accuracy {options:?}: expected a finite nonnegative residual bound"
            ));
        }
        Ok(())
    }
    pub(crate) fn name(self) -> &'static str {
        match self {
            Self::StraightSlot { .. } => "Slot",
            Self::Circle => "Circle",
            Self::ThreePointCircle { .. } => "Three-point circle",
            Self::Rectangle => "Rectangle",
            Self::CenterRectangle => "Center rectangle",
            Self::ThreePointRectangle => "Three-point rectangle",
        }
    }
    pub(crate) fn point_count(self) -> usize {
        if matches!(
            self,
            Self::ThreePointRectangle | Self::ThreePointCircle { .. }
        ) {
            3
        } else {
            2
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
        construction: &dyn SketchCircleConstruction,
    ) -> Result<SketchPlacementRecipe, String> {
        let result = match self {
            Self::Circle => SketchPlacementRecipe::circle(
                accepted[0],
                (cursor[0] - accepted[0][0]).hypot(cursor[1] - accepted[0][1]),
            ),
            Self::ThreePointCircle { options } => {
                return circular_recipe([accepted[0], accepted[1], cursor], options, construction);
            }
            Self::StraightSlot { width } => {
                SketchPlacementRecipe::straight_slot([accepted[0], cursor], width)
            }
            Self::Rectangle => SketchPlacementRecipe::rectangle([accepted[0], cursor]),
            Self::CenterRectangle => SketchPlacementRecipe::center_rectangle(accepted[0], cursor),
            Self::ThreePointRectangle => {
                SketchPlacementRecipe::three_point_rectangle([accepted[0], accepted[1]], cursor)
            }
        };
        result.map_err(|cause| cause.to_string())
    }
    pub(crate) fn prompt(self, placed: usize) -> String {
        let next = match (self, placed) {
            (Self::StraightSlot { .. }, 0) => "the first centre point",
            (Self::StraightSlot { .. }, _) => "the second centre point",
            (Self::Circle, 0) => "the centre point",
            (Self::Circle, _) => "a circumference point",
            (Self::ThreePointCircle { .. }, 0) => "the first circumference point",
            (Self::ThreePointCircle { .. }, 1) => "the second circumference point",
            (Self::ThreePointCircle { .. }, _) => "the third circumference point",
            (Self::Rectangle, 0) => "the first corner",
            (Self::Rectangle, _) => "the opposite corner",
            (Self::CenterRectangle, 0) => "the centre point",
            (Self::CenterRectangle, _) => "a corner",
            (Self::ThreePointRectangle, 0) => "the first base point",
            (Self::ThreePointRectangle, 1) => "the second base point",
            (Self::ThreePointRectangle, _) => "the height point",
        };
        format!("{}: select {next}. Escape cancels.", self.name())
    }
}

fn circular_recipe(
    points: [[f64; 2]; 3],
    options: CircumcircleOptions,
    construction: &dyn SketchCircleConstruction,
) -> Result<SketchPlacementRecipe, String> {
    let circle = construction
        .circumcircle(points, options)
        .map_err(|cause| cause.to_string())?;
    SketchPlacementRecipe::circle(circle.center, circle.radius).map_err(|cause| cause.to_string())
}

#[cfg(test)]
#[path = "tool_tests.rs"]
mod tests;
