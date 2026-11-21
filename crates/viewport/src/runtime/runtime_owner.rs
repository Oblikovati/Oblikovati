// SPDX-License-Identifier: GPL-2.0-only
//! Creation and ownership of renderer-backed views.
use super::*;

impl ViewportRuntime {
    /// Initialize the owner-thread runtime; example: `ViewportRuntime::new(ViewportOptions::default())?`.
    pub fn new(options: ViewportOptions) -> Result<Self, String> {
        #[cfg(target_os = "linux")]
        let x11 = NativeX11::open()?;
        let driver = gui_outcomes::native_gui_driver(options)?;
        Ok(Self {
            state: Rc::new(RuntimeState {
                driver,
                #[cfg(target_os = "linux")]
                x11: Some(x11),
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
        let window = NativeWindow::create(
            self.state
                .x11
                .as_ref()
                .ok_or("viewport runtime: expected native X11 support")?
                .clone(),
            title,
            extent,
            system_frame,
        )?;
        #[cfg(target_os = "windows")]
        let window = NativeWindow::create(title, extent, system_frame)?;
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
            projection_cache: RefCell::new(projection_cache::ProjectionCache::default()),
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
