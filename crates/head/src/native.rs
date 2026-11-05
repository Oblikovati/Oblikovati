// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow};
#[allow(unsafe_code)]
#[path = "native/addin_launch.rs"]
mod addin_launch;
#[path = "native_preferences.rs"]
mod preferences;
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportOptions, ViewportPresentation, ViewportRuntime, ViewportScene,
    ViewportWake,
};
pub struct NativeHeadWindow {
    capture_files: Box<dyn crate::WindowCaptureFiles>,
    capture_requests: Vec<crate::WindowCaptureJob>,
    runtime: ViewportRuntime,
    view: Viewport,
    wake: ViewportWake,
    scene_status: Option<crate::HeadSceneStatus>,
    sketch_scene: Option<crate::HeadSketchSceneUpdate>,
    navigation: crate::navigation::DesktopNavigation,
    sketch_view: Option<(crate::HeadSketchEdit, oblikovati_viewport::ViewportCamera)>,
    command_notice: Option<crate::DesktopCommandNotice>,
    gui: Option<gui_loop::NativeGui>,
    gui_epoch: std::time::Instant,
    clipboard: std::rc::Rc<clipboard::NativeClipboardChannel>,
}
impl NativeHeadWindow {
    /// Open an owner-thread desktop target; example: `let window = NativeHeadWindow::open()?;`.
    pub fn open() -> Result<Self, String> {
        Self::open_with_capture_files(Box::new(crate::NativeWindowCaptureFiles))
    }
    /// Inject PNG publication while retaining the same native renderer.
    /// Example: `NativeHeadWindow::open_with_capture_files(Box::new(files))?`.
    pub fn open_with_capture_files(
        capture_files: Box<dyn crate::WindowCaptureFiles>,
    ) -> Result<Self, String> {
        let runtime = ViewportRuntime::new(ViewportOptions::default())?;
        let mut view = runtime.open_window("Oblikovati", [1440, 900])?;
        view.submit(&ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        })?;
        Ok(Self {
            capture_files,
            capture_requests: Vec::new(),
            runtime,
            view,
            wake: ViewportWake::new()?,
            scene_status: None,
            sketch_scene: None,
            navigation: Default::default(),
            sketch_view: None,
            command_notice: None,
            gui: None,
            gui_epoch: std::time::Instant::now(),
            clipboard: clipboard::NativeClipboardChannel::open(),
        })
    }
}
impl NativeHeadWindow {
    /// Inspect the current displayed generation and detailed declines; example: `window.scene_status()` for a health panel.
    pub fn scene_status(&self) -> Option<&crate::HeadSceneStatus> {
        self.scene_status.as_ref()
    }
}
impl NativeHeadWindow {
    fn synchronize_browser_highlights(
        &mut self,
        commands: &oblikovati_gui::GuiCommandState,
    ) -> Result<bool, String> {
        let selected = commands
            .browser
            .as_ref()
            .filter(|_| {
                self.sketch_scene
                    .as_ref()
                    .and_then(|scene| scene.source)
                    .map(|(document, _)| document)
                    == commands.document
            })
            .map_or(&[][..], |browser| browser.selected_sketches.as_slice());
        self.view.set_selected_sketches(selected)
    }
    fn apply_navigation_inputs(&mut self, inputs: &[ViewportEvent]) -> Result<bool, String> {
        let mut changed = false;
        for event in inputs {
            let command = self.navigation.command(event)?;
            if previous_view_pressed(event) {
                changed |= self.view.previous_view()?;
                continue;
            }
            if let Some(command) = command {
                changed |= self.view.navigate(command)?;
            }
        }
        Ok(changed)
    }
}
impl HeadWindow for NativeHeadWindow {
    fn window_capture_extent(&mut self) -> Result<Option<[u32; 2]>, String> {
        self.view.drawable_extent().map(Some)
    }
    fn window_capture_ready(&self) -> bool {
        self.capture_requests.is_empty()
    }
    fn request_window_captures(
        &mut self,
        requests: Vec<crate::WindowCaptureJob>,
    ) -> Result<(), String> {
        if !self.capture_requests.is_empty()
            || requests.len() > crate::window_capture::WINDOW_CAPTURE_QUEUE_CAPACITY
        {
            return Err(format!(
                "window captures pending={}, offered={}: expected an empty target and at most {} requests",
                self.capture_requests.len(),
                requests.len(),
                crate::window_capture::WINDOW_CAPTURE_QUEUE_CAPACITY
            ));
        }
        self.capture_requests = requests;
        Ok(())
    }
    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &oblikovati_gui::GuiCommandState,
    ) -> Result<crate::HeadGuiUpdate, String> {
        let changed = match &commands.color_scheme {
            Some(palette) => self.view.set_background(palette.background())?,
            None => false,
        };
        let sketch_changed = match &commands.theme {
            Some(theme) => self.view.set_sketch_palette(theme.sketch_palette())?,
            None => false,
        };
        let selection_changed = self.synchronize_browser_highlights(commands)?;
        let mut update = self.advance_native_gui(event, commands)?;
        update.changed |= changed || sketch_changed || selection_changed;
        Ok(update)
    }

    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.view.set_sketch_display(&update.sketches)?;
        self.sketch_scene = Some(update);
        self.refresh_title()
    }

    fn set_sketch_edit(&mut self, edit: Option<crate::HeadSketchEdit>) -> Result<bool, String> {
        self.synchronize_sketch_camera(edit)
    }

    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        let changed = self.view.set_sketch_preview(preview)?;
        if preview.is_none()
            || !matches!(
                self.command_notice,
                Some(crate::DesktopCommandNotice::PreviewFailed { .. })
            )
        {
            return Ok(changed);
        }
        self.view.set_status_text("")?;
        self.command_notice = None;
        self.refresh_title()?;
        Ok(true)
    }

    fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
        self.apply_navigation_inputs(&[*event])
    }

    fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: oblikovati_model::sketch::SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        NativeHeadWindow::sketch_point(self, pixel, plane)
    }

    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        if let (crate::DesktopCommandNotice::SavePathRequired { document }, Some(gui)) =
            (&notice, &mut self.gui)
        {
            gui.request_save_path(*document)?;
        }
        if self.gui.is_none() {
            self.view.set_status_text(&notice.message())?;
        }
        self.command_notice = Some(notice);
        self.refresh_title()
    }

    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.view.submit(&update.scene)?;
        self.scene_status = Some(update.status);
        self.refresh_title()
    }

    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        self.clipboard.service_pending(limit);
        self.view.poll_events(limit)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        let presentation = self.view.present()?;
        self.refresh_title()?;
        Ok(match presentation {
            ViewportPresentation::Completed(capture) => {
                self.publish_window_captures(&capture)?;
                HeadPresentation::Completed
            }
            ViewportPresentation::SurfaceChanged(_) => {
                if let Some(gui) = &mut self.gui {
                    gui.require_publication();
                }
                HeadPresentation::SurfaceChanged
            }
        })
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        let ready = match self.clipboard.endpoint() {
            Some(clipboard) => self
                .view
                .wait_events_with_clipboard(&self.wake, clipboard, timeout),
            None => self.view.wait_events(&self.wake, timeout),
        }?;
        Ok(ready.worker_wake)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.gui.take();
        self.capture_requests.clear();
        let clipboard = self.clipboard.handoff_on_shutdown();
        self.view.close();
        let mut errors = self.runtime.validation_errors()?;
        if let Err(cause) = clipboard {
            errors.push(cause);
        }
        if errors.is_empty() {
            return Ok(());
        }
        Err(format!(
            "head shutdown {errors:?}: expected completed clipboard retirement and no renderer errors"
        ))
    }
}
mod capture;
mod clipboard;
#[derive(Clone)]
struct NativeOwnerWake(ViewportWake);
impl oblikovati_automation::dispatch::DispatchWakeup for NativeOwnerWake {
    fn wake(&self) -> Result<(), String> {
        self.0.wake()
    }
}
type NativeDesktopOwner = crate::DesktopOwner<
    crate::DesktopObservations,
    crate::DesktopObservations,
    oblikovati_persistence::file_access::NativeDocumentFiles,
    oblikovati_model::identity::SystemGuidSource,
    crate::DesktopObservations,
    NativeOwnerWake,
