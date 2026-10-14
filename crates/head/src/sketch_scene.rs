// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{document_sketch::DocumentSketchId, history::DocumentSnapshot};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use oblikovati_viewport::ViewportSketchDisplay;
/// Authoring presentation is independent of solid feature evaluation and active edit state.
#[derive(Clone, Debug)]
pub struct HeadSketchSceneUpdate {
    pub source: Option<(u64, u64)>,
    pub sketches: Vec<ViewportSketchDisplay>,
    pub plane_declines: Vec<(DocumentSketchId, String)>,
}
#[derive(Default)]
pub(crate) struct DesktopSketchScenes {
    source: Option<(u64, u64)>,
}
impl DesktopSketchScenes {
    pub(crate) fn refresh<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &FileWorkspace<O, W, F>,
    ) -> Result<Option<HeadSketchSceneUpdate>, String> {
        let source = active_sketch_source(files)?;
        if source == self.source {
            return Ok(None);
        }
        let update = prepare_sketch_scene(files, source)?;
        self.source = source;
        Ok(Some(update))
    }
    pub(crate) fn clear(&mut self) {
        self.source = None;
    }
}
fn active_sketch_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Option<(u64, u64)>, String> {
    files
        .workspace()
        .active_document()
        .map(|id| {
            files
                .workspace()
                .session(id)
                .map(|session| (id, session.generation()))
        })
        .transpose()
        .map_err(|cause| cause.to_string())
}
fn prepare_sketch_scene<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    source: Option<(u64, u64)>,
) -> Result<HeadSketchSceneUpdate, String> {
    let mut update = HeadSketchSceneUpdate {
        source,
        sketches: Vec::new(),
        plane_declines: Vec::new(),
    };
    if let Some((id, _)) = source {
        let session = files
            .workspace()
            .session(id)
            .map_err(|cause| cause.to_string())?;
        collect_sketches(session.snapshot(), &mut update);
    }
    Ok(update)
}
fn collect_sketches(snapshot: &DocumentSnapshot, update: &mut HeadSketchSceneUpdate) {
    for sketch in snapshot.sketches.sketches() {
        match snapshot.resolve_sketch_plane(sketch.id()) {
            Ok(plane) => update.sketches.push(ViewportSketchDisplay {
                sketch: sketch.id(),
                plane,
                geometry: sketch.geometry().clone(),
            }),
            Err(cause) => update.plane_declines.push((sketch.id(), cause.to_string())),
        }
    }
}

#[cfg(test)]
#[path = "../tests/sketch_scene.rs"]
mod tests;
