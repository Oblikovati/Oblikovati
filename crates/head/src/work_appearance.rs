// SPDX-License-Identifier: GPL-2.0-only
//! Owner service for selected-body appearance clearing.
macro_rules! appearance_work_methods {
    () => {
        /// Clear selected faces' exact overrides.
        /// Example: `work.clear_selected_face_appearances_from_gui(document, keys)?`.
        fn clear_selected_face_appearances_from_gui(
            &mut self,
            document: u64,
            keys: &[String],
        ) -> Result<bool, String> {
            Err(format!(
                "faces {keys:?} in document {document}: expected a selection owner"
            ))
        }
        /// Clear mixed selected overrides in one edit.
        /// Example: `work.clear_selected_mixed_appearance_from_gui(document, bodies, faces)?`.
        fn clear_selected_mixed_appearance_from_gui(
            &mut self, document: u64, bodies: &[String], faces: &[String],
        ) -> Result<bool, String> {
            Err(format!("bodies {bodies:?}, faces {faces:?} in document {document}: expected a selection owner"))
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