>;
fn native_owner(wake: ViewportWake) -> Result<NativeDesktopOwner, String> {
    let themes = preferences::load_native_themes()?;
    let observations = crate::DesktopObservations::default();
    let files = oblikovati_session::files::FileWorkspace::new(
        observations.clone(),
        oblikovati_persistence::file_access::NativeDocumentFiles,
    );
    let version = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST)?.version;
    Ok(crate::DesktopOwner::new(
        files,
        oblikovati_model::identity::SystemGuidSource,
        observations,
        version,
        NativeOwnerWake(wake),
        crate::DesktopDispatchOptions::default(),
    )
    .with_theme_store(themes.library, Box::new(themes.preferences))
    .with_theme_file_declines(themes.declines)
    .with_window_captures(crate::DesktopWindowCaptures::new(
        std::env::temp_dir(),
        u64::from(std::process::id()),
    )))
}
#[cfg(test)]
type NativeScriptDesktop = crate::DesktopScripts<NativeDesktopOwner>;
#[cfg(test)]
fn native_script_owner(wake: ViewportWake) -> Result<NativeScriptDesktop, String> {
    let owner = native_owner(wake)?;
    Ok(crate::DesktopScripts::new(
        owner,
        native_script_engine()?,
        oblikovati_script::ScriptLimits::gui(),
    ))
}
fn native_script_engine() -> Result<oblikovati_script::IsolatedLuaEngine, String> {
    let executable = std::env::current_exe()
        .map_err(|cause| format!("desktop script worker: expected current executable: {cause}"))?;
    Ok(oblikovati_script::IsolatedLuaEngine::new(
        &executable,
        vec![oblikovati_script::WORKER_ARGUMENT.into()],
    ))
}
/// Open the Linux head on its owner thread; example: `run_native(HeadOptions { frames: Some(3) })?`.
pub fn run_native(options: HeadOptions) -> Result<HeadSummary, String> {
    run_native_documents(options, &[])
}
/// Open native documents before the first frame; example: `run_native_documents(options, &["part.opd".into()])?`.
pub fn run_native_documents(
    options: HeadOptions,
    documents: &[std::path::PathBuf],
) -> Result<HeadSummary, String> {
    let mut window = NativeHeadWindow::open()?;
    let owner = match native_owner(window.wake.clone()) {
        Ok(owner) => owner,
        Err(cause) => return crate::frames::combine_retirement(Err(cause), window.finish()),
    };
    addin_launch::run_startup(&mut window, owner, options, documents)
}
#[cfg(test)]
#[path = "../tests/native_wait.rs"]
mod tests;

