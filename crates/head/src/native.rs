// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow};
#[allow(unsafe_code)]
#[path = "native/addin_launch.rs"]
mod addin_launch;
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportOptions, ViewportPresentation, ViewportRuntime, ViewportScene,
    ViewportWake,
};
pub struct NativeHeadWindow {
    runtime: ViewportRuntime,
    view: Viewport,
    wake: ViewportWake,
    scene_status: Option<crate::HeadSceneStatus>,
    sketch_scene: Option<crate::HeadSketchSceneUpdate>,
    navigation: crate::navigation::DesktopNavigation,
    sketch_view: Option<(crate::HeadSketchEdit, oblikovati_viewport::ViewportCamera)>,
    command_notice: Option<crate::DesktopCommandNotice>,
    gui: Option<gui_loop::NativeGui>,
    injected_events: std::collections::VecDeque<ViewportEvent>,
    pending_captures: Vec<std::path::PathBuf>,
    gui_epoch: std::time::Instant,
    clipboard: std::rc::Rc<clipboard::NativeClipboardChannel>,
}
impl NativeHeadWindow {
    /// Open an owner-thread desktop target; example: `let window = NativeHeadWindow::open()?;`.
    pub fn open() -> Result<Self, String> {
        Self::open_with_extent([1440, 900])
    }

    /// Open an owner-thread desktop target with a requested drawable extent; example: `NativeHeadWindow::open_with_extent([800, 600])?`.
    pub fn open_with_extent(extent: [u32; 2]) -> Result<Self, String> {
        validate_window_extent(extent)?;
        let runtime = ViewportRuntime::new(ViewportOptions::default())?;
        let mut view = runtime.open_window("Oblikovati", extent)?;
        view.submit(&ViewportScene {
            background: crate::part_scene::GO_DARK_EMPTY_WORKSPACE_BACKGROUND,
            triangles: Vec::new(),
        })?;
        Ok(Self {
            runtime,
            view,
            wake: ViewportWake::new()?,
            scene_status: None,
            sketch_scene: None,
            navigation: Default::default(),
            sketch_view: None,
            command_notice: None,
            gui: None,
            injected_events: Default::default(),
            pending_captures: Vec::new(),
            gui_epoch: std::time::Instant::now(),
            clipboard: clipboard::NativeClipboardChannel::open(),
        })
    }
}

fn validate_window_extent(extent: [u32; 2]) -> Result<(), String> {
    if extent.contains(&0) {
        return Err(format!(
            "native window extent {extent:?}: expected two positive pixel dimensions"
        ));
    }
    Ok(())
}
impl NativeHeadWindow {
    /// Inspect the current displayed generation and detailed declines; example: `window.scene_status()` for a health panel.
    pub fn scene_status(&self) -> Option<&crate::HeadSceneStatus> {
        self.scene_status.as_ref()
    }

    /// Capture the composed native window to a PNG; example: `window.capture_window_png(path)?`.
    pub fn capture_window_png(
        &self,
        path: &std::path::Path,
    ) -> Result<oblikovati_viewport::ViewportCapture, String> {
        let capture = self.view.capture()?;
        crate::capture::write_rgba_png(path, &capture)?;
        Ok(capture)
    }

    fn queue_capture(&mut self, path: &std::path::Path) -> Result<[u32; 2], String> {
        let extent = self.view.drawable_extent()?;
        self.pending_captures.push(path.into());
        self.wake.wake()?;
        Ok(extent)
    }

    fn write_pending_captures(&mut self) -> Result<(), String> {
        for path in std::mem::take(&mut self.pending_captures) {
            let capture = self.view.capture()?;
            crate::capture::write_rgba_png(&path, &capture)?;
        }
        Ok(())
    }
}
impl NativeHeadWindow {
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
    fn capture_window(&self) -> Result<oblikovati_viewport::ViewportCapture, String> {
        self.view.capture()
    }

    fn queue_window_capture(&mut self, path: &std::path::Path) -> Result<[u32; 2], String> {
        self.queue_capture(path)
    }

