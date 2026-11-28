// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_viewport::ViewportSketchDisplay;
/// Authoring presentation is independent of solid feature evaluation and active edit state.
#[derive(Clone, Debug)]
pub struct HeadSketchSceneUpdate {
    pub source: Option<(u64, u64)>,
    pub sketches: Vec<ViewportSketchDisplay>,
    pub plane_declines: Vec<(DocumentSketchId, String)>,
    /// Block instances whose definitions cannot be resolved for display.
    pub block_declines: Vec<(DocumentSketchId, String)>,
}
/// A hidden sketch still shows while it is chosen in the browser or edited.
/// Example: `sketch_shows(sketch, &session.selected_browser_sketches(), true)`.
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub(crate) fn sketch_shows(
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    called_up: &[DocumentSketchId],
    show_stored: bool,
) -> bool {
    (show_stored && sketch.visible()) || called_up.contains(&sketch.id())
}

#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
#[path = "sketch_scene_runtime.rs"]
mod runtime;
#[cfg(any(target_os = "linux", target_os = "windows", target_arch = "wasm32"))]
pub(crate) use runtime::{ActiveCurvature, DesktopSketchScenes, sketch_document_file_name};
