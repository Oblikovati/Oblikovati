// SPDX-License-Identifier: GPL-2.0-only
//! Owner service for selected-body appearance clearing.
macro_rules! appearance_work_methods {
    () => {
        /// Clear a selected face's exact override.
        /// Example: `work.clear_selected_face_appearance_from_gui(document, key)?`.
        fn clear_selected_face_appearance_from_gui(
            &mut self,
            document: u64,
            key: &str,
        ) -> Result<bool, String> {
            Err(format!(
                "face {key:?} in document {document}: expected a selection owner"
            ))
        }
        /// Clear selected bodies' own appearance overrides in one edit.
        /// Example: `work.clear_selected_body_appearances_from_gui(document, keys)?`.
        fn clear_selected_body_appearances_from_gui(
            &mut self,
            document: u64,
            keys: &[String],
        ) -> Result<bool, String> {
            Err(format!(
                "bodies {keys:?} in document {document}: expected a selection owner"
            ))
        }
    };
}
