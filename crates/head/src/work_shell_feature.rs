// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for a named-face Shell draft (#4106).
macro_rules! shell_work_methods {
    () => {
        /// Preview a Shell draft; example: `work.preview_shell_from_gui(Some(&request))?`.
        fn preview_shell_from_gui(
            &mut self,
            request: Option<&oblikovati_gui::GuiShellFeatureRequest>,
        ) -> Result<bool, String> {
            Err(format!(
                "shell preview {request:?}: expected a certified part owner"
            ))
        }
        /// Commit a Shell draft; example: `work.commit_shell_from_gui(&request)?`.
        fn commit_shell_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiShellFeatureRequest,
        ) -> Result<bool, String> {
            Err(format!(
                "shell {request:?}: expected a certified part owner"
            ))
        }
    };
}
pub(crate) use shell_work_methods;
