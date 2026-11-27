// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for straight-hole preview and creation.
macro_rules! hole_work_methods {
    () => {
        /// Preview a hole on the selected face; example: `work.preview_hole_from_gui(Some(&request))?`.
        fn preview_hole_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiHoleRequest>,
        ) -> Result<bool, String> {
            Err(format!(
                "hole preview {request:?}: expected a certified part owner"
            ))
        }
        /// Commit a selected-face hole; example: `work.commit_hole_from_gui(&request)?`.
        fn commit_hole_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiHoleRequest,
        ) -> Result<bool, String> {
            Err(format!("hole {request:?}: expected a certified part owner"))
        }
    };
}
pub(crate) use hole_work_methods;
