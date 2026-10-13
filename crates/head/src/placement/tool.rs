// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::SketchPlacementRecipe;
/// A recipe choice in the shared mouse placement pipeline; dimensions are in working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum SketchPlacementTool {
    StraightSlot { width: f64 },
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
        Ok(())
    }
    pub(crate) fn name(self) -> &'static str {
        match self {
            Self::StraightSlot { .. } => "Slot",
            Self::Rectangle => "Rectangle",
            Self::CenterRectangle => "Center rectangle",
            Self::ThreePointRectangle => "Three-point rectangle",
        }
    }
    pub(crate) fn point_count(self) -> usize {
        if self == Self::ThreePointRectangle {
            3
        } else {
            2
        }
    }
    pub(crate) fn recipe(
        self,
        accepted: &[[f64; 2]],
        cursor: [f64; 2],
    ) -> Result<SketchPlacementRecipe, String> {
        let expected = self.point_count() - 1;
        if accepted.len() != expected {
            return Err(format!(
                "{} accepted points {}: expected {expected}",
                self.name(),
                accepted.len()
            ));
        }
        let result = match self {
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

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn incomplete_point_sequences_are_refused_before_indexing() {
        for tool in [
            SketchPlacementTool::StraightSlot { width: 2. },
            SketchPlacementTool::Rectangle,
            SketchPlacementTool::CenterRectangle,
            SketchPlacementTool::ThreePointRectangle,
        ] {
            assert!(tool.recipe(&[], [1., 2.]).unwrap_err().contains("expected"));
            assert!(tool.recipe(&[[0., 0.]; 3], [1., 2.]).is_err());
        }
    }
}
