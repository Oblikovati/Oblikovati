// SPDX-License-Identifier: GPL-2.0-only
//! WebAssembly browser window, event queue, and worker-pool bridge over `martensite-tkservice`.
use crate::{ViewportEvent, ViewportWindowAction, types::valid_extent};
use martensite_tkernel::{ncollection::Vec2, standard::Handle, tcollection::AsciiString};
use martensite_tkservice::{
    aspect::{AspectWindow, AspectWindowMembers as _},
    wasm::WasmWindow,
};
use std::{cell::RefCell, collections::VecDeque};

#[cfg(target_arch = "wasm32")]
pub use martensite_tkernel::osd::WasmWorkerPool as WebWorkerPool;

/// Initialize the Martensite multi-threaded WASM compute worker pool from an explicit glue URL.
/// Example: `let pool = init_web_worker_pool_from_url(4, "./pkg/oblikovati_web.js").await?;`.
#[cfg(target_arch = "wasm32")]
pub async fn init_web_worker_pool_from_url(
    max_workers: usize,
    glue_url: &str,
) -> Result<WebWorkerPool, String> {
    martensite_tkernel::osd::init_wasm_worker_pool_from_url(max_workers, glue_url)
        .await
        .map_err(|error| format!("web worker pool init: {error}"))
}

/// Report the number of active background compute workers in the WASM worker pool.
/// Example: `let workers = web_worker_pool_size();`.
pub fn web_worker_pool_size() -> usize {
    #[cfg(target_arch = "wasm32")]
    {
        martensite_tkernel::osd::wasm_worker_pool_size()
    }
    #[cfg(not(target_arch = "wasm32"))]
    {
        0
    }
}

struct WebWindowState {
    title: String,
    maximized: bool,
    events: VecDeque<ViewportEvent>,
}

/// Browser canvas window wrapper matching Oblikovati's platform window surface.
pub struct WebWindow {
    window: Handle<WasmWindow>,
    state: RefCell<WebWindowState>,
}

#[cfg(target_arch = "wasm32")]
pub(crate) type NativeWindow = WebWindow;

impl WebWindow {
    /// Create a virtual canvas window for headless web or native unit tests.
    /// Example: `let window = WebWindow::new_virtual("Oblikovati", [640, 480], 1.0)?;`.
    pub fn new_virtual(
        title: &str,
        extent: [u32; 2],
        device_pixel_ratio: f64,
    ) -> Result<Self, String> {
        let [width, height] = valid_extent(extent)?;
        let wasm = Handle::new(WasmWindow::new(&AsciiString::from(title), true));
        wasm.set_virtual(true);
        wasm.set_device_pixel_ratio(valid_dpr(device_pixel_ratio));
        wasm.set_size_backing(Vec2::new(width, height));
        wasm.map();
        Ok(Self::from_handle(title, wasm))
    }

    /// Bind a worker-owned `OffscreenCanvas` for browser WebGPU presentation.
    /// Example: `let window = WebWindow::from_offscreen_canvas("#viewport", canvas, 2.0)?;`.
    #[cfg(target_arch = "wasm32")]
    pub fn from_offscreen_canvas(
        canvas_id: &str,
        canvas: web_sys::OffscreenCanvas,
        device_pixel_ratio: f64,
    ) -> Result<Self, String> {
        let extent = [canvas.width(), canvas.height()];
        let _ = valid_extent(extent)?;
        let wasm = Handle::new(WasmWindow::from_offscreen_canvas(
            &AsciiString::from(canvas_id),
            canvas,
            valid_dpr(device_pixel_ratio),
        ));
        Ok(Self::from_handle(canvas_id, wasm))
    }

    /// Bind a main-thread `HtmlCanvasElement` for browser WebGPU presentation.
    /// Example: `let window = WebWindow::from_html_canvas("#viewport", canvas, 2.0)?;`.
    #[cfg(target_arch = "wasm32")]
    pub fn from_html_canvas(
        canvas_id: &str,
        canvas: web_sys::HtmlCanvasElement,
        device_pixel_ratio: f64,
    ) -> Result<Self, String> {
        let extent = [canvas.width(), canvas.height()];
        let _ = valid_extent(extent)?;
        let wasm = Handle::new(WasmWindow::from_html_canvas(
            &AsciiString::from(canvas_id),
            canvas,
            valid_dpr(device_pixel_ratio),
        ));
        Ok(Self::from_handle(canvas_id, wasm))
    }

    fn from_handle(title: &str, window: Handle<WasmWindow>) -> Self {
        Self {
            window,
            state: RefCell::new(WebWindowState {
                title: title.to_owned(),
                maximized: false,
                events: VecDeque::new(),
            }),
        }
    }

    /// Enqueue a translated DOM event for the next `poll` call.
    /// Example: `window.push_event(ViewportEvent::Expose);`.
    pub fn push_event(&self, event: ViewportEvent) {
        self.state.borrow_mut().events.push_back(event);
    }

    /// Report whether queued DOM events are waiting to be polled.
    /// Example: `assert!(window.has_pending_input());`.
    pub fn has_pending_input(&self) -> bool {
        !self.state.borrow().events.is_empty()
    }

    /// Read the active device pixel ratio as a backing scale factor.
    /// Example: `let scale = window.backing_scale();`.
    pub fn backing_scale(&self) -> f32 {
        self.window.device_pixel_ratio() as f32
    }

    /// Report whether the canvas window is currently mapped.
    /// Example: `assert!(window.is_mapped());`.
    pub fn is_mapped(&self) -> bool {
        self.window.is_mapped()
    }

