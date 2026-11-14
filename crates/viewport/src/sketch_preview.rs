// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{Sketch2D, SketchEntityId, SketchPlacementRecipe, SketchPlane};
/// Owned transient presentation, either a standalone recipe or a verified private document result.
#[derive(Clone, Debug, PartialEq)]
pub enum ViewportSketchPreview {
    Recipe {
        solve_options: oblikovati_solve::nonlinear::SolveOptions,
        plane: SketchPlane,
        recipe: SketchPlacementRecipe,
    },
    Resolved {
        plane: SketchPlane,
        geometry: Box<Sketch2D>,
        members: Vec<SketchEntityId>,
    },
}
impl ViewportSketchPreview {
    /// Read the common placement frame; example: `preview.plane().to_model(point)?`.
    pub fn plane(&self) -> SketchPlane {
        match self {
            Self::Recipe { plane, .. } | Self::Resolved { plane, .. } => *plane,
        }
    }
    /// Count only transient members, excluding existing source geometry.
    /// Example: `preview.member_count()` for a presentation diagnostic.
    pub fn member_count(&self) -> usize {
        match self {
            Self::Recipe { recipe, .. } => recipe.members.len(),
            Self::Resolved { members, .. } => members.len(),
        }
    }
}
