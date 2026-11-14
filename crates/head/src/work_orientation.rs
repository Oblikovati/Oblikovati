// SPDX-License-Identifier: GPL-2.0-only
//! Orientation services on the owner-thread work contract.
macro_rules! orientation_work_methods {
    () => {
        /// Set the saved orientation selection lock; example: `work.set_orientation_lock_from_gui(true)?`.
        fn set_orientation_lock_from_gui(&mut self, locked: bool) -> Result<bool, String> {
            Err(format!(
                "orientation lock {locked}: expected a desktop preference owner"
            ))
        }
        /// Resolve the current selected model entity to an analytic bounds center.
        /// Example: `work.orientation_selection_pivot()?` before a cube snap.
        fn orientation_selection_pivot(&mut self) -> Result<Option<[f64; 3]>, String> {
            Err("orientation selection pivot: expected a desktop selection owner".into())
        }

        /// Read exact bounds of visible bodies for a selection-centered fit.
        /// Example: `work.orientation_fit_bounds_from_gui()?` after resolving the pivot.
        fn orientation_fit_bounds_from_gui(&mut self) -> Result<Vec<[[f64; 3]; 2]>, String> {
            Err("selection fit bounds: expected a desktop geometry owner".into())
        }
    };
}
