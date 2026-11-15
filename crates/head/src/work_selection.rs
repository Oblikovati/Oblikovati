// SPDX-License-Identifier: GPL-2.0-only
//! Owner services for ambient selection settings.
macro_rules! selection_work_methods {
    () => {
        /// Set the viewport selection mode; example: `work.set_selection_mode_from_gui(mode)?`.
        fn set_selection_mode_from_gui(
            &mut self,
            mode: oblikovati_gui::GuiSelectionMode,
        ) -> Result<bool, String> {
            Err(format!(
                "selection mode {mode:?}: expected a desktop selection owner"
            ))
        }
        /// Set one ambient selection category.
        /// Example: `work.set_selection_enabled_from_gui(kind, true)?`.
        fn set_selection_enabled_from_gui(
            &mut self,
            kind: oblikovati_gui::GuiSelectionKind,
            enabled: bool,
        ) -> Result<bool, String> {
            Err(format!(
                "selection category {kind:?}, enabled {enabled}: expected a desktop selection owner"
            ))
        }
        /// Change an ambient category's pick priority.
        /// Example: `work.move_selection_kind_from_gui(2, 1)?`.
        fn move_selection_kind_from_gui(&mut self, from: usize, to: usize) -> Result<bool, String> {
            Err(format!(
                "selection priority {from}->{to}: expected a desktop selection owner"
            ))
        }
        /// Restore the default selection filter; example: `work.reset_selection_filter_from_gui()?`.
        fn reset_selection_filter_from_gui(&mut self) -> Result<bool, String> {
            Err("selection filter reset: expected a desktop selection owner".into())
        }
    };
}
