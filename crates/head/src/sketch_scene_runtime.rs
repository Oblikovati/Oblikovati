// SPDX-License-Identifier: GPL-2.0-only
use super::HeadSketchSceneUpdate;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_model::history::DocumentSnapshot;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use oblikovati_viewport::{ViewportSketchBlock, ViewportSketchCurvature, ViewportSketchDisplay};
use std::borrow::Cow;
pub(crate) type ActiveCurvature = Option<(DocumentSketchId, Vec<ViewportSketchCurvature>)>;
#[derive(Default)]
pub(crate) struct DesktopSketchScenes {
    source: Option<(u64, u64)>,
    file_name: String,
    /// The sketches shown although hidden; a new choice redraws without any model change.
    called_up: Vec<DocumentSketchId>,
    curvature: ActiveCurvature,
    hide_overlays: bool,
}
impl DesktopSketchScenes {
    pub(crate) fn refresh<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        parts: &mut crate::part_scene::DesktopPartScenes,
        opened_hidden: Option<DocumentSketchId>,
        active_sketch: Option<DocumentSketchId>,
        curvature: ActiveCurvature,
    ) -> Result<Option<HeadSketchSceneUpdate>, String> {
        let source = active_sketch_source(files)?;
        let called_up = self.called_up(files, source, opened_hidden, active_sketch);
        if source == self.source
            && called_up == self.called_up
            && curvature == self.curvature
            && source.map_or(Cow::Borrowed(""), |(id, _)| {
                sketch_document_file_name(files, id)
            }) == self.file_name
        {
            return Ok(None);
        }
        let datums = if self.hide_overlays && active_sketch.is_none() {
            None
        } else {
            scene_datums(files, parts, source)?
        };
        let source = active_sketch_source(files)?;
        let called_up = self.called_up(files, source, opened_hidden, active_sketch);
        let mut update = prepare_sketch_scene(
            files,
            source,
            &called_up,
            datums.as_ref(),
            self.hide_overlays,
        )?;
        if let Some((id, requests)) = &curvature
            && let Some(sketch) = update
                .sketches
                .iter_mut()
                .find(|sketch| sketch.sketch == *id)
        {
            sketch.curvature = requests.clone();
        }
        self.source = source;
        self.called_up = called_up;
        self.curvature = curvature;
        self.file_name = source.map_or_else(String::new, |(id, _)| {
            sketch_document_file_name(files, id).into_owned()
        });
        Ok(Some(update))
    }

    fn called_up<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        source: Option<(u64, u64)>,
        opened_hidden: Option<DocumentSketchId>,
        active_sketch: Option<DocumentSketchId>,
    ) -> Vec<DocumentSketchId> {
        if self.hide_overlays {
            active_sketch.into_iter().collect()
        } else {
            called_up_sketches(files, source, opened_hidden)
        }
    }

    pub(crate) fn set_overlays_visible(&mut self, visible: bool) -> bool {
        let hide = !visible;
        if self.hide_overlays == hide {
            return false;
        }
        self.hide_overlays = hide;
        self.clear();
        true
    }

    pub(crate) fn overlays_visible(&self) -> bool {
        !self.hide_overlays
    }
    pub(crate) fn clear(&mut self) {
        self.source = None;
        self.file_name.clear();
    }
}

