// SPDX-License-Identifier: GPL-2.0-only
//! Owner service for selected-body appearance clearing.
macro_rules! appearance_work_methods {
    () => {
        /// Clear only the selected body's own appearance override.
        /// Example: `work.clear_selected_body_appearance_from_gui(document, key)?`.
        fn clear_selected_body_appearance_from_gui(
            &mut self,
            document: u64,
            key: &str,
        ) -> Result<bool, String> {
            Err(format!(
                "body {key:?} in document {document}: expected a selection owner"
            ))
        }
    };
}
