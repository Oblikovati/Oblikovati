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
        let options = ViewportOptions {
            #[cfg(target_os = "windows")]
            validation: std::env::var_os("OBLIKOVATI_VULKAN_VALIDATION")
                .is_some_and(|value| value == "1"),
            ..ViewportOptions::default()
        };
        let runtime = ViewportRuntime::new(options)?;
        let mut view = runtime.open_window("Oblikovati", extent)?;
        view.submit(&ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        })?;
        #[cfg(target_os = "linux")]
        let clipboard = clipboard::NativeClipboardChannel::open();
        #[cfg(target_os = "windows")]
        let clipboard = clipboard::NativeClipboardChannel::open(view.native_clipboard()?);
        Ok(Self {
            capture_files,
            capture_requests: Vec::new(),
            exit_requested: false,
            runtime,
            validate_vulkan: options.validation,
            view,
            view_projection: Default::default(),
            marked_view: None,
            wake: ViewportWake::new()?,
            scene_status: None,
            current_scene: None,
            scene_depth_bounds: None,
            preview_background: None,
            sketch_scene: None,
            glyph_hover: None,
            tool_hover: None,
            datum_declines: Vec::new(),
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
            clipboard,
            #[cfg(test)]
            _context_lease: context_lease,
        })
    }
}