#[cfg(test)]
#[path = "../tests/native_owner.rs"]
mod owner_tests;

#[cfg(test)]
#[path = "../tests/native_scene.rs"]
mod scene_tests;

#[cfg(test)]
#[path = "../tests/navigation/native.rs"]
mod navigation_tests;

#[cfg(test)]
#[path = "../tests/native_startup.rs"]
mod startup_tests;

fn previous_view_pressed(event: &ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(oblikovati_viewport::ViewportInput::Key {
            key: oblikovati_viewport::ViewportKey::Previous,
            pressed: true
        })
    )
}

impl NativeHeadWindow {
    fn refresh_title(&mut self) -> Result<(), String> {
        let mut title = self
            .scene_status
            .as_ref()
            .map_or_else(|| "Oblikovati".into(), crate::HeadSceneStatus::title);
        title.push_str(&self.sketch_display_title()?);
        if let Some(notice) = &self.command_notice {
            title.push_str(&format!(" - {}", notice.message()));
        }
        self.view.set_title(&title)
    }
}

#[cfg(test)]
#[path = "../tests/native_save.rs"]
mod save_tests;

#[cfg(test)]
#[path = "../tests/native_history.rs"]
mod history_tests;

#[cfg(test)]
#[allow(unsafe_code)]
#[path = "../tests/native_services.rs"]
mod addin_services_tests;

mod projection;

mod sketch_edit;

mod sketch_display;

#[cfg(test)]
#[path = "../tests/native_sketch_display.rs"]
mod sketch_display_tests;

mod gui_composition;

#[cfg(test)]
#[path = "../tests/native_gui_composition.rs"]
mod gui_composition_tests;

mod gui_loop;
#[cfg(test)]
#[path = "../tests/native_gui_loop.rs"]
mod gui_loop_tests;
