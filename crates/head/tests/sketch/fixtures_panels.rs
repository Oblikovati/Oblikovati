// SPDX-License-Identifier: GPL-2.0-only
//! The floating panels sketch scenarios drive: Precise Input, Application Options, Undo history.
use super::*;
impl ApplicationWorld {
    /// Click a toolbar field; its label sits right of it and may share a run with its value.
    pub fn click_precise_field(&mut self, label: &str) {
        let (position, size) = diagnostics::window_geometry("Precise Input")
            .expect("expected the Precise Input toolbar");
        let run = self
            .frame
            .runs
            .iter()
            .find(|run| {
                let center = run.center();
                run.text.ends_with(label)
                    && (0..2).all(|axis| {
                        center[axis] >= position[axis]
                            && center[axis] <= position[axis] + size[axis]
                    })
            })
            .unwrap_or_else(|| panic!("expected a field {label} in {:?}", self.text()))
            .clone();
        let label_width = LABEL_GLYPH_WIDTH * label.len() as f32;
        let x = run.maximum[0] - label_width - FIELD_INSIDE_LABEL;
        self.click([x as i32, run.center()[1] as i32]);
    }
    /// Type both of the toolbar's values and accept them, whatever its fields are named.
    pub fn precise_values(&mut self, first: &str, second: &str) {
        let labels = self.precise_field_labels();
        self.click_precise_field(labels[0]);
        self.type_text(first);
        self.click_precise_field(labels[1]);
        self.type_text(second);
        self.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
        self.pump(Vec::new());
    }
    fn precise_field_labels(&self) -> [&'static str; 2] {
        let text = self.window_text("Precise Input");
        if text.contains("Distance") {
            ["Distance", "Angle"]
        } else if !text.contains("Angle") {
            ["X", "Y"]
        } else if text.contains('Y') {
            ["Y", "Angle"]
        } else {
            ["X", "Angle"]
        }
    }
    /// Open the toolbar through the Create panel while a command takes points.
    pub fn open_precise_input(&mut self) {
        super::super::super::ribbon_panel_expansion::click_expansion_arrow(self, "Create");
        self.pump(Vec::new());
        let button = self
            .run_in_window("##ribbon-additional-Create", "PreciseInput")
            .center();
        self.click(diagnostics::pointer_pixels(button));
        self.pump(Vec::new());
    }
    /// Close it through its own options, the way the recording did.
    pub fn close_precise_input(&mut self) {
        let options = self
            .run_in_window("Precise Input", "...")
            .center()
            .map(|value| value as i32);
        self.click(options);
        self.pump(Vec::new());
        self.click_text("Close");
        self.pump(Vec::new());
        assert!(
            diagnostics::window_geometry("Precise Input").is_none(),
            "expected the Precise Input toolbar to close"
        );
    }
    /// Type one exact point, in millimetres, and accept it.
    pub fn precise_point(&mut self, at: [f64; 2]) {
        self.precise_values(&format!("{:.9}", at[0]), &format!("{:.9}", at[1]));
    }
    /// The text run inside one named window, for a word the ribbon also shows.
    pub fn run_in_window(&self, title: &str, text: &str) -> &diagnostics::RenderedRun {
        let (position, size) = diagnostics::window_geometry(title)
            .unwrap_or_else(|| panic!("expected a window {title:?}"));
        self.frame
            .runs
            .iter()
            .filter(|run| run.text == text)
            .find(|run| {
                let center = run.center();
                (0..2).all(|axis| {
                    center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis]
                })
            })
            .unwrap_or_else(|| {
                let shown = self.window_text(title);
                panic!("expected {text:?} in window {title:?}, which shows {shown:?}")
            })
    }
    pub fn open_application_options(&mut self) {
        self.click_text("Tools");
        self.pump(Vec::new());
        self.click_text("Application Options...");
        self.pump(Vec::new());
    }
    pub fn close_application_options(&mut self) {
        self.escape();
        self.pump(Vec::new());
    }
    /// Right-click on Undo lists what it would undo, a list only that popup shows.
    pub fn open_undo_history(&mut self) {
        let undo = self.run("Undo").center().map(|value| value as i32);
        self.right_click(undo);
        self.pump(Vec::new());
    }
    pub fn undo_history_rows(&self) -> Vec<String> {
        let mut rows: Vec<&diagnostics::RenderedRun> = self
            .frame
            .runs
            .iter()
            .filter(|run| run.text.starts_with("Create") && run.text.len() > "Create".len())
            .collect();
        rows.sort_by(|first, second| first.center()[1].total_cmp(&second.center()[1]));
        rows.into_iter().map(|run| run.text.clone()).collect()
    }
}
