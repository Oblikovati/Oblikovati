// SPDX-License-Identifier: GPL-2.0-only
//! Platform-agnostic Oblikovati Web session bridging `FileWorkspace`, `PartReplayService`, and WebGPU `Viewport`.

mod editing;
mod observers;
mod snapshot;

use crate::{
    WebCommand, WebNavigationController, WebStateSnapshot, WebTheme, empty_part_package,
    navigation::oriented_camera, sample_prism_package,
};
use oblikovati_automation::PartReplayService;
use oblikovati_model::identity::SystemGuidSource;
use oblikovati_persistence::file_access::MemoryDocumentFiles;
use oblikovati_session::{files::FileWorkspace, workspace::OpenActivation};
use oblikovati_viewport::{
    Viewport, ViewportCapture, ViewportEvent, ViewportInput, ViewportKey, ViewportNavigation,
    ViewportOptions, ViewportPresentation, ViewportRuntime, ViewportScene, WebKeyEvent,
    WebPointerEvent, WebResizeEvent, WebWheelEvent, translate_web_key, translate_web_pointer,
    translate_web_resize, translate_web_wheel,
};
use observers::{SilentTransactionObserver, SilentWorkspaceObserver};
use std::path::{Path, PathBuf};

const WEB_APP_VERSION: &str = "0.1.0-web";
pub(super) const DEFAULT_BODY_COLOR: [f32; 4] = [0.68, 0.75, 0.84, 1.0];

/// Complete browser CAD session owning documents, BRep replay, and a WebGPU `Viewport`.
pub struct WebSession {
    files_handle: MemoryDocumentFiles,
    workspace:
        FileWorkspace<SilentTransactionObserver, SilentWorkspaceObserver, MemoryDocumentFiles>,
    replay: PartReplayService,
    runtime: ViewportRuntime,
    viewport: Viewport,
    navigation: WebNavigationController,
    theme: WebTheme,
    normal_debug: bool,
    current_scene: Option<ViewportScene>,
    visible_bodies: usize,
    next_part_index: u64,
    status_message: String,
}

impl WebSession {
    /// Initialize a WebGPU session backed by an offscreen viewport (useful on desktop and tests).
    /// Example: `let session = WebSession::new_offscreen(ViewportOptions::default(), [320, 240])?;`.
    #[cfg(not(target_arch = "wasm32"))]
    pub fn new_offscreen(options: ViewportOptions, extent: [u32; 2]) -> Result<Self, String> {
        let runtime = ViewportRuntime::new_webgpu(options)?;
        let viewport = runtime.open_offscreen(extent)?;
        Self::from_runtime(runtime, viewport)
    }

    /// Initialize a WebGPU session bound to a worker-owned browser `OffscreenCanvas`.
    #[cfg(target_arch = "wasm32")]
    pub async fn new_offscreen_canvas(
        options: ViewportOptions,
        canvas_id: &str,
        canvas: web_sys::OffscreenCanvas,
        device_pixel_ratio: f64,
    ) -> Result<Self, String> {
        let runtime = ViewportRuntime::new_webgpu_async(options).await?;
        let viewport = runtime.open_offscreen_canvas(canvas_id, canvas, device_pixel_ratio)?;
        Self::from_runtime(runtime, viewport)
    }

    fn from_runtime(runtime: ViewportRuntime, mut viewport: Viewport) -> Result<Self, String> {
        let theme = WebTheme::Dark;
        viewport.set_background(theme.viewport_background())?;
        let files_handle = MemoryDocumentFiles::default();
        let workspace = FileWorkspace::new(SilentWorkspaceObserver, files_handle.clone());
        Ok(Self {
            files_handle,
            workspace,
            replay: PartReplayService::default(),
            runtime,
            viewport,
            navigation: WebNavigationController::default(),
            theme,
            normal_debug: false,
            current_scene: None,
            visible_bodies: 0,
            next_part_index: 1,
            status_message: "Ready".into(),
        })
    }