    /// Update the canvas mapped visibility state.
    /// Example: `window.mapped(false)?;`.
    pub fn mapped(&self, mapped: bool) -> Result<(), String> {
        if mapped {
            self.window.map();
        } else {
            self.window.unmap();
        }
        self.push_event(ViewportEvent::Mapped(mapped));
        Ok(())
    }

    /// Drain at most `limit` queued events in FIFO order.
    /// Example: `let batch = window.poll(32)?;`.
    pub fn poll(&self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut state = self.state.borrow_mut();
        let count = limit.min(state.events.len());
        Ok(state.events.drain(..count).collect())
    }

    /// Resize the backing canvas extent in physical pixels.
    /// Example: `window.resize([1280, 720])?;`.
    pub fn resize(&self, extent: [u32; 2]) -> Result<(), String> {
        let [width, height] = valid_extent(extent)?;
        self.window.set_size_backing(Vec2::new(width, height));
        #[cfg(target_arch = "wasm32")]
        if let Ok(Some(target)) = self.window.canvas_target() {
            match target {
                martensite_tkservice::wasm::WasmCanvasTarget::Offscreen(canvas) => {
                    canvas.set_width(extent[0]);
                    canvas.set_height(extent[1]);
                }
                martensite_tkservice::wasm::WasmCanvasTarget::Html(canvas) => {
                    canvas.set_width(extent[0]);
                    canvas.set_height(extent[1]);
                }
            }
        }
        self.push_event(ViewportEvent::Resized(extent));
        Ok(())
    }

    /// Request canvas focus; browser DOM owns actual focus transfer.
    /// Example: `window.request_accessible_focus()?;`.
    pub fn request_accessible_focus(&self) -> Result<(), String> {
        self.push_event(ViewportEvent::Focus(true));
        Ok(())
    }

    /// Record the logical document title for browser chrome display.
    /// Example: `window.set_title("Oblikovati - Part1")?;`.
    pub fn set_title(&self, title: &str) -> Result<(), String> {
        self.state.borrow_mut().title = title.to_owned();
        Ok(())
    }

    /// Return the current logical window title.
    /// Example: `let title = window.title()?;`.
    pub fn title(&self) -> Result<String, String> {
        Ok(self.state.borrow().title.clone())
    }

    /// Report any text-input degradation; browser DOM handles IME natively.
    /// Example: `assert!(window.text_input_issue().is_none());`.
    pub fn text_input_issue(&self) -> Option<String> {
        None
    }

    /// Apply a logical window chrome action to browser window state.
    /// Example: `window.window_action(ViewportWindowAction::ToggleMaximize)?;`.
    pub fn window_action(&self, action: ViewportWindowAction) -> Result<(), String> {
        let mut state = self.state.borrow_mut();
        match action {
            ViewportWindowAction::Minimize => state.maximized = false,
            ViewportWindowAction::ToggleMaximize => state.maximized = !state.maximized,
            ViewportWindowAction::Move => {}
        }
        Ok(())
    }

    /// Report whether the browser view is logically maximized.
    /// Example: `let maximized = window.window_maximized()?;`.
    pub fn window_maximized(&self) -> Result<bool, String> {
        Ok(self.state.borrow().maximized)
    }

    /// Expose the underlying Martensite `AspectWindow` handle.
    /// Example: `let aspect = window.window();`.
    pub fn window(&self) -> Handle<dyn AspectWindow> {
        self.window.clone().cast()
    }
}

fn valid_dpr(dpr: f64) -> f64 {
    if dpr.is_finite() && dpr > 0.0 {
        dpr
    } else {
        1.0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn web_window_manages_extent_events_title_and_actions() {
        let window = WebWindow::new_virtual("Oblikovati Web", [800, 600], 2.0).unwrap();
        assert_eq!(window.backing_scale(), 2.0);
        assert!(window.is_mapped());
        assert_eq!(window.window().size(), (800, 600));
        assert_eq!(window.title().unwrap(), "Oblikovati Web");
        assert!(window.text_input_issue().is_none());

        window.set_title("Part1.opd").unwrap();
        assert_eq!(window.title().unwrap(), "Part1.opd");

        window.resize([1280, 720]).unwrap();
        assert_eq!(window.window().size(), (1280, 720));
        assert!(window.has_pending_input());

        window.mapped(false).unwrap();
        assert!(!window.is_mapped());
        window.mapped(true).unwrap();
        assert!(window.is_mapped());

        window.request_accessible_focus().unwrap();
        window
            .window_action(ViewportWindowAction::ToggleMaximize)
            .unwrap();
        assert!(window.window_maximized().unwrap());
        window
            .window_action(ViewportWindowAction::ToggleMaximize)
            .unwrap();
        assert!(!window.window_maximized().unwrap());
        window
            .window_action(ViewportWindowAction::Minimize)
            .unwrap();
        window.window_action(ViewportWindowAction::Move).unwrap();

        let events = window.poll(16).unwrap();
        assert_eq!(
            events,
            vec![
                ViewportEvent::Resized([1280, 720]),
                ViewportEvent::Mapped(false),
                ViewportEvent::Mapped(true),
                ViewportEvent::Focus(true),
            ]
        );
        assert!(!window.has_pending_input());
        assert_eq!(web_worker_pool_size(), 0);
        assert!(WebWindow::new_virtual("bad", [0, 600], 1.0).is_err());
    }
}
