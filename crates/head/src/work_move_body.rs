// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for body motion preview and creation.
macro_rules! move_body_work_methods {
    () => {
        /// Preview ordered body motion; example: `work.preview_move_body_from_gui(Some(&request))?`.
        fn preview_move_body_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiMoveBodyRequest>,
        ) -> Result<bool, String> {
            Err(format!(
                "move body preview {request:?}: expected a certified part owner"
            ))
        }
        /// Commit body motion; example: `work.commit_move_body_from_gui(&request)?`.
        fn commit_move_body_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiMoveBodyRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "move body {request:?}: expected a certified part owner"
            ))
        }
    };
}
pub(crate) use move_body_work_methods;
