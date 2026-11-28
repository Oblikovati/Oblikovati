// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for Part Modify Hole preview, creation, and editing.
macro_rules! hole_work_methods {
    () => {
        /// Dispatch a Hole Properties panel action; example: `work.hole_from_gui(action)?`.
        fn hole_from_gui(&mut self, action: oblikovati_gui::GuiHoleAction) -> Result<bool, String> {
            Err(format!("hole {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Slot Properties panel action; example: `work.slot_from_gui(action)?`.
        fn slot_from_gui(&mut self, action: oblikovati_gui::GuiSlotAction) -> Result<bool, String> {
            Err(format!("slot {action:?}: expected a certified part owner"))
        }
        /// Dispatch a Fillet v2 Properties panel action; example: `work.fillet_v2_from_gui(action)?`.
        fn fillet_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiFilletV2Action,
        ) -> Result<bool, String> {
            Err(format!(
                "fillet {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Chamfer v2 Properties panel action; example: `work.chamfer_v2_from_gui(action)?`.
        fn chamfer_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiChamferV2Action,
        ) -> Result<bool, String> {
            Err(format!(
                "chamfer {action:?}: expected a certified part owner"
            ))
        }
        /// Dispatch a Shell v2 Properties panel action; example: `work.shell_v2_from_gui(action)?`.
        fn shell_v2_from_gui(
            &mut self,
            action: oblikovati_gui::GuiShellV2Action,
        ) -> Result<bool, String> {
            Err(format!("shell {action:?}: expected a certified part owner"))
        }
    };
}
pub(crate) use hole_work_methods;
