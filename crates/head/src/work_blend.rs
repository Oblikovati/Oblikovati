// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for blend feature edits and Chamfer creation.
macro_rules! blend_work_methods {
    () => {
        /// Preview a Fillet radius edit; example: `work.preview_fillet_edit(&request)?`.
        fn preview_fillet_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "fillet edit {request:?}: expected a certified Fillet owner"
            ))
        }
        /// Commit a Fillet radius edit; example: `work.commit_fillet_edit(&request)?`.
        fn commit_fillet_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "fillet edit {request:?}: expected a certified Fillet owner"
            ))
        }
        /// Preview a Chamfer distance edit; example: `work.preview_chamfer_edit(&request)?`.
        fn preview_chamfer_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "chamfer edit {request:?}: expected a certified Chamfer owner"
            ))
        }
        /// Commit a Chamfer distance edit; example: `work.commit_chamfer_edit(&request)?`.
        fn commit_chamfer_edit(
            &mut self,
            request: &oblikovati_gui::GuiFilletEditRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "chamfer edit {request:?}: expected a certified Chamfer owner"
            ))
        }
        /// Preview selected edges as a chamfer; example: `work.preview_chamfer_from_gui(Some(&request))?`.
        fn preview_chamfer_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiFilletRequest>,
        ) -> Result<bool, String> {
            Err(format!(
                "chamfer preview {request:?}: expected a certified chamfer owner"
            ))
        }
        /// Commit selected edges as a chamfer; example: `work.commit_chamfer_from_gui(&request)?`.
        fn commit_chamfer_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiFilletRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "chamfer {request:?}: expected a certified chamfer owner"
            ))
        }
    };
}
pub(crate) use blend_work_methods;
