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
        /// Read current reference overlays; example: `work.datum_planes()?` before presentation.
        fn datum_planes(&mut self) -> Result<crate::HeadDatumPlaneUpdate, String> {
            Ok(crate::HeadDatumPlaneUpdate::default())
        }
        /// Toggle the display and picking of user work planes for this desktop session.
        fn set_work_plane_overlays(&mut self, _visible: bool) -> bool {
            false
        }
        /// Dispatch a Work Point command action; example: `work.work_point_from_gui(action)?`.
        fn work_point_from_gui(
            &mut self,
            action: oblikovati_gui::GuiWorkPointAction,
        ) -> Result<bool, String> {
            Err(format!("work point {action:?}: expected a datum owner"))
        }
    };
}

macro_rules! datum_view_owner_methods {
    () => {
        fn datum_planes(&mut self) -> Result<crate::HeadDatumPlaneUpdate, String> {
            self.datum_planes_for_view()
        }
        fn set_work_plane_overlays(&mut self, visible: bool) -> bool {
            let hidden = !visible;
            std::mem::replace(&mut self.work_plane_overlays_hidden, hidden) != hidden
        }
        fn work_point_from_gui(
            &mut self,
            action: oblikovati_gui::GuiWorkPointAction,
        ) -> Result<bool, String> {
            self.work_point_command(action)
        }
    };
}
pub(crate) use datum_view_owner_methods;
