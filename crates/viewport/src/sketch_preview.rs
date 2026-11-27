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
    Text {
        plane: SketchPlane,
        anchor: [f64; 2],
        style: Box<oblikovati_api::types::SketchTextStyle>,
        working_scale: f64,
        replaces: Option<(
            oblikovati_model::document_sketch::DocumentSketchId,
            SketchEntityId,
        )>,
    },
}
impl ViewportSketchPreview {
    /// Read the common placement frame; example: `preview.plane().to_model(point)?`.
    pub fn plane(&self) -> SketchPlane {
        match self {
            Self::Recipe { plane, .. }
            | Self::Resolved { plane, .. }
            | Self::Text { plane, .. } => *plane,
        }
    }
    /// Count only transient members, excluding existing source geometry.
    /// Example: `preview.member_count()` for a presentation diagnostic.
    pub fn member_count(&self) -> usize {
        match self {
            Self::Recipe { recipe, .. } => recipe.members.len(),
            Self::Resolved { members, .. } => members.len(),
            Self::Text { style, .. } => usize::from(!style.content.trim().is_empty()),
        }
    }
}
