// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::CircumcircleOptions;
#[cfg(any(target_os = "linux", target_os = "windows", test))]
use oblikovati_model::sketch::{SketchCircleConstruction, SketchPlacementRecipe};
#[cfg(any(target_os = "linux", target_os = "windows", test))]
#[path = "circular_recipes.rs"]
mod circular_recipes;
#[cfg(any(target_os = "linux", target_os = "windows", test))]
#[path = "ellipse_recipe.rs"]
mod ellipse_recipe;
/// A recipe choice in the shared mouse placement pipeline; dimensions are in working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacementTool {
    Line,
    Point,
    StraightSlot {
        width: f64,
    },
    Circle,
    CenterPointArcSlot {
        width: f64,
    },
    ThreePointArcSlot {
        width: f64,
        options: CircumcircleOptions,
    },
    Polygon {
        sides: usize,
    },
    /// Caller sets the residual bound in document working units, independent of view scale.
    ThreePointCircle {
        options: CircumcircleOptions,
    },
    Rectangle,
    CenterRectangle,
    CenterPointArc,
    Ellipse,
    ThreePointArc {
        options: CircumcircleOptions,
    },
    ThreePointRectangle,
    ThreePointCenterRectangle,
}
#[cfg(any(target_os = "linux", target_os = "windows", test))]
impl SketchPlacementTool {
    pub(crate) fn validate(self) -> Result<(), String> {
        if let Self::StraightSlot { width }
        | Self::CenterPointArcSlot { width }
        | Self::ThreePointArcSlot { width, .. } = self
            && (!width.is_finite() || width <= 0.)
        {
            return Err(format!(
                "slot width {width}: expected a positive finite working-unit length"
            ));
        }
        if let Self::ThreePointCircle { options }
        | Self::ThreePointArc { options }
        | Self::ThreePointArcSlot { options, .. } = self
            && (!options.maximum_point_residual.is_finite() || options.maximum_point_residual < 0.)
        {
            return Err(format!(
                "circular construction accuracy {options:?}: expected a finite nonnegative residual bound"
            ));
        }
        if let Self::Polygon { sides } = self
            && sides < 3
        {
            return Err(format!("polygon sides {sides}: expected at least 3"));
        }
        Ok(())
    }
    pub(crate) fn name(self) -> &'static str {
        match self {
            Self::StraightSlot { .. } => "Slot",
            Self::Line => "Line",
            Self::Point => "Point",
            Self::Circle => "Circle",
            Self::CenterPointArcSlot { .. } => "Centre-point arc slot",
            Self::ThreePointArcSlot { .. } => "Three-point arc slot",
            Self::Polygon { .. } => "Polygon",
            Self::Ellipse => "Ellipse",
            Self::CenterPointArc => "Centre-point arc",
            Self::ThreePointArc { .. } => "Three-point arc",
            Self::ThreePointCircle { .. } => "Three-point circle",
            Self::Rectangle => "Rectangle",
            Self::CenterRectangle => "Center rectangle",
            Self::ThreePointRectangle => "Three-point rectangle",
            Self::ThreePointCenterRectangle => "Three-point center rectangle",
        }
    }
    pub(crate) fn point_count(self) -> usize {
        match self {
            Self::Point => 1,
            Self::StraightSlot { .. }
            | Self::Circle
            | Self::Line
            | Self::Rectangle
            | Self::CenterRectangle
            | Self::Polygon { .. } => 2,
            Self::ThreePointRectangle
            | Self::ThreePointCenterRectangle
            | Self::ThreePointCircle { .. }
            | Self::ThreePointArc { .. }
            | Self::CenterPointArc
            | Self::Ellipse
            | Self::CenterPointArcSlot { .. }
            | Self::ThreePointArcSlot { .. } => 3,
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
        use circular_recipes::{
            CircularRecipeKind as Circular, center_recipe, center_slot_recipe, circular_recipe,
            radial_recipe,
        };
        use oblikovati_model::sketch::SketchPlacementRecipe as Recipe;
        let first = accepted.first().copied().unwrap_or(cursor);
        let next = accepted.get(1).copied().unwrap_or(cursor);
        let result = match self {
            Self::Line => Recipe::line_chain(&[first, cursor], false),
            Self::Point => Recipe::point(cursor),
            Self::CenterPointArcSlot { width } => {
                center_slot_recipe([first, next, cursor], width, service)
            }
            Self::ThreePointArcSlot { width, options } => circular_recipe(
                [first, cursor, next],
                Circular::Slot { width },
                options,
                service,
            ),
            Self::CenterPointArc => center_recipe([first, next, cursor], service),
            Self::ThreePointArc { options } => {
                circular_recipe([first, cursor, next], Circular::Arc, options, service)
            }
            Self::ThreePointCircle { options } => {
                circular_recipe([first, next, cursor], Circular::Circle, options, service)
            }
            Self::Ellipse => ellipse_recipe::from_clicks([first, next, cursor]),
            Self::Polygon { sides } => {
                Recipe::polygon(oblikovati_model::sketch::SketchPolygonSeed {
                    center: first,
                    through: cursor,
                    sides,
                    inscribed: true,
                })
            }
            Self::Circle => radial_recipe([first, cursor]),
            Self::StraightSlot { width } => Recipe::straight_slot([first, cursor], width),
            Self::Rectangle => Recipe::rectangle([first, cursor]),
            Self::CenterRectangle => Recipe::center_rectangle(first, cursor),
            Self::ThreePointCenterRectangle => {
                Recipe::three_point_center_rectangle([first, next], cursor)
            }
            Self::ThreePointRectangle => Recipe::three_point_rectangle([first, next], cursor),
        };
        result.map_err(|cause| cause.to_string())
    }
    /// No circle runs through these points, so this place is simply not one the command can use yet.
    pub(crate) fn has_no_curve_through(
        self,
        accepted: &[[f64; 2]],
        cursor: [f64; 2],
        construction: &dyn SketchCircleConstruction,
    ) -> bool {
        let (
            Self::ThreePointArc { options }
            | Self::ThreePointCircle { options }
            | Self::ThreePointArcSlot { options, .. },
            [start, end],
        ) = (self, accepted)
        else {
            return false;
        };
        construction
            .circumcircle([*start, cursor, *end], options)
            .is_err()
    }
    /// The values asked for at each stage, as a range of the shape's fields.
    pub(crate) fn stage_fields(self, stage: usize) -> std::ops::Range<usize> {
        match (self, stage) {
            (Self::CenterPointArc, 1) => 0..2,
            (Self::CenterPointArc, 2) => 2..3,
            (Self::ThreePointRectangle | Self::ThreePointCenterRectangle, 1) => 0..2,
            (Self::ThreePointRectangle | Self::ThreePointCenterRectangle, 2) => 2..3,
            // The end's distance and direction become its point; only the radius outlives them.
            (Self::ThreePointArc { .. }, 1) => 0..2,
            (Self::ThreePointArc { .. }, 2) => 0..1,
            _ if stage + 1 == self.point_count() => 0..self.typed_field_count(),
            _ => 0..0,
        }
    }
    /// How many values can be typed at the stage that finishes the shape.
    pub(crate) fn typed_field_count(self) -> usize {
        match self {
            Self::Circle => 1,
            Self::Rectangle | Self::CenterRectangle => 2,
            _ => 0,
        }
    }
    /// Which kind of value a field holds; only an arc's sweep is not a length.
    pub(crate) fn typed_field_dimension(
        self,
        index: usize,
    ) -> oblikovati_model::quantity::Dimension {
        let angle = match self {
            Self::CenterPointArc => index == 1 || index == 2,
            Self::ThreePointRectangle
            | Self::ThreePointCenterRectangle
            | Self::ThreePointArc { .. } => index == 1,
            _ => false,
        };
        if angle {
            return oblikovati_model::quantity::Dimension::Angle;
        }
        oblikovati_model::quantity::Dimension::Length
    }
    /// A typed zero keeps the field red on some tools; others drop the shape without a word.
    pub(crate) fn typed_zero_is_invalid(self) -> bool {
        !matches!(self, Self::Rectangle | Self::CenterRectangle)
    }
    /// A size must not collapse the shape; a direction may point anywhere, zero included.
    pub(crate) fn typed_value_is_usable(self, index: usize, value: f64) -> bool {
        let direction = index == 1
            && matches!(
                self,
                Self::CenterPointArc
                    | Self::ThreePointRectangle
                    | Self::ThreePointCenterRectangle
                    | Self::ThreePointArc { .. }
            );
        direction || !self.typed_zero_is_invalid() || value > 0.
    }
    pub(crate) fn prompt(self, placed: usize) -> String {
        match (self, placed) {
            (Self::Circle, 0) => return "Select center of circle".into(),
            (Self::Circle, _) => return "Select point on circle".into(),
            (Self::Rectangle, 0) => return "Select first corner".into(),
            (Self::Rectangle, _) => return "Select opposite corner".into(),
            (Self::CenterRectangle, 0) => return "Select center".into(),
            (Self::CenterRectangle, _) => return "Select corner".into(),
            (Self::ThreePointRectangle, 0) => return "Select first corner".into(),
            (Self::ThreePointRectangle, 1) => return "Select second corner".into(),
            (Self::ThreePointRectangle, _) => return "Select third corner".into(),
            (Self::ThreePointCenterRectangle, 0) => return "Select center".into(),
            (Self::ThreePointCenterRectangle, 1) => return "Select second point".into(),
            (Self::ThreePointCenterRectangle, _) => return "Select third point".into(),
            (Self::CenterPointArc, 0) => return "Select center".into(),
            (Self::CenterPointArc, 1) => return "Select start point".into(),
            (Self::CenterPointArc, _) => return "Select end point".into(),
            (Self::Ellipse, 0) => return "Select center".into(),
            (Self::Ellipse, 1) => return "Select first axis point".into(),
            (Self::Ellipse, _) => return "Select point on ellipse".into(),
            (Self::ThreePointArc { .. }, 0) => return "Select start of arc".into(),
            (Self::ThreePointArc { .. }, 1) => return "Select end of arc".into(),
            (Self::ThreePointArc { .. }, _) => return "Select point on arc".into(),
            _ => {}
        }
        let labels = self.point_labels();
        let next = labels[placed.min(labels.len() - 1)];
        format!("{}: select {next}. Escape cancels.", self.name())
    }
    fn point_labels(self) -> &'static [&'static str] {
        match self {
            Self::StraightSlot { .. } => &["the first centre point", "the second centre point"],
            Self::Line => &["the first point", "the next point"],
            Self::Point => &["the point position"],
            Self::Circle => &["the centre point", "a circumference point"],
            Self::Polygon { .. } => &["the centre point", "a vertex"],
            Self::Ellipse => &[
                "the centre point",
                "the major-axis endpoint",
                "the minor radius",
            ],
            Self::ThreePointCircle { .. } => &[
                "the first circumference point",
                "the second circumference point",
                "the third circumference point",
            ],
            Self::CenterPointArc | Self::CenterPointArcSlot { .. } => {
                &["the centre point", "the start point", "the end point"]
            }
            Self::ThreePointArc { .. } | Self::ThreePointArcSlot { .. } => {
                &["the start point", "the end point", "a point on the arc"]
            }
            Self::Rectangle => &["the first corner", "the opposite corner"],
            Self::CenterRectangle => &["the centre point", "a corner"],
            Self::ThreePointCenterRectangle => {
                &["the center", "the second point", "the third point"]
            }
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

#[cfg(test)]
#[path = "ellipse_tool_tests.rs"]
mod ellipse_tests;

#[cfg(test)]
#[path = "arc_slot_tool_tests.rs"]
mod arc_slot_tests;
