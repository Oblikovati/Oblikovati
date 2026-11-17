// SPDX-License-Identifier: GPL-2.0-only
//! Default inspection services for the desktop frame owner.
macro_rules! inspect_work_methods {
    () => {
        /// Apply one inspection choice; example: `work.inspect_from_gui(action)`.
        fn inspect_from_gui(
            &mut self,
            action: oblikovati_gui::GuiInspectAction,
        ) -> Result<bool, String> {
            Err(format!(
                "inspection {action:?}: expected a document with an edited sketch"
            ))
        }
        /// Measure the named selected face; example: `work.measure_selected_face_from_gui(document, key)?`.
        fn measure_selected_face_from_gui(
            &mut self,
            document: u64,
            key: &str,
        ) -> Result<bool, String> {
            Err(format!(
                "face {key:?} in document {document}: expected a selected-face measurement owner"
            ))
        }
        /// Open properties for one selected face; example: `work.open_face_properties_from_gui(document, key)?`.
        fn open_face_properties_from_gui(
            &mut self,
            document: u64,
            key: &str,
        ) -> Result<bool, String> {
            Err(format!(
                "face {key:?} in document {document}: expected a face properties owner"
            ))
        }
        /// Close face properties; example: `work.close_face_properties_from_gui()?`.
        fn close_face_properties_from_gui(&mut self) -> Result<bool, String> {
            Err("face properties: expected a face properties owner".into())
        }
    };
}
pub(crate) use inspect_work_methods;
