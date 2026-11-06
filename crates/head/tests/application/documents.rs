// SPDX-License-Identifier: GPL-2.0-only
//! Actions and readings shared by the document scenarios: they use real input and rendered output.
use super::{ApplicationWorld, diagnostics};
use crate::HeadWork;
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};
const TAB_STRIP: &str = "##open-document-tabs";
/// A tab ends this far past its label; further marks belong to the next tab.
const CLOSE_MARK_REACH: f32 = 30.;
/// Two viewport corners well away from the browser and the ribbon.
const RECTANGLE: [[i32; 2]; 2] = [[760, 380], [1060, 560]];
/// The viewport rectangle a scenario reads, clear of the browser and tabs.
pub(super) const VIEWPORT_X: [usize; 2] = [400, 1400];
pub(super) const VIEWPORT_Y: [usize; 2] = [250, 650];

impl ApplicationWorld {
    pub fn state(&mut self) -> oblikovati_gui::GuiCommandState {
        self.app().owner.gui_command_state().unwrap()
    }
    pub fn active_id(&mut self) -> u64 {
        self.state().document.expect("expected an active document")
    }
    pub fn document_name(&mut self, id: u64) -> String {
        let state = self.state();
        let row = state.documents.iter().find(|row| row.id == id);
        row.unwrap_or_else(|| panic!("document {id}: expected an open document"))
            .name
            .clone()
    }
    pub fn document_is_open(&mut self, id: u64) -> bool {
        self.state().documents.iter().any(|row| row.id == id)
    }
    /// Setup only: a Given's document is not what its scenario measures.
    pub fn add_part(&mut self) -> u64 {
        self.app().owner.create_part_document().unwrap();
        let id = self.active_id();
        self.parts.push(id);
        self.pump(Vec::new());
        id
    }
    pub fn scratch_folder(&mut self) -> std::path::PathBuf {
        let scratch = self
            .scratch
            .get_or_insert_with(|| tempfile::tempdir().unwrap());
        scratch.path().to_path_buf()
    }
    /// Setup only: bind a document to a file without driving the dialog it does not measure.
    pub fn save_part_to_file(&mut self, id: u64, name: &str) -> std::path::PathBuf {
        let path = self.scratch_folder().join(name);
        self.app()
            .owner
            .complete_save_path(id, Some(&path))
            .unwrap();
        self.pump(Vec::new());
        assert!(
            path.exists(),
            "saved part {path:?}: expected a file on disk"
        );
        path
    }
    pub fn command_key(&mut self, key: ViewportPhysicalKey, command: ViewportKey) {
        let modifiers = oblikovati_viewport::ViewportModifiers {
            control: true,
            ..Default::default()
        };
        let event = |pressed, modifiers| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key: key,
                pressed,
                command: Some(command),
                modifiers,
            })
        };
        self.pump(vec![event(true, modifiers)]);
        self.pump(vec![event(false, Default::default())]);
    }
    pub fn press_save_shortcut(&mut self) {
        self.command_key(ViewportPhysicalKey::S, ViewportKey::Save);
    }
    pub fn escape(&mut self) {
        self.key(ViewportPhysicalKey::Escape, false);
    }
    pub fn type_text(&mut self, text: &str) {
        let events = text
            .chars()
            .map(|letter| ViewportEvent::Input(ViewportInput::Text(letter)))
            .collect();
        self.pump(events);
    }
    pub fn hover(&mut self, position: [i32; 2]) {
        self.pump(vec![ViewportEvent::Input(ViewportInput::Motion {
            position,
            modifiers: Default::default(),
        })]);
    }
    pub fn right_click(&mut self, position: [i32; 2]) {
        self.hover(position);
        let press = |pressed| {
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Right,
                pressed,
                position,
                modifiers: Default::default(),
            })
        };
        self.pump(vec![press(true)]);
        self.pump(vec![press(false)]);
    }
    /// Replace an input field's text through the keyboard, then press Enter.
    pub fn enter_in_field(&mut self, label: &str, text: &str) {
        let run = self.run(label).clone();
        let field = [run.maximum[0] as i32 + 60, run.center()[1] as i32];
        self.click(field);
        self.type_text(text);
        self.key(ViewportPhysicalKey::Enter, false);
    }
    /// The tab's own label: the same name also appears in the browser and in dialogs.
    pub fn tab_run(&self, name: &str) -> Option<diagnostics::RenderedRun> {
        let (position, size) = diagnostics::window_geometry(TAB_STRIP)?;
        self.frame
            .runs
            .iter()
            .find(|run| {
                let center = run.center();
                run.text.starts_with(name)
                    && center[1] >= position[1]
                    && center[1] <= position[1] + size[1]
                    && center[0] >= position[0]
                    && center[0] <= position[0] + size[0]
            })
            .cloned()
    }
    pub fn expect_tab(&self, name: &str) -> diagnostics::RenderedRun {
        self.tab_run(name).unwrap_or_else(|| {
            panic!(
                "rendered text {:?}: expected a tab named {name:?}",
                self.text()
            )
        })
    }
    pub fn click_tab(&mut self, id: u64) {
        let name = self.document_name(id);
        let center = self.expect_tab(&name).center();
        self.click(center.map(|value| value as i32));
    }
    pub fn open_tab_menu(&mut self, id: u64) {
        let name = self.document_name(id);
        let center = self.expect_tab(&name).center();
        self.right_click(center.map(|value| value as i32));
    }
    /// Find the drawn close mark to the right of the label and click it: hovering shows it.
    pub fn click_close_mark(&mut self, id: u64) {
        let name = self.document_name(id);
        let center = self.expect_tab(&name).center();
        self.hover(center.map(|value| value as i32));
        let run = self.expect_tab(&name);
        let marks: Vec<[f32; 2]> = self
            .frame
            .marks
            .iter()
            .filter(|mark| {
                mark.list == run.list
                    && mark.position[0] > run.maximum[0]
                    && mark.position[0] < run.maximum[0] + CLOSE_MARK_REACH
                    && mark.position[1] >= run.minimum[1] - 2.
                    && mark.position[1] <= run.maximum[1] + 2.
            })
            .map(|mark| mark.position)
            .collect();
        assert!(
            !marks.is_empty(),
            "tab {name:?}: expected a drawn close mark"
        );
        let mean =
            |axis: usize| marks.iter().map(|mark| mark[axis]).sum::<f32>() / marks.len() as f32;
        self.click([mean(0) as i32, mean(1) as i32]);
    }
    /// Close the active document with the mark a user clicks.
    pub fn close_active_tab(&mut self) {
        let id = self.active_id();
        self.click_close_mark(id);
    }
    pub fn start_sketch(&mut self) {
        self.click_text("Sketch XY");
        assert!(
            self.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch,
            "Sketch XY: expected the sketch environment"
        );
    }
    pub fn draw_rectangle(&mut self) {
        self.click_text("Rectangle");
        for corner in RECTANGLE {
            self.click(corner);
        }
    }
    pub fn finish_sketch(&mut self) {
        self.click_text("Finish Sketch");
        assert!(
            self.state().authoring == oblikovati_gui::GuiAuthoringContext::Part,
            "Finish Sketch: expected the part environment"
        );
    }
    /// A finished sketch through the ribbon, so the document is modified the way a user does it.
    pub fn add_finished_sketch(&mut self) {
        self.start_sketch();
        self.finish_sketch();
    }
    pub fn browser_sketch_count(&mut self) -> usize {
        self.state()
            .browser
            .map_or(0, |browser| browser.sketches.len())
    }
    /// Assert the browser shows this document: its root names it and it lists its sketches.
    pub fn expect_browser_of(&mut self, id: u64) {
        let name = self.document_name(id);
        let (position, size) = diagnostics::window_geometry("Model").expect("expected the browser");
        let shown = self.frame.runs.iter().any(|run| {
            let center = run.center();
            run.text.contains(&name)
                && center[0] >= position[0]
                && center[0] <= position[0] + size[0]
                && center[1] >= position[1]
                && center[1] <= position[1] + size[1]
        });
        assert!(
            shown,
            "rendered text {:?}: expected {name:?} in the browser",
            self.text()
        );
        self.expect_text("Origin");
    }
    /// Assert the sketch is listed: rendered text, against the owner's row name.
    pub fn expect_browser_sketch(&mut self) {
        let browser = self.state().browser.expect("expected a browser");
        let sketch = browser.sketches.first().expect("expected a browser sketch");
        self.expect_text(&sketch.name.clone());
    }
    pub fn expect_active(&mut self, id: u64) {
        assert_eq!(
            self.state().document,
            Some(id),
            "expected document {id} active"
        );
        self.expect_browser_of(id);
    }
    /// Assert something is drawn over the viewport: its empty background has uniform rows.
    pub fn expect_drawn_region(&mut self) {
        let capture = super::capture_framebuffer(&mut self.app().window.native);
        let width = capture.extent[0] as usize;
        let drawn = capture
            .rgba
            .chunks_exact(width * 4)
            .skip(VIEWPORT_Y[0])
            .take(VIEWPORT_Y[1] - VIEWPORT_Y[0])
            .any(|row| {
                let span = &row[VIEWPORT_X[0] * 4..VIEWPORT_X[1] * 4];
                span.chunks_exact(4).any(|pixel| pixel != &span[..4])
            });
        assert!(
            drawn,
            "viewport: expected drawn geometry over the background"
        );
    }
    fn pixel_at(&mut self, position: [usize; 2]) -> [u8; 4] {
        let capture = super::capture_framebuffer(&mut self.app().window.native);
        let at = (position[1] * capture.extent[0] as usize + position[0]) * 4;
        std::array::from_fn(|channel| capture.rgba[at + channel])
    }
    /// A selected radio button fills its centre; its rim keeps the frame colour.
    pub fn radio_selected(&mut self, label: &str) -> bool {
        let run = self.run(label).clone();
        let y = run.center()[1] as usize;
        let centre = run.minimum[0] as usize - 14;
        self.pixel_at([centre, y]) != self.pixel_at([centre - 8, y])
    }
    /// Selected text sits on a highlight: the most common colour behind it differs from the field.
    pub fn text_is_selected(&mut self, text: &str) -> bool {
        let run = self.run(text).clone();
        let y = run.center()[1] as usize;
        let field = self.pixel_at([run.minimum[0] as usize - 3, y]);
        let mut counts = std::collections::BTreeMap::new();
        for x in run.minimum[0] as usize..run.maximum[0] as usize {
            *counts.entry(self.pixel_at([x, y])).or_insert(0) += 1;
        }
        let common = counts.into_iter().max_by_key(|(_, count)| *count);
        common.is_some_and(|(pixel, _)| pixel != field)
    }
    /// Choose a file through File, Open and the listing a user sees.
    pub fn reopen_through_dialog(&mut self, path: &std::path::Path) {
        self.click_text("File");
        self.click_text("Open...");
        let folder = path.parent().unwrap().to_str().unwrap().to_string();
        self.enter_in_field("Look in", &folder);
        self.click_text(path.file_name().unwrap().to_str().unwrap());
        self.click_text("Open");
        self.expect_no_text("Look in");
    }
    pub fn activate_through_tab(&mut self, id: u64) {
        self.click_tab(id);
        assert_eq!(
            self.state().document,
            Some(id),
            "tab click: expected document {id} active"
        );
    }
    /// Accept Save As in the scenario's own folder, so no test writes beside the sources.
    pub fn accept_save_as(&mut self) -> std::path::PathBuf {
        let folder = self.scratch_folder();
        self.enter_in_field("Save in", folder.to_str().unwrap());
        self.click_text("Save");
        self.expect_no_text("Save in");
        let path = std::fs::read_dir(&folder)
            .unwrap()
            .map(|entry| entry.unwrap().path())
            .max_by_key(|path| std::fs::metadata(path).unwrap().modified().unwrap())
            .expect("Save As: expected a file in the chosen folder");
        self.saved_path = Some(path.clone());
        path
    }
}
