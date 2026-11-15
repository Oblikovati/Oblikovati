// SPDX-License-Identifier: GPL-2.0-only
//! Construct the native window and its owner-thread services.
use super::*;
use oblikovati_viewport::ViewportOptions;

impl NativeHeadWindow {
    /// Open an owner-thread desktop target; example: `let window = NativeHeadWindow::open()?;`.
    pub fn open() -> Result<Self, String> {
        Self::open_with_capture_files(Box::new(crate::NativeWindowCaptureFiles))
    }
    /// Inject PNG publication while retaining the same native renderer.
    /// Example: `NativeHeadWindow::open_with_capture_files(Box::new(files))?`.
    pub fn open_with_capture_files(
        capture_files: Box<dyn crate::WindowCaptureFiles>,
    ) -> Result<Self, String> {
        Self::open_with_options([1440, 900], capture_files)
    }
    /// Open a window at the requested drawable size.
    /// Example: `NativeHeadWindow::open_with_extent([800, 600])?`.
    pub fn open_with_extent(extent: [u32; 2]) -> Result<Self, String> {
        Self::open_with_options(extent, Box::new(crate::NativeWindowCaptureFiles))
    }
    fn open_with_options(
        extent: [u32; 2],
        capture_files: Box<dyn crate::WindowCaptureFiles>,
    ) -> Result<Self, String> {
        #[cfg(test)]
        let context_lease = crate::test_context::acquire();
        validate_window_extent(extent)?;
        let runtime = ViewportRuntime::new(ViewportOptions::default())?;
        let mut view = runtime.open_window("Oblikovati", extent)?;
        view.submit(&ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        })?;
        Ok(Self {
            capture_files,
            capture_requests: Vec::new(),
            exit_requested: false,
            runtime,
            view,
            view_projection: Default::default(),
            marked_view: None,
            wake: ViewportWake::new()?,
            scene_status: None,
            current_scene: None,
            preview_background: None,
            sketch_scene: None,
            navigation: Default::default(),
            sketch_view: None,
            document_title: None,
            view_slot: 0,
            saved_views: Default::default(),
            command_notice: None,
            gui: None,
            injected_events: Default::default(),
            ribbon_settings_directory: None,
            gui_epoch: std::time::Instant::now(),
            clipboard: clipboard::NativeClipboardChannel::open(),
            #[cfg(test)]
            _context_lease: context_lease,
        })
    }
}
