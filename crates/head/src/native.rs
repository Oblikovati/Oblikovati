// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow, run_window};
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportOptions, ViewportPresentation, ViewportRuntime, ViewportScene,
    ViewportWake,
};
struct NativeHeadWindow {
    runtime: ViewportRuntime,
    view: Viewport,
    wake: ViewportWake,
}
impl NativeHeadWindow {
    fn open() -> Result<Self, String> {
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
    fn wait(&mut self) -> Result<bool, String> {
        self.view
            .wait_events(&self.wake, None)
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
/// Open the Linux head on its owner thread; example: `run_native(HeadOptions { frames: Some(3) })?`.
pub fn run_native(options: HeadOptions) -> Result<HeadSummary, String> {
    run_window(&mut NativeHeadWindow::open()?, options)
}
#[cfg(test)]
#[path = "../tests/native_wait.rs"]
mod tests;
