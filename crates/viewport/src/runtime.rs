// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    ViewportCamera, ViewportCapture, ViewportEvent, ViewportOptions, ViewportPresentation,
    ViewportScene,
    renderer::RenderView,
    target::ViewTarget,
    x11::{NativeWindow, NativeX11},
};
use martensite_vulkan::{Graphic3dRenderDriver, VulkanOptions};
use rustycad_tkernel::standard::Handle;
use std::{cell::Cell, collections::VecDeque, rc::Rc};
#[path = "view_history.rs"]
mod view_history;
struct RuntimeState {
    driver: Handle<Graphic3dRenderDriver>,
    x11: Option<Rc<NativeX11>>,
    next_view: Cell<u64>,
}
/// One owner's renderer runtime. Vulkan resources stay in Martensite; this type cannot be sent to a worker.
///
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_viewport::ViewportRuntime>();
/// ```
pub struct ViewportRuntime {
    state: Rc<RuntimeState>,
}
impl ViewportRuntime {
    /// Initialize before other application X11 connections; example: `ViewportRuntime::new(ViewportOptions::default())?`.
    pub fn new(options: ViewportOptions) -> Result<Self, String> {
        let x11 = NativeX11::open()?;
        let driver = Graphic3dRenderDriver::new(VulkanOptions {
            validation: options.validation,
            device_index: options.physical_device,
        })
        .map_err(|error| error.to_string())?;
        Ok(Self {
            state: Rc::new(RuntimeState {
                driver,
                x11: Some(x11),
                next_view: Cell::new(1),
            }),
        })
    }
    /// Create an application-owned native target; example: `runtime.open_window("Oblikovati", [640,480])?`.
    pub fn open_window(&self, title: &str, extent: [u32; 2]) -> Result<Viewport, String> {
        let x11 = self
            .state
            .x11
            .as_ref()
            .ok_or("viewport runtime: expected native X11 support")?
            .clone();
        self.create(ViewTarget::Native(NativeWindow::create(
            x11, title, extent,
        )?))
    }
    /// Create an explicit capture target on the same kernel renderer; example: `runtime.open_offscreen([64,64])?`.
    pub fn open_offscreen(&self, extent: [u32; 2]) -> Result<Viewport, String> {
        self.create(ViewTarget::offscreen(extent)?)
    }
    fn create(&self, target: ViewTarget) -> Result<Viewport, String> {
        let id = self.state.next_view.get();
        let next = id
            .checked_add(1)
            .ok_or("viewport id exhausted: expected an unused u64 view id")?;
        let extent = target.extent();
        let render = RenderView::new(&self.state.driver, &target.window(), extent)?;
        self.state.next_view.set(next);
        Ok(Viewport {
            render: Some(render),
            target: Some(target),
            _runtime: self.state.clone(),
            id,
            scene: 0,
            camera: 0,
            surface: 0,
            extent,
            completed: None,
            overlay: 0,
            view_history: VecDeque::new(),
        })
    }
    /// Name the selected device; example: `runtime.device_name()` for diagnostics.
    pub fn device_name(&self) -> String {
        self.state.driver.device_name()
    }
    /// Read kernel Vulkan validation errors; example: `runtime.validation_errors()` after viewport acceptance.
    pub fn validation_errors(&self) -> Vec<String> {
        self.state.driver.validation_errors()
    }
}
/// Kernel views retire before their native target; the shared runtime outlives both.
///
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_viewport::Viewport>();
/// ```
pub struct Viewport {
    render: Option<RenderView>,
    target: Option<ViewTarget>,
    _runtime: Rc<RuntimeState>,
    id: u64,
    scene: u64,
    camera: u64,
    surface: u64,
    extent: [u32; 2],
    completed: Option<[u64; 4]>,
    overlay: u64,
    view_history: VecDeque<crate::renderer::SavedViewCamera>,
}
impl Viewport {
    /// Read this runtime's view identity; example: `viewport.id()` tags captures.
    pub fn id(&self) -> u64 {
        self.id
    }
    fn live(&self) -> Result<&RenderView, String> {
        self.render
            .as_ref()
            .ok_or_else(|| format!("viewport {}: expected a live view", self.id))
    }
    fn generations(&self) -> [u64; 4] {
        [self.scene, self.camera, self.surface, self.overlay]
    }
    /// Replace validated presentation inputs; example: `viewport.submit(&scene)?`.
    pub fn submit(&mut self, scene: &ViewportScene) -> Result<(), String> {
        self.live()?;
        let next = next_generation(self.scene)?;
        self.render.as_mut().expect("live view").submit(scene)?;
        self.scene = next;
        self.completed = None;
        Ok(())
    }
    /// Replace a validated orthographic camera atomically; example: `viewport.set_camera(camera)?`.
    pub fn set_camera(&mut self, camera: ViewportCamera) -> Result<(), String> {
        self.live()?;
        self.refresh_extent()?;
        let next = next_generation(self.camera)?;
        let extent = self.extent;
        self.render
            .as_mut()
            .expect("live view")
            .camera(camera, extent)?;
        self.camera = next;
        self.completed = None;
        Ok(())
    }
    /// Read the current kernel camera without exposing a mutable handle; example: `viewport.camera_state()?`.
    pub fn camera_state(&self) -> Result<ViewportCamera, String> {
        self.live().map(RenderView::camera_state)
    }
    /// Apply camera input and invalidate only a changed frame; example: `viewport.navigate(ViewportNavigation::Pan([10, 0]))?`.
    pub fn navigate(&mut self, command: crate::ViewportNavigation) -> Result<bool, String> {
        self.live()?;
        self.refresh_extent()?;
        command.validate(self.extent)?;
        if command.is_stationary() {
            return Ok(false);
        }
        let next = next_generation(self.camera)?;
        if !self.apply_camera_navigation(command)? {
            return Ok(false);
        }
        self.camera = next;
        self.completed = None;
        Ok(true)
    }
    /// Resize and report actual drawable size; example: `viewport.resize([800,600])?`.
    pub fn resize(&mut self, extent: [u32; 2]) -> Result<[u32; 2], String> {
        self.live()?;
        next_generation(self.camera)?;
        next_generation(self.surface)?;
        let actual = self.target.as_ref().expect("live target").resize(extent)?;
        self.completed = None;
        self.resized(actual)?;
        Ok(actual)
    }
    fn resized(&mut self, extent: [u32; 2]) -> Result<(), String> {
        let camera = next_generation(self.camera)?;
        let surface = next_generation(self.surface)?;
        self.live()?.resized(extent);
        self.camera = camera;
        self.surface = surface;
        self.extent = extent;
        self.completed = None;
        Ok(())
    }
    /// Poll bounded native lifecycle events; offscreen targets have no native events.
    /// Example: `viewport.poll_events(32)?` lets the owner react to close requests.
    pub fn poll_events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        self.live()?;
        let events = match self.target.as_ref().expect("live target") {
            ViewTarget::Native(native) => native.poll(limit),
            ViewTarget::Offscreen(_) => Vec::new(),
        };
        self.observe_visibility(&events)?;
        self.refresh_extent()?;
        Ok(events)
    }
    fn observe_visibility(&mut self, events: &[ViewportEvent]) -> Result<(), String> {
        for event in events {
            if matches!(event, ViewportEvent::Mapped(_)) {
                self.completed = None;
                self.surface = next_generation(self.surface)?;
            }
        }
        Ok(())
    }
    fn refresh_extent(&mut self) -> Result<(), String> {
        let actual = self.target.as_ref().expect("live target").extent();
        if actual == self.extent {
            return Ok(());
        }
        self.completed = None;
        self.resized(actual)
    }
    /// Map or hide a native target without a second renderer; example: `viewport.set_mapped(false)?`.
    pub fn set_mapped(&mut self, mapped: bool) -> Result<(), String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!("viewport {}: expected a native target", self.id));
        };
        let surface = next_generation(self.surface)?;
        self.completed = None;
        native.mapped(mapped)?;
        self.surface = surface;
        Ok(())
    }
    /// Present and return only a fresh completed frame; example: `let capture=viewport.redraw()?;`.
    pub fn redraw(&mut self) -> Result<ViewportCapture, String> {
        match self.present()? {
            ViewportPresentation::Completed(capture) => Ok(capture),
            ViewportPresentation::SurfaceChanged(extent) => Err(format!(
                "viewport {} changed extent to {extent:?} during redraw: expected a new frame",
                self.id
            )),
        }
    }
    /// Present once and distinguish an observed resize from a renderer fault.
    /// Example: schedule a new frame after `ViewportPresentation::SurfaceChanged(extent)`.
    pub fn present(&mut self) -> Result<ViewportPresentation, String> {
        self.draw()?;
        let frame = self
            .live()?
            .renderer
            .capture()
            .map_err(|error| format!("viewport {} capture: {error}", self.id))?;
        if let Some(extent) = self.observe_draw_extent()? {
            return Ok(ViewportPresentation::SurfaceChanged(extent));
        }
        self.require_frame_extent([frame.width, frame.height])?;
        self.completed = Some(self.generations());
        Ok(ViewportPresentation::Completed(self.capture_from(frame)))
    }
    fn draw(&mut self) -> Result<(), String> {
        self.completed = None;
        self.live()?;
        self.refresh_extent()?;
        let preview = self.live()?.preview_source();
        self.set_sketch_preview(preview.as_ref())?;
        self.refresh_sketch_display()?;
        self.target
            .as_ref()
            .expect("live target")
            .require_presentation()?;
        self.live()?
            .renderer
            .try_redraw()
            .map_err(|error| format!("viewport {} redraw: {error}", self.id))
    }
    fn observe_draw_extent(&mut self) -> Result<Option<[u32; 2]>, String> {
        if self.target.as_ref().expect("live target").extent() == self.extent {
            return Ok(None);
        }
        self.refresh_extent()?;
        Ok(Some(self.extent))
    }
    /// Read only a current successful frame; example: `viewport.capture()?` refuses stale source/camera/size.
    pub fn capture(&self) -> Result<ViewportCapture, String> {
        self.require_current_frame()?;
        let frame = self
            .live()?
            .renderer
            .capture()
            .map_err(|error| format!("viewport {} capture: {error}", self.id))?;
        self.require_frame_extent([frame.width, frame.height])?;
        Ok(self.capture_from(frame))
    }
    fn capture_from(&self, frame: martensite_vulkan::FrameCapture) -> ViewportCapture {
        ViewportCapture {
            view_id: self.id,
            scene_generation: self.scene,
            camera_generation: self.camera,
            surface_generation: self.surface,
            overlay_generation: self.overlay,
            extent: self.extent,
            rgba: frame.rgba,
        }
    }
    fn require_frame_extent(&self, extent: [u32; 2]) -> Result<(), String> {
        if extent == self.extent {
            return Ok(());
        }
        Err(format!(
            "viewport frame {extent:?}: expected extent {:?}",
            self.extent
        ))
    }
    fn require_current_frame(&self) -> Result<(), String> {
        self.live()?;
        let target = self.target.as_ref().expect("live target");
        target.require_presentation()?;
        if self.completed != Some(self.generations()) || target.extent() != self.extent {
            return Err(format!(
                "viewport {} frame {:?}: expected completed current generations {:?} and extent {:?}",
                self.id,
                self.completed,
                self.generations(),
                self.extent
            ));
        }
        Ok(())
    }
    /// Wait without rendering, preserving queued native events; example: `view.wait_events(&wake, None)?;`.
    pub fn wait_events(
        &self,
        wake: &crate::ViewportWake,
        timeout: Option<std::time::Duration>,
    ) -> Result<crate::ViewportReady, String> {
        self.live()?;
        match self.target.as_ref().expect("live target") {
            ViewTarget::Native(native) => native.wait_events(wake, timeout),
            ViewTarget::Offscreen(_) => Err(format!(
                "viewport {}: expected a native event source",
                self.id
            )),
        }
    }
    /// Capture the actual X11 drawable independently of GPU readback; example: `viewport.native_pixels()?`.
    pub fn native_pixels(&self) -> Result<Vec<u8>, String> {
        self.live()?;
        match self.target.as_ref().expect("live target") {
            ViewTarget::Native(native) => native.pixels(),
            ViewTarget::Offscreen(_) => {
                Err(format!("viewport {}: expected a native drawable", self.id))
            }
        }
    }
    /// Retire kernel views before releasing the native window; example: `viewport.close()` is idempotent.
    pub fn close(&mut self) {
        self.completed = None;
        self.render.take();
        self.target.take();
    }
}
impl Drop for Viewport {
    fn drop(&mut self) {
        self.close();
    }
}
fn next_generation(current: u64) -> Result<u64, String> {
    current
        .checked_add(1)
        .ok_or_else(|| format!("viewport generation {current}: expected an incrementable u64"))
}

