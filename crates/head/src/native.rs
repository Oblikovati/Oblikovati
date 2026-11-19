// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow};
#[allow(unsafe_code)]
#[path = "native/addin_launch.rs"]
mod addin_launch;
#[path = "native_preferences.rs"]
mod preferences;
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportPresentation, ViewportRuntime, ViewportScene, ViewportWake,
};
mod saved_view;
use saved_view::{SavedWindowView, SketchCameraState};
pub struct NativeHeadWindow {
    capture_files: Box<dyn crate::WindowCaptureFiles>,
    capture_requests: Vec<crate::WindowCaptureJob>,
    exit_requested: bool,
    runtime: ViewportRuntime,
    validate_vulkan: bool,
    view: Viewport,
    view_projection: oblikovati_gui::GuiViewProjection,
    marked_view: Option<oblikovati_viewport::ViewportCamera>,
    wake: ViewportWake,
    scene_status: Option<crate::HeadSceneStatus>,
    current_scene: Option<ViewportScene>,
    scene_depth_bounds: Option<scene_depth::SceneBounds>,
    presentation_bounds: Option<[[f64; 3]; 2]>,
    #[cfg(test)]
    presentation_bounds_refreshes: usize,
    preview_background: Option<oblikovati_viewport::ViewportBackground>,
    sketch_scene: Option<crate::HeadSketchSceneUpdate>,
    glyph_hover: Option<oblikovati_viewport::ViewportSketchFeedback>,
    glyph_target: Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        oblikovati_model::sketch::SketchConstraintId,
    )>,
    glyph_candidates: Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        Vec<oblikovati_gui::GuiConstraintCandidate>,
    )>,
    tool_hover: Option<oblikovati_viewport::ViewportSketchFeedback>,
    datum_declines: Vec<(oblikovati_model::feature::FeatureId, String)>,
    navigation: crate::navigation::DesktopNavigation,
    sketch_view: Option<SketchCameraState>,
    /// The window of a document in front; each keeps its own view while another is shown.
    /// What the title bar names: the active document, with its window number when it has one.
    document_title: Option<String>,
    view_slot: u64,
    saved_views: std::collections::BTreeMap<u64, SavedWindowView>,
    command_notice: Option<crate::DesktopCommandNotice>,
    gui: Option<gui_loop::NativeGui>,
    #[cfg(target_os = "linux")]
    accessibility: crate::accessibility::unix::UnixFrameAccessibility,
    #[cfg(target_os = "windows")]
    accessibility: Option<crate::accessibility::windows::WindowsFrameAccessibility>,
    #[cfg(target_os = "linux")]
    accessibility_bounds: [i32; 4],
    accessibility_title: String,
    injected_events: std::collections::VecDeque<ViewportEvent>,
    ribbon_settings_directory: Option<std::path::PathBuf>,
    gui_epoch: std::time::Instant,
    clipboard: std::rc::Rc<clipboard::NativeClipboardChannel>,
    // Last so the GUI context drops before another test acquires the lease.
    #[cfg(test)]
    _context_lease: crate::test_context::TestContextLease,
}
mod open;
mod scene_depth;
mod scene_status;
#[macro_use]
#[path = "native/view_inputs.rs"]
mod view_inputs;
use view_inputs::validate_window_extent;
mod head_window;

mod capture;
#[cfg(target_os = "linux")]
mod clipboard;
#[cfg(target_os = "windows")]
#[path = "native/clipboard_windows.rs"]
mod clipboard;
mod owner;
use owner::{NativeDesktopOwner, NativeOwnerWake, native_owner_at, native_script_engine};
#[cfg(test)]
use owner::{NativeScriptDesktop, native_owner, native_script_owner};
mod run;
pub use run::{run_native, run_native_documents};
#[cfg(test)]
#[path = "../tests/navigation/native.rs"]
mod navigation_tests;
#[cfg(test)]
#[path = "../tests/native_owner.rs"]
mod owner_tests;
#[cfg(test)]
#[path = "../tests/native_scene.rs"]
mod scene_tests;
#[cfg(test)]
#[path = "../tests/native_startup.rs"]
mod startup_tests;
#[cfg(test)]
#[path = "../tests/native_wait.rs"]
mod tests;

#[cfg(test)]
#[allow(unsafe_code)]
#[path = "../tests/native_services.rs"]
mod addin_services_tests;
#[cfg(test)]
#[path = "../tests/native_history.rs"]
mod history_tests;
mod projection;
#[cfg(test)]
#[path = "../tests/native_save.rs"]
mod save_tests;
mod sketch_display;
mod sketch_edit;
mod title;

mod gui_composition;
#[cfg(test)]
#[path = "../tests/native_sketch_display.rs"]
mod sketch_display_tests;

mod accessibility;
#[cfg(test)]
#[path = "../tests/native_gui_composition.rs"]
mod gui_composition_tests;
mod gui_loop;
#[cfg(all(test, target_os = "linux"))]
#[path = "../tests/native_gui_loop.rs"]
mod gui_loop_tests;
