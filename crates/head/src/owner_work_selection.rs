// SPDX-License-Identifier: GPL-2.0-only
//! Viewport selection methods on the desktop work boundary.
macro_rules! desktop_selection_work {
    () => {
        fn face_edges_on_line(
            &mut self,
            line: [[f64; 3]; 2],
        ) -> Result<Vec<[[f64; 3]; 2]>, String> {
            self.edges_of_face_on_line(line)
        }
        fn click_model_from_gui(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
        ) -> Result<bool, String> {
            self.click_model(document, line)
        }
        fn click_model_with_window(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
            window: f64,
        ) -> Result<bool, String> {
            self.click_model_with_window(document, line, Some(window))
        }
        fn click_model_with_modifiers(
            &mut self,
            document: u64,
            line: [[f64; 3]; 2],
            window: Option<f64>,
            toggle: bool,
        ) -> Result<bool, String> {
            self.click_model_with_modifiers(document, line, window, toggle)
        }
        fn selected_browser_outline(&mut self) -> Vec<[[f64; 3]; 2]> {
            self.current_browser_outline()
        }
        fn selected_edge_outlines(&mut self) -> Vec<Vec<[[f64; 3]; 2]>> {
            self.current_selected_edge_outlines()
        }
    };
}
