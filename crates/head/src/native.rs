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
    navigation: crate::navigation::DesktopNavigation,
    command_notice: Option<crate::DesktopCommandNotice>,
}
impl NativeHeadWindow {
    /// Open an owner-thread desktop target; example: `let window = NativeHeadWindow::open()?;`.
    pub fn open() -> Result<Self, String> {
        let runtime = ViewportRuntime::new(ViewportOptions::default())?;
        let mut view = runtime.open_window("Oblikovati", [1440, 900])?;
        view.submit(&ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        })?;
        Ok(Self {
            runtime,
            view,
            wake: ViewportWake::new()?,
            scene_status: None,
            navigation: Default::default(),
            command_notice: None,
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
    fn navigation_events(
        &mut self,
        mut inputs: Vec<ViewportEvent>,
    ) -> Result<Vec<ViewportEvent>, String> {
        if self.apply_navigation_inputs(&inputs)? {
            inputs.push(ViewportEvent::Expose);
        }
        Ok(inputs)
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
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.view.set_status_text(&notice.message())?;
        self.command_notice = Some(notice);
        self.refresh_title()
    }

    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.view.submit(&update.scene)?;
        self.scene_status = Some(update.status);
        self.refresh_title()
    }

    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let inputs = self.view.poll_events(limit)?;
        self.navigation_events(inputs)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(match self.view.present()? {
            ViewportPresentation::Completed(_) => HeadPresentation::Completed,
            ViewportPresentation::SurfaceChanged(_) => HeadPresentation::SurfaceChanged,
        })
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.view
            .wait_events(&self.wake, timeout)
            .map(|ready| ready.worker_wake)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.view.close();
        let errors = self.runtime.validation_errors();
        if errors.is_empty() {
            return Ok(());
        }
        Err(format!(
            "head Vulkan validation {errors:?}: expected no renderer errors"
        ))
    }
}
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
    Ok(crate::DesktopOwner::new(
        files,
        oblikovati_model::identity::SystemGuidSource,
        observations,
        version,
        NativeOwnerWake(wake),
        crate::DesktopDispatchOptions::default(),
    ))
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