    /// Load the built-in parametric rectangular prism sample Part and frame it in the viewport.
    /// Example: `let doc_id = session.load_sample_part()?;`.
    pub fn load_sample_part(&mut self) -> Result<u64, String> {
        let name = format!("SamplePart{}", self.next_part_index);
        self.next_part_index += 1;
        let package = sample_prism_package(&name)?;
        let mut guids = SystemGuidSource;
        let reg = self
            .workspace
            .create_unbound(
                package,
                SilentTransactionObserver,
                WEB_APP_VERSION,
                &mut guids,
            )
            .map_err(|err| err.to_string())?;
        self.workspace
            .activate(reg.session_id)
            .map_err(|err| err.to_string())?;
        self.status_message = format!("Loaded parametric sample {name}");
        self.refresh_active_scene(true)?;
        Ok(reg.session_id)
    }

    /// Create a new empty Part document and activate it.
    /// Example: `let doc_id = session.new_part(None)?;`.
    pub fn new_part(&mut self, name: Option<String>) -> Result<u64, String> {
        let label = name.unwrap_or_else(|| {
            let n = self.next_part_index;
            self.next_part_index += 1;
            format!("Part{n}")
        });
        let package = empty_part_package(&label);
        let mut guids = SystemGuidSource;
        let reg = self
            .workspace
            .create_unbound(
                package,
                SilentTransactionObserver,
                WEB_APP_VERSION,
                &mut guids,
            )
            .map_err(|err| err.to_string())?;
        self.workspace
            .activate(reg.session_id)
            .map_err(|err| err.to_string())?;
        self.status_message = format!("Created {label}");
        self.refresh_active_scene(true)?;
        Ok(reg.session_id)
    }

    /// Open a `.opd` document from raw uploaded bytes into the in-memory browser store.
    /// Example: `let id = session.open_document_bytes("Bracket.opd", &bytes)?;`.
    pub fn open_document_bytes(&mut self, name: &str, bytes: &[u8]) -> Result<u64, String> {
        let sanitized = sanitize_virtual_name(name);
        let path = virtual_web_path(&sanitized);
        self.files_handle
            .insert(&path, bytes.to_vec())
            .map_err(|err| err.to_string())?;
        let mut guids = SystemGuidSource;
        let outcome = self
            .workspace
            .open(
                &path,
                SilentTransactionObserver,
                WEB_APP_VERSION,
                &mut guids,
                OpenActivation::Activate,
            )
            .map_err(|err| err.to_string())?;
        let id = outcome.session_id();
        self.status_message = format!("Opened {sanitized}");
        self.refresh_active_scene(true)?;
        Ok(id)
    }

    /// Save the active document into the in-memory store and return its filename and `.opd` bytes.
    /// Example: `let (filename, bytes) = session.export_active_document_bytes()?;`.
    pub fn export_active_document_bytes(&mut self) -> Result<(String, Vec<u8>), String> {
        let id = self
            .workspace
            .workspace()
            .active_document()
            .ok_or_else(|| "export document: expected an active document".to_owned())?;
        let display = self
            .workspace
            .workspace()
            .display_name(id)
            .map_err(|err| err.to_string())?
            .to_owned();
        let filename = ensure_opd_extension(&display);
        let path = self
            .workspace
            .path(id)
            .map(Path::to_path_buf)
            .unwrap_or_else(|_| virtual_web_path(&format!("export-{id}-{filename}")));
        let mut guids = SystemGuidSource;
        self.workspace
            .save_as(id, &path, WEB_APP_VERSION, &mut guids)
            .map_err(|err| err.to_string())?;
        let bytes = self
            .files_handle
            .get(&path)
            .ok_or_else(|| format!("export {path:?}: expected persisted package bytes"))?;
        self.status_message = format!("Exported {filename} ({} bytes)", bytes.len());
        Ok((filename, bytes))
    }

    /// Parse and execute a JSON-encoded [`WebCommand`], returning the updated JSON snapshot.
    /// Example: `let state = session.execute_command_json(r#"{"type":"fit"}"#)?;`.
    pub fn execute_command_json(&mut self, json: &str) -> Result<String, String> {
        let command: WebCommand = serde_json::from_str(json)
            .map_err(|err| format!("web command JSON: expected a valid WebCommand: {err}"))?;
        let snapshot = self.execute_command(command)?;
        serde_json::to_string(&snapshot).map_err(|err| err.to_string())
    }

