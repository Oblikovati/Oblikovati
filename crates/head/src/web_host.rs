// SPDX-License-Identifier: GPL-2.0-only
//! Step-driven browser desktop host running the exact `oblikovati-gui` and `oblikovati-head` UI.
use crate::{
    DesktopDispatchOptions, DesktopObservations, DesktopOwner, HeadOptions, HeadWindow, HeadWork,
    NativeHeadWindow,
    frames::{FrameControl, run_owner_frame},
};
use oblikovati_automation::dispatch::DispatchWakeup;
use oblikovati_model::identity::SystemGuidSource;
use oblikovati_persistence::file_access::MemoryDocumentFiles;
use oblikovati_session::files::FileWorkspace;
use oblikovati_viewport::{
    Viewport, ViewportCapture, ViewportEvent, ViewportRuntime, ViewportWake,
};
use std::path::PathBuf;

const MAX_SETTLE_TICKS: usize = 8;

#[derive(Clone)]
pub struct WebOwnerWake(ViewportWake);

impl DispatchWakeup for WebOwnerWake {
    fn wake(&self) -> Result<(), String> {
        self.0.wake()
    }
}

/// Document owner wired to in-memory `.opd` storage for the browser desktop host.
pub type WebDesktopOwner = DesktopOwner<
    DesktopObservations,
    DesktopObservations,
    MemoryDocumentFiles,
    SystemGuidSource,
    DesktopObservations,
    WebOwnerWake,
>;

/// Result of advancing the browser desktop host by one event/animation step.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct WebStepOutcome {
    /// Suggested delay in milliseconds before the next animation/caret tick, if any.
    pub wake_after_ms: Option<u32>,
    /// True once the window has accepted an application close request.
    pub closed: bool,
}

/// Owns the full Oblikovati desktop window, Dear ImGui UI engine, and CAD document owner in WASM.
pub struct WebDesktopHost {
    window: NativeHeadWindow,
    owner: WebDesktopOwner,
    control: FrameControl,
    files: MemoryDocumentFiles,
}

impl WebDesktopHost {
    /// Initialize the full Oblikovati desktop UI around a worker-owned WebGPU viewport.
    /// Example: `let host = WebDesktopHost::new(runtime, view, false)?;`.
    pub fn new(
        runtime: ViewportRuntime,
        view: Viewport,
        validate_webgpu: bool,
    ) -> Result<Self, String> {
        let window = NativeHeadWindow::open_web(runtime, view, validate_webgpu)?;
        let observations = DesktopObservations::default();
        let mem_files = MemoryDocumentFiles::default();
        let files = FileWorkspace::new(observations.clone(), mem_files.clone());
        let version = oblikovati_cli::BuildIdentity::parse(oblikovati_cli::BUILD_MANIFEST)?.version;
        let owner = DesktopOwner::new(
            files,
            SystemGuidSource,
            observations,
            version,
            WebOwnerWake(window.wake_handle()),
            DesktopDispatchOptions::default(),
        )
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())?;
        let mut host = Self {
            window,
            owner,
            control: FrameControl::new(HeadOptions { frames: None }),
            files: mem_files,
        };
        let _ = host.step()?;
        Ok(host)
    }

    /// Push a translated DOM input or resize event into the desktop window queue.
    pub fn push_event(&mut self, event: ViewportEvent) {
        self.window.push_web_event(event);
    }

    /// Resize the browser WebGPU canvas backing store and enqueue a resize event.
    pub fn resize(&mut self, extent: [u32; 2]) -> Result<(), String> {
        self.window.resize_web(extent)
    }

    /// Advance the exact desktop owner/GUI/viewport frame loop without blocking the worker.
    pub fn step(&mut self) -> Result<WebStepOutcome, String> {
        let mut wake_after = None;
        for _ in 0..MAX_SETTLE_TICKS {
            let activity = run_owner_frame(&mut self.window, &mut self.owner, &mut self.control)?;
            wake_after = activity.wake_after;
            if !activity.pending {
                break;
            }
        }
        let wake_after_ms = wake_after.map(|d| d.as_millis().min(u32::MAX as u128) as u32);
        Ok(WebStepOutcome {
            wake_after_ms,
            closed: false,
        })
    }

    /// Stage an uploaded `.opd` file into virtual `/web/` storage and open it in the desktop UI.
    pub fn open_uploaded_document(&mut self, name: &str, bytes: &[u8]) -> Result<(), String> {
        let sanitized: String = name
            .chars()
            .map(|c| {
                if c.is_ascii_alphanumeric() || matches!(c, '.' | '-' | '_') {
                    c
                } else {
                    '_'
                }
            })
            .collect();
        let file_name = if sanitized.is_empty() {
            "Uploaded.opd".to_string()
        } else {
            sanitized
        };
        let virtual_path = PathBuf::from(format!("/web/{file_name}"));
        self.files
            .insert(&virtual_path, bytes.to_vec())
            .map_err(|cause| format!("upload {file_name:?}: {cause}"))?;
        crate::startup::open_startup_documents(&mut self.owner, &[virtual_path])?;
        let _ = self.step()?;
        Ok(())
    }

    /// Capture the current composed WebGPU framebuffer asynchronously for verification.
    pub async fn capture_async(&self) -> Result<ViewportCapture, String> {
        self.window.capture_web_async().await
    }

    /// Return the active WebGPU adapter/device name.
    pub fn device_name(&self) -> String {
        self.window.device_name()
    }

    /// Return accumulated WebGPU validation errors.
    pub fn validation_errors(&self) -> Result<Vec<String>, String> {
        self.window.validation_errors()
    }

    /// Return the current drawable framebuffer extent.
    pub fn drawable_extent(&mut self) -> Result<[u32; 2], String> {
        self.window.drawable_extent()
    }

    /// Return the current GUI command state summary for diagnostics.
    pub fn open_document_count(&mut self) -> Result<usize, String> {
        Ok(self.owner.gui_command_state()?.documents.len())
    }

    /// Cleanly retire the desktop owner and WebGPU viewport.
    pub fn finish(&mut self) -> Result<(), String> {
        let owner_res = self.owner.finish();
        let win_res = self.window.finish();
        owner_res.and(win_res)
    }
}
