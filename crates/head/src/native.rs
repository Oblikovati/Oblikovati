// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow, run_window_with_owner};
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportOptions, ViewportPresentation, ViewportRuntime, ViewportScene,
    ViewportWake,
};
pub struct NativeHeadWindow {
    runtime: ViewportRuntime,
    view: Viewport,
    wake: ViewportWake,
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
        })
    }
}
impl HeadWindow for NativeHeadWindow {
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        self.view.poll_events(limit)
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
/// Open the Linux head on its owner thread; example: `run_native(HeadOptions { frames: Some(3) })?`.
pub fn run_native(options: HeadOptions) -> Result<HeadSummary, String> {
    let mut window = NativeHeadWindow::open()?;
    let owner = native_owner(window.wake.clone())?;
    let executable = std::env::current_exe()
        .map_err(|error| format!("desktop script worker: expected current executable: {error}"))?;
    let engine = oblikovati_script::IsolatedLuaEngine::new(
        &executable,
        vec![oblikovati_script::WORKER_ARGUMENT.into()],
    );
    let mut owner =
        crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
    run_window_with_owner(&mut window, options, &mut owner)
}
#[cfg(test)]
#[path = "../tests/native_wait.rs"]
mod tests;

#[cfg(test)]
#[path = "../tests/native_owner.rs"]
mod owner_tests;
