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
/// Example: `sketch_shows(sketch, &session.selected_browser_sketches())`.
#[cfg(any(target_os = "linux", target_os = "windows"))]
pub(crate) fn sketch_shows(
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    called_up: &[DocumentSketchId],
) -> bool {
    sketch.visible() || called_up.contains(&sketch.id())
}

#[cfg(any(target_os = "linux", target_os = "windows"))]
#[path = "sketch_scene_runtime.rs"]
mod runtime;
#[cfg(any(target_os = "linux", target_os = "windows"))]
pub(crate) use runtime::DesktopSketchScenes;
