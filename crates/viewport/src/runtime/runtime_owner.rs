// SPDX-License-Identifier: GPL-2.0-only
//! Creation and ownership of renderer-backed views.
use super::*;

impl ViewportRuntime {
    /// Set Vulkan swapchain synchronization for native Linux windows.
    /// Example: `runtime.set_vertical_sync(false);` before opening a window.
    #[cfg(target_os = "linux")]
    pub fn set_vertical_sync(&self, enabled: bool) {
        use martensite_tkservice::graphic3d::Graphic3dGraphicDriver;
        self.state.driver.set_vertical_sync(enabled);
    }

    /// Initialize the owner-thread Vulkan runtime; example: `ViewportRuntime::new(ViewportOptions::default())?`.
    #[cfg(any(target_os = "linux", target_os = "windows"))]
    pub fn new(options: ViewportOptions) -> Result<Self, String> {
        #[cfg(target_os = "linux")]
        let linux = NativeLinux::open()?;
        let driver = gui_outcomes::native_gui_driver(options)?;
        Ok(Self {
            state: Rc::new(RuntimeState {
                driver,
                #[cfg(target_os = "linux")]
                linux: Some(linux),
                #[cfg(target_os = "windows")]
                native_windows: true,
                next_view: Cell::new(1),
            }),
        })
    }
    /// Asynchronously initialize a WebGPU-backed viewport runtime on WebAssembly or desktop.
    /// Example: `let runtime = ViewportRuntime::new_webgpu_async(ViewportOptions::default()).await?;`.
    pub async fn new_webgpu_async(options: ViewportOptions) -> Result<Self, String> {
        let driver = gui_outcomes::webgpu_gui_driver_async(options).await?;
        Ok(Self {
            state: Rc::new(RuntimeState {
                driver,
                #[cfg(target_os = "linux")]
                linux: None,
                #[cfg(target_os = "windows")]
                native_windows: false,
                next_view: Cell::new(1),
            }),
        })
    }
    /// Synchronously initialize a WebGPU-backed viewport runtime on native platforms.
    /// Example: `let runtime = ViewportRuntime::new_webgpu(ViewportOptions::default())?;`.
    #[cfg(not(target_arch = "wasm32"))]
    pub fn new_webgpu(options: ViewportOptions) -> Result<Self, String> {
        let driver = gui_outcomes::webgpu_gui_driver_blocking(options)?;
        Ok(Self {
            state: Rc::new(RuntimeState {
                driver,
                #[cfg(target_os = "linux")]
                linux: None,
                #[cfg(target_os = "windows")]
                native_windows: false,
                next_view: Cell::new(1),
            }),
        })
    }
    /// Create an application-owned native target; example: `runtime.open_window("Oblikovati", [640,480])?`.
    pub fn open_window(&self, title: &str, extent: [u32; 2]) -> Result<Viewport, String> {
        self.open_native_target(title, extent, false)
    }
    /// Open a separately managed window with a system title bar.
    /// Example: `runtime.open_document_window("Drawing1", [640, 480])?`.
    pub fn open_document_window(&self, title: &str, extent: [u32; 2]) -> Result<Viewport, String> {
        self.open_native_target(title, extent, true)
    }
    fn open_native_target(
        &self,
        title: &str,
        extent: [u32; 2],
        system_frame: bool,
    ) -> Result<Viewport, String> {
        #[cfg(target_os = "linux")]
        let window = self
            .state
            .linux
            .as_ref()
            .ok_or("viewport runtime: expected a native Linux window backend")?
            .create(title, extent, system_frame)?;
        #[cfg(target_os = "windows")]
        let window = {
            if !self.state.native_windows {
                return Err("viewport runtime: expected a native Windows window backend".into());
            }
            NativeWindow::create(title, extent, system_frame)?
        };
        #[cfg(target_arch = "wasm32")]
        let window = {
            let _ = system_frame;
            NativeWindow::new_virtual(title, extent, 1.0)?
        };
        self.create(ViewTarget::Native(Box::new(window)))
    }
    /// Bind a worker-owned browser `OffscreenCanvas` for WebGPU presentation.
    /// Example: `let view = runtime.open_offscreen_canvas("#viewport", canvas, 2.0)?;`.
    #[cfg(target_arch = "wasm32")]
    pub fn open_offscreen_canvas(
        &self,
        canvas_id: &str,
        canvas: web_sys::OffscreenCanvas,
        device_pixel_ratio: f64,
    ) -> Result<Viewport, String> {
        let window = NativeWindow::from_offscreen_canvas(canvas_id, canvas, device_pixel_ratio)?;
        self.create(ViewTarget::Native(Box::new(window)))
    }
    /// Bind a main-thread browser `HtmlCanvasElement` for WebGPU presentation.
    /// Example: `let view = runtime.open_html_canvas("#viewport", canvas, 2.0)?;`.
    #[cfg(target_arch = "wasm32")]
    pub fn open_html_canvas(
        &self,
        canvas_id: &str,
        canvas: web_sys::HtmlCanvasElement,
        device_pixel_ratio: f64,
    ) -> Result<Viewport, String> {
        let window = NativeWindow::from_html_canvas(canvas_id, canvas, device_pixel_ratio)?;
        self.create(ViewTarget::Native(Box::new(window)))
    }
    /// Set up a native Windows window before it becomes visible.
    /// Example: `runtime.open_window_with_setup("Oblikovati", [640, 480], |_| {})?`.
    #[cfg(target_os = "windows")]
    pub fn open_window_with_setup(
        &self,
        title: &str,
        extent: [u32; 2],
        setup: impl FnOnce(isize),
    ) -> Result<Viewport, String> {
        if !self.state.native_windows {
            return Err("viewport runtime: expected a native Windows window backend".into());
        }
        let window = NativeWindow::create_with_setup(title, extent, setup)?;
        self.create(ViewTarget::Native(Box::new(window)))
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
            gui_surface: None,
            view_history: VecDeque::new(),
            layout_framing: None,
            projection_cache: RefCell::new(projection_cache::ProjectionCache::default()),
            #[cfg(target_os = "linux")]
            preview_rgb: Default::default(),
        })
    }
    /// Name the selected device; example: `runtime.device_name()` for diagnostics.
    pub fn device_name(&self) -> String {
        self.state.driver.device_name()
    }
    /// Read complete kernel Vulkan validation observations; example: `runtime.validation_errors()?` after viewport acceptance.
    pub fn validation_errors(&self) -> Result<Vec<String>, String> {
        self.state
            .driver
            .validation_errors()
            .map_err(|cause| cause.to_string())
    }
}
