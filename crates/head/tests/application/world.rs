// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWindow;
#[path = "workspace_profile.rs"]
mod workspace_profile;
#[derive(cucumber::World)]
#[world(init = Self::new)]
pub(super) struct ApplicationWorld {
    pub app: Option<NativeApplication>,
    _preferences: tempfile::TempDir,
    pub frame: diagnostics::RenderedFrame,
    capture_namespace: Option<String>,
    pub scenario_feature_path: Option<std::path::PathBuf>,
    pub previous_document: Option<u64>,
    #[cfg(target_os = "linux")]
    pub window_frame_before: Option<[i32; 4]>,
    #[cfg(target_os = "linux")]
    pub window_frame_handle: Option<String>,
    pub observed_view_camera: Option<oblikovati_viewport::ViewportCamera>,
    pub orientation_hover: Option<super::viewport_orientation::hover::HoverObservation>,
    #[cfg(target_os = "linux")]
    pub orientation_hud: Vec<super::viewport_orientation::hud_contract::HudSample>,
    /// The picked face boundary before a Zoom Selected operation.
    pub observed_zoom_face_edges: Vec<[[f64; 3]; 2]>,
    pub ribbon_height: f32,
    pub ribbon_layout_samples: Vec<super::ribbon_display::layout_contract::RibbonSample>,
    pub panel_selector_focus: Option<u32>,
    pub modify_title_anchor: Option<[f32; 2]>,
    pub work_features_x_before_explore_toggle: Option<f32>,
    pub saved_path: Option<std::path::PathBuf>,
    pub bytes_before_save: Option<Vec<u8>>,
    pub scratch: Option<tempfile::TempDir>,
    pub parts: Vec<u64>,
    pub other_document: Option<u64>,
    pub closed_document: Option<u64>,
    pub application_close_requested: bool,
    pub workspace_color: Option<[u8; 4]>,
    pub compact_options_idle_color: Option<[u8; 4]>,
    pub initial_breadcrumb: Option<String>,
    pub proposed_name: Option<String>,
    pub browser_width: Option<f32>,
    pub strip_left: Option<f32>,
    pub dragged_tab_x: Option<f32>,
    /// The edited sketch's view with nothing drawn, for what a scenario later added or removed.
    pub sketch_baseline: Option<oblikovati_viewport::ViewportCapture>,
    pub observed_counts: Vec<usize>,
    pub observed_document_windows: Vec<oblikovati_gui::GuiDocumentWindow>,
    pub observed_sketch: Option<ObservedSketch>,
    /// Published ellipse center, first-axis point, and perimeter point in sketch units.
    pub ellipse_fixture_points: Option<[[f64; 2]; 3]>,
    pub ellipse_alias_center: Option<[f64; 2]>,
    pub curvature_initial: Option<CurvatureObservation>,
    pub curvature_applied: Option<CurvatureObservation>,
    pub text_creation_baseline: Option<oblikovati_model::sketch::Sketch2D>,
    /// The empty editor before entering content that the scenario will cancel.
    pub text_defaults_before_cancel: Option<oblikovati_gui::GuiSketchTextDraft>,
    pub observed_row_gap: Option<f32>,
    pub open_dialog_restore_bounds: Option<([f32; 2], [f32; 2])>,
    pub moved_customize_origin: Option<[f32; 2]>,
    pub customize_size_before_drag: Option<[[f32; 2]; 3]>,
    pub shell_button_texture: Option<usize>,
    pub observed_row_plain: Option<[u8; 4]>,
    pub observed_view: Option<[i32; 2]>,
    pub last_capture: Option<oblikovati_viewport::ViewportCapture>,
    #[cfg(target_os = "linux")]
    pub extrusion_edit_distances: Option<[f64; 2]>,
    pub observed_second_view: Option<[i32; 2]>,
    pub observed_annotations: Option<bool>,
    /// Text read before a later step changes what the window shows.
    pub observed_rows: Option<String>,
    pub detached_drawing_menu: bool,
    /// Quick-access choices captured before dismissing their menu.
    pub observed_quick_access: Option<Vec<(String, bool)>>,
    /// The toolbar icon selected for a tooltip scenario.
    pub observed_toolbar_command: Option<String>,
    /// Viewport and panel width before a docking-boundary resize.
    pub docking_resize: Option<([u32; 4], f32)>,
    pub docking_node: Option<u32>,
    pub docking_active_tab: Option<bool>,
    pub docking_selected_tab: Option<String>,
    pub floating_appearance_position: Option<[f32; 2]>,
    /// The latest drag, for a step that checks the preview it left.
    pub last_drag: Option<[[i32; 2]; 2]>,
    /// The dimension value last opened, for a step that reopens "the annotation".
    pub opened_value: Option<String>,
}
pub(super) struct NativeApplication {
    pub window: ScriptedGuiWindow,
    pub owner: NativeDesktopOwner,
}
impl fmt::Debug for ApplicationWorld {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter
            .debug_struct("ApplicationWorld")
            .field("text", &self.text())
            .finish_non_exhaustive()
    }
}
impl ApplicationWorld {
    pub(super) fn new() -> Self {
        Self::open_profile(tempfile::tempdir().unwrap())
    }
    fn open_profile(preferences: tempfile::TempDir) -> Self {
        let mut native = NativeHeadWindow::open().unwrap();
        native.ribbon_settings_directory = Some(preferences.path().to_owned());
        let owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
        let mut world = Self {
            app: Some(NativeApplication {
                window: scripted_window(native),
                owner,
            }),
            _preferences: preferences,
            frame: Default::default(),
            capture_namespace: None,
            scenario_feature_path: None,
            previous_document: None,
            #[cfg(target_os = "linux")]
            window_frame_before: None,
            #[cfg(target_os = "linux")]
            window_frame_handle: None,
            observed_view_camera: None,
            observed_zoom_face_edges: Vec::new(),
            ribbon_height: 0.,
            ribbon_layout_samples: Vec::new(),
            panel_selector_focus: None,
            modify_title_anchor: None,
            work_features_x_before_explore_toggle: None,
            saved_path: None,
            bytes_before_save: None,
            scratch: None,
            parts: Vec::new(),
            other_document: None,
            closed_document: None,
            application_close_requested: false,
            workspace_color: None,
            compact_options_idle_color: None,
            initial_breadcrumb: None,
            proposed_name: None,
            browser_width: None,
            strip_left: None,
            dragged_tab_x: None,
            sketch_baseline: None,
            observed_counts: Vec::new(),
            orientation_hover: None,
            #[cfg(target_os = "linux")]
            orientation_hud: Vec::new(),
            observed_document_windows: Vec::new(),
            observed_sketch: None,
            ellipse_fixture_points: None,
            ellipse_alias_center: None,
            curvature_initial: None,
            curvature_applied: None,
            text_creation_baseline: None,
            text_defaults_before_cancel: None,
            observed_row_gap: None,
            open_dialog_restore_bounds: None,
            moved_customize_origin: None,
            customize_size_before_drag: None,
            shell_button_texture: None,
            observed_row_plain: None,
            observed_view: None,
            last_capture: None,
            #[cfg(target_os = "linux")]
            extrusion_edit_distances: None,
            observed_second_view: None,
            observed_annotations: None,
            observed_rows: None,
            detached_drawing_menu: false,
            observed_quick_access: None,
            observed_toolbar_command: None,
            docking_resize: None,
            docking_node: None,
            docking_active_tab: None,
            docking_selected_tab: None,
            floating_appearance_position: None,
            last_drag: None,
            opened_value: None,
        };
        world.pump(Vec::new());
        world
    }
    pub fn record_initial_sketch(&mut self) {
        self.observed_sketch = Some(ObservedSketch {
            geometry: self.edited_geometry(),
            history: Default::default(),
            witnesses: Vec::new(),
        });
    }
    pub fn app(&mut self) -> &mut NativeApplication {
        self.app
            .as_mut()
            .expect("expected an active native scenario")
    }
    pub fn pump(&mut self, events: Vec<ViewportEvent>) {
        let app = self.app();
        app.window.scripted = events;
        crate::frames::run_frames(
            &mut app.window,
            HeadOptions { frames: Some(3) },
            &mut app.owner,
        )
        .unwrap();
        self.frame = diagnostics::rendered_frame();
    }
    /// Identify wheel scenarios; example: `world.navigation_wheel_scenario()`.
    pub(super) fn navigation_wheel_scenario(&self) -> bool {
        super::scenario_capture::is_navigation_wheel(self.scenario_feature_path.as_deref())
    }
    pub fn click(&mut self, position: [i32; 2]) {
        #[cfg(target_os = "linux")]
        if self.navigation_wheel_scenario() {
            super::desktop_input::click(self, position);
            return;
        }
        #[cfg(target_os = "linux")]
        if self.project_dialog_x11_input() {
            super::x11_input::click(position);
            self.pump(Vec::new());
            return;
        }
        self.pump(pointer_click_events(position, 1));
    }
    pub fn key(&mut self, key: oblikovati_viewport::ViewportPhysicalKey, control: bool) {
        #[cfg(target_os = "linux")]
        if self.project_dialog_x11_input() {
            super::project_dialog::native_input::press_project_key(key, control);
            self.pump(Vec::new());
            return;
        }
        use oblikovati_viewport::ViewportInput;
        let modifiers = oblikovati_viewport::ViewportModifiers {
            control,
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
        // One pump, so the key is down for one frame: Dear ImGui repeats a key held across slow
        // software-rendered frames, and a person's press is shorter than the repeat delay.
        self.pump(vec![
            event(true, modifiers),
            event(false, Default::default()),
        ]);
    }
    /// All visible text, one run per line, for assertions and failure messages.
    pub fn text(&self) -> String {
        let runs: Vec<&str> = self
            .frame
            .runs
            .iter()
            .map(|run| run.text.as_str())
            .collect();
        runs.join("\n")
    }
    pub fn shows(&self, text: &str) -> bool {
        let wanted = squeezed(text);
        diagnostics::shows_run(&self.frame, &wanted)
    }
    pub fn expect_text(&self, text: &str) {
        assert!(
            self.shows(text),
            "expected {text:?} in rendered text {:?}",
            self.text()
        );
    }
    pub fn expect_no_text(&self, text: &str) {
        assert!(
            !self.shows(text),
            "rendered text {:?}: expected no {text:?}",
            self.text()
        );
    }
    /// The topmost run reading exactly `text`, else the topmost run containing it.
    pub fn run(&self, text: &str) -> &diagnostics::RenderedRun {
        let wanted = squeezed(text);
        let exact = self
            .frame
            .runs
            .iter()
            .rev()
            .find(|r| squeezed(&r.text) == wanted);
        exact
            .or_else(|| diagnostics::panel_caption_run(&self.frame, &wanted))
            .or_else(|| {
                self.frame
                    .runs
                    .iter()
                    .rev()
                    .find(|r| squeezed(&r.text).contains(&wanted))
            })
            .unwrap_or_else(|| panic!("expected {text:?} in rendered text {:?}", self.text()))
    }
    /// Press visible text with a real pointer event; a test never reaches for widget state.
    pub fn click_text(&mut self, text: &str) {
        let exact = self
            .frame
            .runs
            .iter()
            .any(|run| squeezed(&run.text) == squeezed(text));
        let center = if exact {
            diagnostics::plane_choice_center(&self.frame, text)
                .unwrap_or_else(|| self.run(text).center())
        } else {
            super::quick_access_icons::standard_icon_center(text)
                .unwrap_or_else(|| self.run(text).center())
        };
        self.click(diagnostics::pointer_pixels(center));
    }
    pub fn drag(&mut self, from: [i32; 2], to: [i32; 2]) {
        self.pump(pointer_drag_press(from));
        for step in 1..=4 {
            let at = std::array::from_fn(|axis| from[axis] + (to[axis] - from[axis]) * step / 4);
            self.pump(vec![pointer_motion(at)]);
        }
        self.pump(pointer_drag_release(to));
    }
    /// A check mark drawn on the same row as `label`; a cleared option has none.
    pub fn checked(&self, label: &str) -> bool {
        self.checked_run(self.run(label))
    }
    /// Inspect a specific rendered row; example: `world.checked_run(world.run("Option"))`.
    pub fn checked_run(&self, run: &diagnostics::RenderedRun) -> bool {
        diagnostics::run_is_checked(&self.frame, run)
    }
    /// The framebuffer colour just left of a label, inside the widget that holds it.
    pub fn pixel_left_of(&mut self, label: &str) -> [u8; 4] {
        let run = self.run(label).clone();
        let capture = super::capture_framebuffer(&mut self.app().window.native);
        let point = diagnostics::pointer_pixels([run.minimum[0] - 3., run.center()[1]]);
        let at = (point[1] as usize * capture.extent[0] as usize + point[0] as usize) * 4;
        std::array::from_fn(|channel| capture.rgba[at + channel])
    }
    pub fn extent(&mut self) -> [u32; 2] {
        self.app()
            .window
            .native
            .window_capture_extent()
            .unwrap()
            .unwrap()
    }
    /// Read viewport pixel bounds; example: `world.viewport_bounds()` after a frame.
    pub fn viewport_bounds(&mut self) -> [u32; 4] {
        let extent = self.extent();
        self.app()
            .window
            .native
            .gui
            .as_ref()
            .expect("expected the GUI")
            .document_area(extent)
            .expect("expected viewport bounds")
    }
    /// Assert the viewport shows in this rectangle: its background gradient has uniform rows.
    pub fn expect_blank_region(&mut self, x: [usize; 2], y: [usize; 2]) {
        let capture = super::capture_framebuffer(&mut self.app().window.native);
        let controls = super::diagnostics::window_occlusions();
        assert!(
            super::viewport_background::gradient_region_visible(&capture, x, y, &controls),
            "region x={x:?} y={y:?}: expected viewport gradient outside GUI windows"
        );
    }
    /// Scope intermediate captures to one scenario, preserving its final root image.
    pub(super) fn set_capture_namespace(&mut self, namespace: String) {
        self.capture_namespace = Some(namespace);
    }
    pub fn save_capture(&mut self, scenario: &str) {
        let folder = capture_folder(&reports_folder(), self.capture_namespace.as_deref());
        self.save_capture_in(scenario, &folder);
    }
    pub(super) fn save_final_capture(&mut self, scenario: &str) {
        let folder =
            capture_folder(&reports_folder(), self.capture_namespace.as_deref()).join("final");
        self.save_capture_in(scenario, &folder);
    }
    fn save_capture_in(&mut self, scenario: &str, folder: &std::path::Path) {
        let Some(app) = self.app.as_mut() else { return };
        let capture = super::capture_framebuffer(&mut app.window.native);
        let name: String = scenario
            .chars()
            .map(|letter| {
                if letter.is_alphanumeric() {
                    letter
                } else {
                    '-'
                }
            })
            .collect();
        std::fs::create_dir_all(folder).expect("scenario capture directory must exist");
        write_png(&folder.join(format!("{name}.png")), &capture)
            .expect("scenario capture must be saved");
    }
    pub fn close(&mut self) {
        if let Some(mut app) = self.app.take() {
            app.owner.finish().unwrap();
            app.window.finish().unwrap();
        }
    }
    pub fn preferences_directory(&self) -> &std::path::Path {
        self._preferences.path()
    }
}
impl Drop for ApplicationWorld {
    fn drop(&mut self) {
        self.close();
    }
}
pub(super) fn squeezed(text: &str) -> String {
    text.chars()
        .filter(|letter| !letter.is_whitespace())
        .collect()
}
pub(super) fn reports_folder() -> std::path::PathBuf {
    let folder = std::env::var_os("OBK_BDD_REPORTS").map_or_else(
        super::report_destination::default_folder,
        std::path::PathBuf::from,
    );
    assert!(
        folder.is_absolute(),
        "BDD reports {folder:?}: expected an absolute output directory"
    );
    folder
}
pub(super) fn write_png(
    path: &std::path::Path,
    capture: &oblikovati_viewport::ViewportCapture,
) -> Result<(), png::EncodingError> {
    let file = std::io::BufWriter::new(super::report_destination::create_file(path)?);
    let mut encoder = png::Encoder::new(file, capture.extent[0], capture.extent[1]);
    encoder.set_color(png::ColorType::Rgba);
    encoder.set_depth(png::BitDepth::Eight);
    encoder.write_header()?.write_image_data(&capture.rgba)
}
pub(super) fn pointer_motion(position: [i32; 2]) -> ViewportEvent {
    ViewportEvent::Input(oblikovati_viewport::ViewportInput::Motion {
        position,
        modifiers: Default::default(),
    })
}
pub(super) fn pointer_button(position: [i32; 2], pressed: bool) -> ViewportEvent {
    ViewportEvent::Input(oblikovati_viewport::ViewportInput::Button {
        button: oblikovati_viewport::ViewportButton::Left,
        pressed,
        position,
        modifiers: Default::default(),
    })
}
fn pointer_drag_press(position: [i32; 2]) -> Vec<ViewportEvent> {
    vec![pointer_motion(position), pointer_button(position, true)]
}
fn pointer_drag_release(position: [i32; 2]) -> Vec<ViewportEvent> {
    vec![pointer_motion(position), pointer_button(position, false)]
}

/// One observed settings checkpoint and its renderer request.
#[derive(Clone, Copy)]
pub(super) struct CurvatureObservation {
    pub sliders: oblikovati_gui::GuiCurvatureSliders,
    pub comb: oblikovati_viewport::ViewportSketchCurvature,
}

/// The fixture's actual geometry, committed history, and visible curve witnesses.
pub(super) struct ObservedSketch {
    pub geometry: oblikovati_model::sketch::Sketch2D,
    pub history: [Vec<String>; 2],
    pub witnesses: Vec<[i32; 2]>,
}

fn capture_folder(root: &std::path::Path, namespace: Option<&str>) -> std::path::PathBuf {
    match namespace {
        Some(name) => root.join("scenarios").join(name),
        None => root.to_owned(),
    }
}

#[cfg(test)]
#[path = "world_capture_tests.rs"]
mod capture_scope_tests;