    /// Execute one typed [`WebCommand`] and return the updated workbench snapshot.
    /// Example: `let state = session.execute_command(WebCommand::Fit)?;`.
    pub fn execute_command(&mut self, command: WebCommand) -> Result<WebStateSnapshot, String> {
        match command {
            WebCommand::NewPart { name } => {
                self.new_part(name)?;
            }
            WebCommand::LoadSamplePart => {
                self.load_sample_part()?;
            }
            WebCommand::ActivateDocument { id } => {
                self.workspace.activate(id).map_err(|err| err.to_string())?;
                self.status_message = format!("Activated document #{id}");
                self.refresh_active_scene(true)?;
            }
            WebCommand::CloseDocument { id } => {
                self.workspace
                    .close_discarding_changes(id)
                    .map_err(|err| err.to_string())?;
                self.status_message = format!("Closed document #{id}");
                self.refresh_active_scene(true)?;
            }
            WebCommand::SetParameter { name, expression } => {
                self.edit_parameter(&name, &expression)?;
            }
            WebCommand::Undo => self.apply_undo()?,
            WebCommand::Redo => self.apply_redo()?,
            WebCommand::Orient { view } => {
                let next = oriented_camera(self.viewport.camera_state()?, view);
                self.viewport.set_camera(next)?;
                let _ = self
                    .viewport
                    .navigate(ViewportNavigation::Fit { margin: 0.12 });
                self.status_message = format!("Oriented view: {view:?}");
            }
            WebCommand::Fit => {
                let _ = self
                    .viewport
                    .navigate(ViewportNavigation::Fit { margin: 0.12 });
                self.status_message = "Fitted view".into();
            }
            WebCommand::Home => {
                let _ = self
                    .viewport
                    .navigate(ViewportNavigation::Home { margin: 0.12 });
                self.status_message = "Reset to Home view".into();
            }
            WebCommand::SetProjection { perspective } => {
                let mut camera = self.viewport.camera_state()?;
                camera.perspective = perspective;
                self.viewport.set_camera(camera)?;
                self.status_message = if perspective {
                    "Projection: Perspective".into()
                } else {
                    "Projection: Orthographic".into()
                };
            }
            WebCommand::SetTheme { theme } => {
                self.theme = theme;
                self.viewport.set_background(theme.viewport_background())?;
                self.status_message = format!("Theme: {theme:?}");
            }
            WebCommand::SetNormalDebug { enabled } => {
                self.normal_debug = enabled;
                self.viewport
                    .set_normal_debug(self.current_scene.as_ref(), enabled)?;
                self.status_message = if enabled {
                    "Normal debug enabled".into()
                } else {
                    "Normal debug disabled".into()
                };
            }
            WebCommand::Orbit { yaw, pitch } => {
                let _ = self
                    .viewport
                    .navigate(ViewportNavigation::Orbit { yaw, pitch })?;
            }
            WebCommand::Pan { dx, dy } => {
                let _ = self.viewport.navigate(ViewportNavigation::Pan([dx, dy]))?;
            }
            WebCommand::Zoom { factor, anchor } => {
                let _ = self
                    .viewport
                    .navigate(ViewportNavigation::Zoom { factor, anchor })?;
            }
        }
        Ok(self.snapshot())
    }

    /// Handle a translated browser pointer event; returns true if the camera moved.
    /// Example: `let dirty = session.handle_pointer(event)?;`.
    pub fn handle_pointer(&mut self, event: WebPointerEvent) -> Result<bool, String> {
        let dpr = f64::from(self.viewport.backing_scale());
        let Some(translated) = translate_web_pointer(event, dpr) else {
            return Ok(false);
        };
        self.apply_viewport_event(translated)
    }

