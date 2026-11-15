// SPDX-License-Identifier: GPL-2.0-only
//! Datum operations on the owner-thread work contract.
macro_rules! datum_work_methods {
    () => {
        /// Create a datum; example: `work.create_offset_plane_from_gui(document, base, "50 mm")?`.
        fn create_offset_plane_from_gui(
            &mut self,
            document: u64,
            base: oblikovati_gui::GuiOffsetPlaneBase,
            offset: &str,
        ) -> Result<bool, String> {
            Err(format!("offset plane in document {document}, base {base:?}, offset {offset:?}: expected a datum owner"))
        }
        /// Validate an offset; example: `work.validate_offset_plane_from_gui(document, "50 mm")?`.
        fn validate_offset_plane_from_gui(&mut self, document: u64, offset: &str) -> Result<bool, String> {
            Err(format!("offset {offset:?} in document {document}: expected a datum owner"))
        }
    };
}
