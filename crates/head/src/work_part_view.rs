// SPDX-License-Identifier: GPL-2.0-only
//! Live display operations for part view history.
macro_rules! part_view_work_methods {
    () => {
        /// Move a history cursor with its live display; example: `work.jump_history_with_window(document, 0, window)?`.
        fn jump_history_with_window(
            &mut self,
            document: u64,
            position: i64,
            _: &mut dyn crate::HeadWindow,
        ) -> Result<bool, String> {
            self.jump_history_from_gui(document, position)
        }
        /// Apply a part-view choice with its live display; example: `work.part_view_choice_from_gui(document, choice, window)?`.
        fn part_view_choice_from_gui(
            &mut self,
            document: u64,
            choice: &oblikovati_gui::GuiBrowserChoice,
            _: &mut dyn crate::HeadWindow,
        ) -> Result<bool, String> {
            Err(format!(
                "part view choice {choice:?}, document {document}: expected a display owner"
            ))
        }
        /// Route input with a mutable display for view-history commands; example: `work.process_live_window_inputs(&events, window)?`.
        fn process_live_window_inputs(
            &mut self,
            inputs: &[oblikovati_viewport::ViewportEvent],
            window: &mut dyn crate::HeadWindow,
        ) -> Result<bool, String> {
            self.process_window_inputs(inputs, window)
        }
    };
}
pub(crate) use part_view_work_methods;