    /// Handle a translated browser wheel event; returns true if the camera zoomed.
    /// Example: `let dirty = session.handle_wheel(event)?;`.
    pub fn handle_wheel(&mut self, event: WebWheelEvent) -> Result<bool, String> {
        let dpr = f64::from(self.viewport.backing_scale());
        let Some(translated) = translate_web_wheel(event, dpr) else {
            return Ok(false);
        };
        self.apply_viewport_event(translated)
    }

    /// Handle a browser keyboard event (Undo, Redo, Home); returns true if state changed.
    /// Example: `let dirty = session.handle_key(&event)?;`.
    pub fn handle_key(&mut self, event: &WebKeyEvent) -> Result<bool, String> {
        let mut changed = false;
        for translated in translate_web_key(event) {
            if let ViewportEvent::Input(ViewportInput::PhysicalKey {
                command: Some(command),
                pressed: true,
                ..
            }) = translated
            {
                match command {
                    ViewportKey::Undo => {
                        let _ = self.apply_undo();
                        changed = true;
                        continue;
                    }
                    ViewportKey::Redo => {
                        let _ = self.apply_redo();
                        changed = true;
                        continue;
                    }
                    _ => {}
                }
            }
            changed |= self.apply_viewport_event(translated)?;
        }
        Ok(changed)
    }

    /// Resize the WebGPU canvas from CSS dimensions and device pixel ratio.
    /// Example: `let actual = session.resize(event)?;`.
    pub fn resize(&mut self, event: WebResizeEvent) -> Result<[u32; 2], String> {
        let target = translate_web_resize(event)?;
        self.viewport.resize(target)
    }

    /// Present the current scene on WebGPU without blocking for pixel readback.
    /// Example: `session.render_frame()?;`.
    pub fn render_frame(&mut self) -> Result<bool, String> {
        self.viewport.present_frame()
    }

    /// Present and asynchronously read back RGBA pixels for tests and thumbnails.
    /// Example: `let outcome = session.present_async().await?;`.
    pub async fn present_async(&mut self) -> Result<ViewportPresentation, String> {
        self.viewport.present_async().await
    }

    /// Asynchronously read back the last completed WebGPU frame.
    /// Example: `let capture = session.capture_async().await?;`.
    pub async fn capture_async(&self) -> Result<ViewportCapture, String> {
        self.viewport.capture_async().await
    }

    /// Read accumulated WebGPU validation messages from the active runtime.
    /// Example: `assert!(session.validation_errors()?.is_empty());`.
    pub fn validation_errors(&self) -> Result<Vec<String>, String> {
        self.runtime.validation_errors()
    }

    /// Retire the active WebGPU viewport cleanly.
    /// Example: `session.close();`.
    pub fn close(&mut self) {
        self.viewport.close();
    }

    fn apply_viewport_event(&mut self, event: ViewportEvent) -> Result<bool, String> {
        let Some(step) = self.navigation.consume(event) else {
            return Ok(false);
        };
        self.viewport.navigate(step)
    }
}

fn sanitize_virtual_name(name: &str) -> String {
    let base = Path::new(name)
        .file_name()
        .and_then(|s| s.to_str())
        .filter(|s| !s.is_empty())
        .unwrap_or("uploaded.opd");
    ensure_opd_extension(base)
}

fn virtual_web_path(file_name: &str) -> PathBuf {
    #[cfg(windows)]
    const ROOT: &str = r"C:\web";
    #[cfg(not(windows))]
    const ROOT: &str = "/web";
    Path::new(ROOT).join(file_name)
}

fn ensure_opd_extension(name: &str) -> String {
    if name.ends_with(".opd") || name.ends_with(".obk") {
        name.to_owned()
    } else {
        format!("{name}.opd")
    }
}

#[cfg(test)]
mod tests {
    use super::virtual_web_path;

    #[test]
    fn virtual_web_path_is_platform_absolute() {
        let path = virtual_web_path("SamplePart1.opd");
        #[cfg(not(target_arch = "wasm32"))]
        assert!(path.is_absolute(), "expected absolute path, got {path:?}");
        assert_eq!(
            path.file_name().and_then(|name| name.to_str()),
            Some("SamplePart1.opd")
        );
    }
}
