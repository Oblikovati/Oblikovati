// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, geometry::require_position};
/// A rigid seed edit in document working units; constraints still govern recompute.
#[derive(Clone, Copy, Debug)]
pub enum SketchSelectionTransform {
    Move([f64; 2]),
    Rotate { center: [f64; 2], radians: f64 },
}
impl SketchSelectionTransform {
    /// Name the history step; example: `SketchSelectionTransform::Move([1.0, 0.0]).label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Move(_) => "Move sketch entities",
            Self::Rotate { .. } => "Rotate sketch entities",
        }
    }
    pub(super) fn affine(self) -> Result<SelectionAffine, SketchError> {
        match self {
            Self::Move(vector) => {
                require_position(vector)?;
                Ok(SelectionAffine {
                    matrix: [[1.0, 0.0], [0.0, 1.0]],
                    offset: vector,
                })
            }
            Self::Rotate { center, radians } => SelectionAffine::rotation(center, radians),
        }
    }
}
pub(super) struct SelectionAffine {
    matrix: [[f64; 2]; 2],
    offset: [f64; 2],
}
impl SelectionAffine {
    fn rotation(center: [f64; 2], radians: f64) -> Result<Self, SketchError> {
        require_position(center)?;
        if !radians.is_finite() {
            return Err(SketchError::Definition(format!(
                "rotation angle {radians:?}: expected finite radians"
            )));
        }
        let (cos, sin) = (radians.cos(), radians.sin());
        let matrix = [[cos, -sin], [sin, cos]];
        let offset = [
            center[0] - (cos * center[0] + -sin * center[1]),
            center[1] - (sin * center[0] + cos * center[1]),
        ];
        require_position(offset)?;
        Ok(Self { matrix, offset })
    }
    pub(super) fn direction(&self, [x, y]: [f64; 2]) -> [f64; 2] {
        [
            self.matrix[0][0] * x + self.matrix[0][1] * y,
            self.matrix[1][0] * x + self.matrix[1][1] * y,
        ]
    }
    pub(super) fn point(&self, position: [f64; 2]) -> [f64; 2] {
        // #3770: preserve Go affine2.point arithmetic, including signed-zero behavior.
        let directed = self.direction(position);
        [directed[0] + self.offset[0], directed[1] + self.offset[1]]
    }
}
