// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for editing a Fillet radius.
macro_rules! fillet_edit_work_methods {
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
    };
}
pub(crate) use fillet_edit_work_methods;