    fn inject_viewport_input(&mut self, event: ViewportEvent) -> Result<(), String> {
        self.injected_events.push_back(event);
        self.wake.wake()?;
        Ok(())
    }

    fn gui_parameters_open(&self) -> bool {
        self.gui.as_ref().is_some_and(|gui| gui.parameters_open())
    }
    fn set_gui_parameter_snapshot(
        &mut self,
        snapshot: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        let Some(gui) = &mut self.gui else {
            return Ok(false);
        };
        gui.set_parameters(snapshot)
    }
    fn show_gui_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        let Some(gui) = &mut self.gui else {
            return Err(format!(
                "GUI parameter view issue {issue:?}: expected an initialized GUI"
            ));
        };
        gui.report_parameter_issue(issue)
    }
    fn acknowledge_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
        result: Result<(), String>,
    ) -> Result<(), String> {
        let Some(gui) = &mut self.gui else {
            return Ok(());
        };
        gui.acknowledge_parameter(action, result);
        Ok(())
    }

    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &oblikovati_gui::GuiCommandState,
    ) -> Result<crate::HeadGuiUpdate, String> {
        self.advance_native_gui(event, commands)
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
        let queued = limit.min(self.injected_events.len());
        let mut events: Vec<_> = self.injected_events.drain(..queued).collect();
        if events.len() < limit {
            events.extend(self.view.poll_events(limit - events.len())?);
        }
        Ok(events)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        let presentation = self.view.present()?;
        self.write_pending_captures()?;
        self.refresh_title()?;
        Ok(match presentation {
            ViewportPresentation::Completed(_) => HeadPresentation::Completed,
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
        let clipboard = self.clipboard.handoff_on_shutdown();
        self.view.close();
        let mut errors = self.runtime.validation_errors();
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
    let observations = crate::DesktopObservations::default();
    let files = oblikovati_session::files::FileWorkspace::new(
        observations.clone(),
        oblikovati_persistence::file_access::NativeDocumentFiles,
    );
    let version = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST)?.version;
    Ok(crate::DesktopOwner::with_sketch_preferences(
        files,
        oblikovati_model::identity::SystemGuidSource,
        observations,
        version,
        NativeOwnerWake(wake),
        crate::DesktopDispatchOptions::default(),
        native_sketch_preferences()?,
    ))
}

fn native_sketch_preferences()
-> Result<Box<dyn oblikovati_model::application_options::SketchPreferenceAccess>, String> {
    let path = native_options_path()?;
    let parent = path
        .parent()
        .ok_or_else(|| format!("native preferences {path:?}: expected a parent directory"))?;
    std::fs::create_dir_all(parent).map_err(|cause| {
        format!("native preferences {parent:?}: expected a writable directory: {cause}")
    })?;
    let preferences = oblikovati_persistence::application_options::SketchPreferenceFile::load(
        &path,
        oblikovati_persistence::file_access::NativeDocumentFiles,
    )
    .map_err(|cause| {
        format!("native preferences {path:?}: expected a readable options file: {cause}")
    })?;
    Ok(Box::new(preferences))
}

fn native_options_path() -> Result<std::path::PathBuf, String> {
    #[cfg(target_os = "windows")]
    let root = std::env::var_os("APPDATA")
        .map(std::path::PathBuf::from)
        .ok_or_else(|| {
            "native preferences: expected APPDATA to locate the user config directory".to_owned()
        })?;
    #[cfg(not(target_os = "windows"))]
    let root = std::env::var_os("HOME")
        .map(std::path::PathBuf::from)
        .ok_or_else(|| {
            "native preferences: expected HOME to locate the user config directory".to_owned()
        })?;
    #[cfg(target_os = "windows")]
    return Ok(root.join("oblikovati").join("options.yaml"));
    #[cfg(not(target_os = "windows"))]
    Ok(root.join(".oblikovati").join("options.yaml"))
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
#[path = "../tests/native_automation.rs"]
mod automation_tests;

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
