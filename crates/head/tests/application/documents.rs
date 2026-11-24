// SPDX-License-Identifier: GPL-2.0-only
//! Actions and readings shared by the document scenarios: they use real input and rendered output.
use super::{ApplicationWorld, diagnostics, world::squeezed};
use crate::HeadWork;
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};
const TAB_STRIP: &str = "##open-document-tabs";
/// A tab ends this far past its label; further marks belong to the next tab.
const CLOSE_MARK_REACH: f32 = 30.;
/// Two viewport corners well away from the browser and the ribbon.
pub(super) const RECTANGLE: [[i32; 2]; 2] = [[760, 380], [1060, 560]];
/// The viewport rectangle a scenario reads, clear of the browser and tabs.
/// Between the browser and the orientation control at the view's right.
pub(super) const VIEWPORT_X: [usize; 2] = [400, 1290];
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
    /// Setup only: a document through the owner command the chooser itself sends.
    pub fn add_template(&mut self, kind: oblikovati_gui::GuiTemplateKind) -> u64 {
        let template = oblikovati_gui::GuiDocumentTemplate {
            kind,
            units: oblikovati_gui::GuiTemplateUnits::Metric,
        };
        self.app().owner.create_document_template(template).unwrap();
        self.pump(Vec::new());
        self.active_id()
    }
    pub fn add_drawing(&mut self) -> u64 {
        self.add_template(oblikovati_gui::GuiTemplateKind::Drawing)
    }
    /// Supply an editable sketch for a generic Given while retaining any named fixture.
    /// For example, `world.ensure_editable_sketch()` keeps an already edited YZ sketch.
    pub fn ensure_editable_sketch(&mut self) {
        if self.app().owner.active_sketch_edit().is_none() {
            self.start_empty_sketch();
        }
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
    /// Control+Shift plus a key, sent as one press with its modifiers.
    pub fn control_shift_key(&mut self, key: ViewportPhysicalKey) {
        let modifiers = oblikovati_viewport::ViewportModifiers {
            control: true,
            shift: true,
            ..Default::default()
        };
        let event = |pressed, modifiers| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key: key,
                pressed,
                command: None,
                modifiers,
            })
        };
        self.pump(vec![event(true, modifiers)]);
        self.pump(vec![event(false, Default::default())]);
    }
    pub fn press_save_shortcut(&mut self) {
        self.command_key(ViewportPhysicalKey::S, ViewportKey::Save);
    }
    /// As the window system reports it: the physical key carrying its Cancel command.
    pub fn escape(&mut self) {
        #[cfg(target_os = "linux")]
        if self.navigation_wheel_scenario() {
            super::x11_input::press_keysym(0xff1b);
            self.pump(Vec::new());
            return;
        }
        let event = |pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Escape,
                layout_key: ViewportPhysicalKey::Escape,
                pressed,
                command: Some(ViewportKey::Cancel),
                modifiers: Default::default(),
            })
        };
        self.pump(vec![event(true), event(false)]);
    }
    pub fn type_text(&mut self, text: &str) {
        let events = text
            .chars()
            .map(|letter| ViewportEvent::Input(ViewportInput::Text(letter)))
            .collect();
        self.pump(events);
    }
    pub fn hover(&mut self, position: [i32; 2]) {
        #[cfg(target_os = "linux")]
        if self.navigation_wheel_scenario() || self.project_dialog_native_input() {
            super::x11_input::move_pointer(position);
            self.pump(Vec::new());
            return;
        }
        self.pump(vec![ViewportEvent::Input(ViewportInput::Motion {
            position,
            modifiers: Default::default(),
        })]);
    }
    pub fn right_click(&mut self, position: [i32; 2]) {
        #[cfg(target_os = "linux")]
        if self.navigation_wheel_scenario() {
            super::x11_input::right_click(position);
            self.pump(Vec::new());
            return;
        }
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
        let name = super::world::squeezed(name);
        self.frame
            .runs
            .iter()
            .find(|run| {
                let center = run.center();
                run.text.starts_with(&name)
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
        self.click_mark_after(&run);
    }
    /// Close a panel tab by its label; the topmost run with these words is the tab.
    pub fn click_run_close_mark(&mut self, label: &str) {
        let center = self.run(label).center();
        self.hover(center.map(|value| value as i32));
        let run = self.run(label).clone();
        self.click_mark_after(&run);
    }
    fn click_mark_after(&mut self, run: &diagnostics::RenderedRun) {
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
            "tab {run:?}: expected a drawn close mark"
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
        self.click_text("Start 2D Sketch");
        self.pump(Vec::new());
        self.click_text("XY Plane");
        self.pump(Vec::new());
        assert!(
            self.state().authoring == oblikovati_gui::GuiAuthoringContext::Sketch,
            "Start 2D Sketch: expected the sketch environment"
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
        let rendered_name = squeezed(&name);
        let (position, size) = diagnostics::window_geometry("Model").expect("expected the browser");
        let shown = self.frame.runs.iter().any(|run| {
            let center = run.center();
            squeezed(&run.text).contains(&rendered_name)
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
                span.as_chunks::<4>()
                    .0
                    .iter()
                    .any(|pixel| pixel[..] != span[..4])
            });
        assert!(
            drawn,
            "viewport: expected drawn geometry over the background"
        );
    }
    /// The active document, without asserting which of its windows is in front.
    pub fn expect_active_by_id(&mut self, id: u64) {
        assert_eq!(self.active_id(), id, "active document");
    }
    pub fn pixel_at(&mut self, position: [usize; 2]) -> [u8; 4] {
        let capture = super::capture_framebuffer(&mut self.app().window.native);
        let at = (position[1] * capture.extent[0] as usize + position[0]) * 4;
        std::array::from_fn(|channel| capture.rgba[at + channel])
    }
    /// A selected radio button fills its centre; its rim keeps the frame colour.
    pub fn radio_selected(&mut self, label: &str) -> bool {
        let run = self.run(label).clone();
        let y = run.center()[1] as usize;
        let centre = run.minimum[0] as usize - 14;
        let capture = super::capture_framebuffer(&mut self.app().window.native);
        let sample = |x: usize| {
            let offset = (y * capture.extent[0] as usize + x) * 4;
            &capture.rgba[offset..offset + 4]
        };
        sample(centre) != sample(centre - 8)
    }
    /// Selected text sits on a highlight: the most common colour behind it differs from the field.
    pub fn text_is_selected(&mut self, text: &str) -> bool {
        let run = self.run(text).clone();
        self.text_run_is_selected(&run)
    }

    /// Check a particular run when more than one field has the same value.
    pub fn text_run_is_selected(&mut self, run: &diagnostics::RenderedRun) -> bool {
        let y = run.center()[1] as usize;
        let field = self.pixel_at([run.minimum[0] as usize - 3, y]);
        let mut counts = std::collections::BTreeMap::new();
        for x in run.minimum[0] as usize..run.maximum[0] as usize {
            *counts.entry(self.pixel_at([x, y])).or_insert(0) += 1;
        }
        let common = counts.into_iter().max_by_key(|(_, count)| *count);
        common.is_some_and(|(pixel, _)| pixel != field)
    }
    /// A selected row's accent reaches past its text; the row's width shows it without the glyphs.
    pub fn row_highlighted(&mut self, text: &str) -> bool {
        let run = self.run(text).clone();
        let x = run.maximum[0] as usize + 20;
        let y = run.center()[1] as usize;
        // Far below any drawing tree row, the same column shows the plain panel.
        self.pixel_at([x, y]) != self.pixel_at([x, 700])
    }
    /// A row's accent against another row of the same list, which reads plain.
    pub fn row_accented_beside(&mut self, text: &str, plain: &str) -> bool {
        let row = self.run(text).clone();
        let other = self.run(plain).center()[1] as usize;
        let x = row.maximum[0].max(self.run(plain).maximum[0]) as usize + 6;
        self.pixel_at([x, row.center()[1] as usize]) != self.pixel_at([x, other])
    }
    /// The last part of a row's text, where a match at its end is marked.
    pub fn end_of_text_is_marked(&mut self, text: &str) -> bool {
        let run = self.run(text).clone();
        let y = run.center()[1] as usize;
        let start = (run.minimum[0] + (run.maximum[0] - run.minimum[0]) * 0.6) as usize;
        let plain = self.pixel_at([run.minimum[0] as usize - 3, y]);
        let mut counts = std::collections::BTreeMap::new();
        for x in start..run.maximum[0] as usize {
            *counts.entry(self.pixel_at([x, y])).or_insert(0) += 1;
        }
        let common = counts.into_iter().max_by_key(|(_, count)| *count);
        common.is_some_and(|(pixel, _)| pixel != plain)
    }
    /// Press a branch's disclosure control, drawn just left of its label.
    pub fn press_disclosure(&mut self, branch: &str) {
        let label = self.run(branch).clone();
        let at = [label.minimum[0] as i32 - 12, label.center()[1] as i32];
        self.click(at);
        self.pump(Vec::new());
    }
    /// The location steps as rendered, without spaces; no other text in a dialog holds `>`.
    pub fn breadcrumb_text(&self) -> String {
        let dialog_list = self.run("Save in").list;
        self.frame
            .runs
            .iter()
            .find(|run| run.list == dialog_list && run.text.contains('>'))
            .unwrap_or_else(|| panic!("expected a breadcrumb in rendered text {:?}", self.text()))
            .text
            .clone()
    }
    /// Open a known fixture by its full path; example: `world.open_fixture(&path)`.
    pub fn open_fixture(&mut self, path: &std::path::Path) {
        self.click_text("File");
        self.click_text("Open...");
        #[cfg(target_os = "linux")]
        if self.navigation_wheel_scenario() {
            for character in path.to_str().expect("fixture path: expected UTF-8").chars() {
                super::x11_input::press_path_character(character);
                self.pump(Vec::new());
            }
            super::x11_input::press_keysym(0xff0d);
            self.pump(Vec::new());
            self.expect_no_text("Look in");
            return;
        }
        self.type_text(path.to_str().expect("fixture path: expected UTF-8"));
        self.key(ViewportPhysicalKey::Enter, false);
        self.pump(Vec::new());
        self.expect_no_text("Look in");
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
