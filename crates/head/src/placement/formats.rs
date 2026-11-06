// SPDX-License-Identifier: GPL-2.0-only
use super::SketchPlacementState;
use oblikovati_model::{
    history::SketchLineChainCreation,
    sketch::{SketchCircleConstruction, SketchError, SketchPlacementRecipe},
};
impl SketchPlacementState {
    pub(super) fn formatted_line_recipe(
        &self,
        input: &SketchLineChainCreation,
    ) -> Result<SketchPlacementRecipe, SketchError> {
        let mut recipe = SketchPlacementRecipe::line_chain(&input.points, false)?;
        recipe.apply_creation_formats(input.construction, input.centerline)?;
        Ok(recipe)
    }
    pub(crate) fn refresh_creation_formats(
        &mut self,
        construction: &dyn SketchCircleConstruction,
    ) -> Result<(), String> {
        let Some(point) = self.cursor_point else {
            return Ok(());
        };
        self.cursor(Some(point), false, construction)?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};
    fn rectangle_draft() -> SketchPlacementState {
        let mut state = SketchPlacementState::new(
            1,
            DocumentSketchId(1),
            0,
            SketchPlane::xy(),
            crate::SketchPlacementTool::Rectangle,
        )
        .unwrap();
        state
            .cursor(Some([0., 0.]), true, &MartensiteSketchConstruction)
            .unwrap();
        state
            .cursor(Some([4., 3.]), false, &MartensiteSketchConstruction)
            .unwrap();
        state
    }
    #[test]
    fn creation_format_refresh_rebuilds_roles_and_keeps_the_same_seed() {
        let mut state = rectangle_draft();
        let original = state.preview.clone();
        state.formats.construction = true;
        state.formats.centerline = true;
        state
            .refresh_creation_formats(&MartensiteSketchConstruction)
            .unwrap();
        assert!(
            state
                .preview
                .as_ref()
                .unwrap()
                .members
                .iter()
                .all(|member| member.construction && member.centerline)
        );
        assert_eq!(
            state.preview.as_ref().unwrap().points,
            original.as_ref().unwrap().points
        );
        state.formats = Default::default();
        state
            .refresh_creation_formats(&MartensiteSketchConstruction)
            .unwrap();
        assert_eq!(state.preview, original);
    }
    #[test]
    fn missing_cursor_cannot_restore_a_stale_preview_when_format_changes() {
        let mut state = rectangle_draft();
        state
            .cursor(None, false, &MartensiteSketchConstruction)
            .unwrap();
        state.formats.construction = true;
        state
            .refresh_creation_formats(&MartensiteSketchConstruction)
            .unwrap();
        assert!(state.preview.is_none());
    }
}