#[cfg(test)]
#[path = "../tests/recorded.rs"]
mod tests;

impl Viewport {
    /// Update native presentation status without changing geometry; example: `view.set_title("Oblikovati - feature failed")?`.
    pub fn set_title(&mut self, title: &str) -> Result<(), String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native title target",
                self.id
            ));
        };
        native.set_title(title)
    }
    /// Read the actual X11 name; example: `view.native_title()?` checks displayed status metadata.
    pub fn native_title(&self) -> Result<String, String> {
        self.live()?;
        let Some(ViewTarget::Native(native)) = &self.target else {
            return Err(format!(
                "viewport {}: expected a native title target",
                self.id
            ));
        };
        native.title()
    }
}

impl Viewport {
    /// Show screen-space command feedback through the existing kernel view.
    /// Example: `viewport.set_status_text("Saved part.opd")?`; an empty string clears it.
    pub fn set_status_text(&mut self, text: &str) -> Result<bool, String> {
        self.update_overlay(|render| render.status_text(text))
    }
}

mod projection;

mod sketch_preview;

mod sketch_display;

impl Viewport {
    fn update_overlay(
        &mut self,
        replace: impl FnOnce(&mut RenderView) -> Result<bool, String>,
    ) -> Result<bool, String> {
        self.live()?;
        self.refresh_extent()?;
        let next = next_generation(self.overlay)?;
        if !replace(self.render.as_mut().expect("live view"))? {
            return Ok(false);
        }
        self.overlay = next;
        self.completed = None;
        Ok(true)
    }
}

mod gui_composition;
