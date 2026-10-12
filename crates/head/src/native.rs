// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow};
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
            if let Some(command) = self.navigation.command(event)? {
                changed |= self.view.navigate(command)?;
            }
        }
        Ok(changed)
    }
}
impl HeadWindow for NativeHeadWindow {
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.view.submit(&update.scene)?;
        self.view.set_title(&update.status.title())?;
        self.scene_status = Some(update.status);
        Ok(())
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
type NativeScriptDesktop = crate::DesktopScripts<NativeDesktopOwner>;
fn native_script_owner(wake: ViewportWake) -> Result<NativeScriptDesktop, String> {
    let executable = std::env::current_exe()
        .map_err(|cause| format!("desktop script worker: expected current executable: {cause}"))?;
    let owner = native_owner(wake)?;
    let engine = oblikovati_script::IsolatedLuaEngine::new(
        &executable,
        vec![oblikovati_script::WORKER_ARGUMENT.into()],
    );
    Ok(crate::DesktopScripts::new(
        owner,
        engine,
        oblikovati_script::ScriptLimits::gui(),
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
    let mut owner = match native_script_owner(window.wake.clone()) {
        Ok(owner) => owner,
        Err(cause) => return crate::frames::combine_retirement(Err(cause), window.finish()),
    };
    crate::run_window_with_documents(&mut window, options, &mut owner, documents)
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