/// Read the file-name field source. For example, a saved `/tmp/Part.opd` uses `Part.opd`.
pub(crate) fn sketch_document_file_name<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Cow<'_, str> {
    if let Some(name) = files
        .path(document)
        .ok()
        .and_then(std::path::Path::file_name)
    {
        return name.to_string_lossy();
    }
    Cow::Borrowed(files.workspace().display_name(document).unwrap_or_default())
}
fn scene_datums<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    parts: &mut crate::part_scene::DesktopPartScenes,
    source: Option<(u64, u64)>,
) -> Result<Option<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation>, String> {
    use oblikovati_automation::PartUpdateAccess;
    let Some((id, _)) = source else {
        return Ok(None);
    };
    let snapshot = files
        .workspace()
        .session(id)
        .map_err(|error| error.to_string())?
        .snapshot();
    if !oblikovati_automation::requires_part_replay(snapshot) {
        return Ok(None);
    }
    parts.datum_evaluation(files, id).map(Some)
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
fn called_up_sketches<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    source: Option<(u64, u64)>,
    opened_hidden: Option<DocumentSketchId>,
) -> Vec<DocumentSketchId> {
    let session = source.and_then(|(id, _)| files.workspace().session(id).ok());
    let Some(session) = session else {
        return Vec::new();
    };
    // Only a hidden sketch needs calling up: choosing one in view must not redraw the scene.
    let snapshot = session.snapshot();
    let mut called_up: Vec<_> = session
        .selected_browser_sketches()
        .into_iter()
        .filter(|sketch| {
            snapshot
                .sketches
                .get(*sketch)
                .is_ok_and(|placed| !placed.visible())
        })
        .collect();
    called_up.extend(opened_hidden);
    called_up
}
fn prepare_sketch_scene<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    source: Option<(u64, u64)>,
    called_up: &[DocumentSketchId],
    datums: Option<&oblikovati_model::feature::work_geometry::WorkGeometryEvaluation>,
    hide_overlays: bool,
) -> Result<HeadSketchSceneUpdate, String> {
    let mut update = HeadSketchSceneUpdate {
        source,
        sketches: Vec::new(),
        plane_declines: Vec::new(),
        block_declines: Vec::new(),
    };
    if let Some((id, _)) = source {
        let session = files
            .workspace()
            .session(id)
            .map_err(|cause| cause.to_string())?;
        let file_name = sketch_document_file_name(files, id);
        collect_sketches(
            session.snapshot(),
            &file_name,
            called_up,
            datums,
            hide_overlays,
            &mut update,
        );
    }
    Ok(update)
}
fn line_color([red, green, blue]: [u8; 3]) -> oblikovati_model::theme::Rgba {
    let [r, g, b] = [red, green, blue].map(|channel| f32::from(channel) / 255.);
    oblikovati_model::theme::Rgba { r, g, b, a: 1. }
}
fn collect_sketches(
    snapshot: &DocumentSnapshot,
    file_name: &str,
    called_up: &[DocumentSketchId],
    datums: Option<&oblikovati_model::feature::work_geometry::WorkGeometryEvaluation>,
    hide_overlays: bool,
    update: &mut HeadSketchSceneUpdate,
) {
    let frames = datums.and_then(|datums| snapshot.bind_sketch_frames(datums).ok());
    for sketch in snapshot
        .sketches
        .sketches()
        .iter()
        .filter(|sketch| super::sketch_shows(sketch, called_up, !hide_overlays))
    {
        let plane = match (datums, &frames) {
            (Some(_), Some(frames)) => frames.resolve(sketch.id()),
            (Some(datums), None) => snapshot.resolve_sketch_plane_with_datums(sketch.id(), datums),
            (None, _) => snapshot.resolve_sketch_plane(sketch.id()),
        };
        match plane {
            Ok(plane) => {
                let blocks = scene_blocks(snapshot, file_name, sketch, update);
                let (geometry, text_declines) = sketch
                    .geometry()
                    .text_display_geometry(&snapshot.parameters, file_name);
                update.sketches.push(ViewportSketchDisplay {
                    working_scale: snapshot.parameters.units().working_scale(),
                    sketch: sketch.id(),
                    plane,
                    geometry,
                    text_declines,
                    blocks,
                    curvature: Vec::new(),
                    line_color: sketch.line_style().color.map(line_color),
                })
            }
            Err(cause) => update.plane_declines.push((sketch.id(), cause.to_string())),
        }
    }
}
fn scene_blocks(
    snapshot: &DocumentSnapshot,
    file_name: &str,
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    update: &mut HeadSketchSceneUpdate,
) -> Vec<ViewportSketchBlock> {
    let mut blocks = Vec::new();
    for instance in sketch.block_instances() {
        let Some(definition) = snapshot
            .sketches
            .block_definitions()
            .iter()
            .find(|item| item.name() == instance.definition())
        else {
            update.block_declines.push((
                sketch.id(),
                format!(
                    "block instance {}: missing definition {:?}",
                    instance.id().0,
                    instance.definition()
                ),
            ));
            continue;
        };
        let (geometry, text_declines) = definition
            .geometry()
            .text_display_geometry(&snapshot.parameters, file_name);
        blocks.push(ViewportSketchBlock {
            instance: instance.clone(),
            geometry,
            text_declines,
            definition_color: definition.color().cloned(),
        });
    }
    blocks
}

#[cfg(test)]
#[path = "../tests/sketch_scene.rs"]
mod tests;
