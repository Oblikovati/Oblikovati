// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_viewport::ViewportSketchDisplay;
/// Authoring presentation is independent of solid feature evaluation and active edit state.
#[derive(Clone, Debug)]
pub struct HeadSketchSceneUpdate {
    pub source: Option<(u64, u64)>,
    pub sketches: Vec<ViewportSketchDisplay>,
    pub plane_declines: Vec<(DocumentSketchId, String)>,
}

#[cfg(target_os = "linux")]
#[path = "sketch_scene_runtime.rs"]
mod runtime;
#[cfg(target_os = "linux")]
pub(crate) use runtime::DesktopSketchScenes;
