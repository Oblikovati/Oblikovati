// SPDX-License-Identifier: GPL-2.0-only
//! Desktop boundary for Part Modify Hole preview, creation, and editing.
macro_rules! hole_work_methods {
    () => {
        /// Dispatch a Hole Properties panel action; example: `work.hole_from_gui(action)?`.
        fn hole_from_gui(&mut self, action: oblikovati_gui::GuiHoleAction) -> Result<bool, String> {
            Err(format!("hole {action:?}: expected a certified part owner"))
        }
    };
}
pub(crate) use hole_work_methods;
