// SPDX-License-Identifier: GPL-2.0-only
//! Keep fixture observations at the public geometry/history and native rendering seams.
use super::super::super::world::ObservedSketch;
use super::ApplicationWorld;

impl ApplicationWorld {
    /// Remember the actual fixture before a cancel-only command; example: `remember_edited_sketch(vec![pixel])`.
    pub fn remember_edited_sketch(&mut self, witnesses: Vec<[i32; 2]>) {
        self.observed_sketch = Some(ObservedSketch {
            geometry: self.edited_geometry(),
            history: self.edited_sketch_history(),
            witnesses,
        });
    }
    /// Check identity, constraints, history and visibility after cancellation.
    /// Example: `world.expect_edited_sketch_retained()`.
    pub fn expect_edited_sketch_retained(&mut self) {
        let current = self.edited_geometry();
        let history = self.edited_sketch_history();
        let previous = self
            .observed_sketch
            .as_ref()
            .expect("expected a remembered sketch fixture");
        assert_eq!(
            current, previous.geometry,
            "retained sketch geometry and constraints"
        );
        assert_eq!(history, previous.history, "retained committed history");
        for pixel in previous.witnesses.clone() {
            self.expect_segment_drawn(pixel, pixel);
        }
    }
    fn edited_sketch_history(&mut self) -> [Vec<String>; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let session = self
            .app()
            .owner
            .documents()
            .workspace()
            .session(edit.document)
            .unwrap();
        [session.undo_labels(), session.redo_labels()]
            .map(|labels| labels.into_iter().map(str::to_owned).collect())
    }
}
