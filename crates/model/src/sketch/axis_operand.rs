// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, ellipse::variables::EllipseComponent};
/// An authored analytic direction; the ellipse selector belongs to its own operand.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case", deny_unknown_fields)]
pub enum SketchAxisOperand {
    Line {
        line: SketchEntityId,
    },
    Ellipse {
        ellipse: SketchEntityId,
        major_axis: bool,
    },
}
impl SketchAxisOperand {
    /// Inspect the source identity; example: `operand.entity()` for constraint enumeration.
    pub fn entity(self) -> SketchEntityId {
        match self {
            Self::Line { line } => line,
            Self::Ellipse { ellipse, .. } => ellipse,
        }
    }
    pub(super) fn columns(self, sketch: &Sketch2D) -> Result<Vec<usize>, SketchError> {
        match self {
            Self::Line { line } => sketch
                .point_variables(&sketch.line(line)?.endpoints)
                .into_iter()
                .map(|variable| sketch.variable_column(variable))
                .collect(),
            Self::Ellipse { ellipse, .. } => Ok(vec![
                sketch.ellipse_column(ellipse, EllipseComponent::Orientation)?,
            ]),
        }
    }
    pub(super) fn layout(self) -> AxisLayout {
        match self {
            Self::Line { .. } => AxisLayout::Line,
            Self::Ellipse { major_axis, .. } => AxisLayout::Ellipse { major_axis },
        }
    }
}
#[derive(Clone, Copy)]
pub(super) enum AxisLayout {
    Line,
    Ellipse { major_axis: bool },
}
impl AxisLayout {
    pub(super) fn width(self) -> usize {
        match self {
            Self::Line => 4,
            Self::Ellipse { .. } => 1,
        }
    }
}
