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
            validation: native_validation(
                std::env::var_os("OBLIKOVATI_VULKAN_VALIDATION").as_deref(),
            ),
            ..ViewportOptions::default()
        };
        let runtime = ViewportRuntime::new(options)?;
        #[cfg(target_os = "linux")]
        if std::env::var_os("OBLIKOVATI_VSYNC").is_some_and(|value| value == "0") {
            runtime.set_vertical_sync(false);
        }
        let wake = ViewportWake::new()?;
        #[cfg(target_os = "linux")]
        let mut view = runtime.open_window("Oblikovati", extent)?;
        #[cfg(target_os = "windows")]
        let (mut view, accessibility) = {
            let mut adapter = None;
            let signal = wake.clone();
            let view = runtime.open_window_with_setup("Oblikovati", extent, |hwnd| {
                adapter = Some(
                    crate::accessibility::windows::WindowsFrameAccessibility::new(
                        hwnd,
                        move || {
                            let _ = signal.wake();
                        },
                    ),
                );
            })?;
            (view, adapter)
        };
        view.submit(&ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        })?;
        #[cfg(target_os = "linux")]
        let clipboard = clipboard::NativeClipboardChannel::open();
        #[cfg(target_os = "windows")]
        let clipboard = clipboard::NativeClipboardChannel::open(view.native_clipboard()?);
        #[cfg(target_os = "linux")]
        let accessibility_bounds = view.window_screen_bounds()?;
        #[cfg(target_os = "linux")]
        let accessibility = {
            let signal = wake.clone();
            crate::accessibility::unix::UnixFrameAccessibility::new(move || {
                let _ = signal.wake();
            })
        };
        Ok(Self {
            capture_files,
            capture_requests: Vec::new(),
            exit_requested: false,
            runtime,
            validate_vulkan: options.validation,
            view,
            orientation_transition: None,
            drawing_windows: Default::default(),
            view_projection: Default::default(),
            marked_view: None,
            wake,
            scene_status: None,
            current_scene: None,
            scene_depth_bounds: None,
            presentation_bounds: None,
            #[cfg(test)]
            presentation_bounds_refreshes: 0,
            preview_background: None,
            sketch_scene: None,
            glyph_hover: None,
            glyph_target: None,
            glyph_candidates: None,
            tool_hover: None,
            datum_declines: Vec::new(),
            navigation: Default::default(),
            sketch_view: None,
            document_title: None,
            status_pointer: None,
            status_projection: Default::default(),
            view_slot: 0,
            saved_views: Default::default(),
            command_notice: None,
            gui: None,
            accessibility,
            #[cfg(target_os = "linux")]
            accessibility_bounds,
            accessibility_title: "Oblikovati".into(),
            injected_events: Default::default(),
            ribbon_settings_directory: None,
            gui_epoch: std::time::Instant::now(),
            clipboard,
            #[cfg(test)]
            _context_lease: context_lease,
        })
    }
}

fn native_validation(setting: Option<&std::ffi::OsStr>) -> bool {
    match setting.and_then(std::ffi::OsStr::to_str) {
        Some("1") => true,
        Some("0") => false,
        _ => cfg!(all(target_os = "linux", debug_assertions)),
    }
}

#[cfg(test)]
mod tests {
    use super::native_validation;
    use std::ffi::OsStr;

    #[test]
    fn native_validation_can_be_overridden_in_both_build_profiles() {
        assert!(native_validation(Some(OsStr::new("1"))));
        assert!(!native_validation(Some(OsStr::new("0"))));
        assert_eq!(
            native_validation(None),
            cfg!(all(target_os = "linux", debug_assertions))
        );
    }
}
