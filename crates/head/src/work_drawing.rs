// SPDX-License-Identifier: GPL-2.0-only
macro_rules! drawing_work_methods {
    () => {
        /// Show or clear a drawing preview; example: `work.base_view_preview_from_gui(None)?`.
        fn base_view_preview_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiBaseViewRequest>,
        ) -> Result<bool, String> {
            Err(format!(
                "drawing view preview {request:?}: expected a drawing owner"
            ))
        }
        /// Create a drawing view through the public drawing commands; example: `work.base_view_from_gui(&request)?`.
        fn base_view_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiBaseViewRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "drawing view {request:?}: expected a drawing owner"
            ))
        }
    };
}
pub(crate) use drawing_work_methods;
