// SPDX-License-Identifier: GPL-2.0-only
//! Source inspection, live preview, edit seeds, and commit for Derived Part.
use super::*;
use oblikovati_automation::{
    DeriveInputs, PreparedDerivedPart, build_derived_part_definition,
    prepare_derived_part_definition, replace_derived_part,
};
use oblikovati_gui::{
    GuiDeriveAction, GuiDeriveEditSeed, GuiDerivePreview, GuiDerivePreviewState, GuiDeriveRequest,
    GuiDeriveSourceSummary,
};
use oblikovati_model::{
    feature::{FeatureId, derived_part::DerivedPartDefinition, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::{
    collections::BTreeMap,
    path::{Path, PathBuf},
};

#[derive(Clone)]
enum DeriveCommit {
    Create(Box<PreparedDerivedPart>),
    Edit(FeatureId, Box<DerivedPartDefinition>),
}

pub(in crate::owner) struct DerivePreview {
    request: GuiDeriveRequest,
    result: Result<DeriveCommit, String>,
}

#[derive(Default)]
pub(in crate::owner) struct DeriveOwnerState {
    pub(in crate::owner) preview: Option<DerivePreview>,
    pub(in crate::owner) source: Option<(u64, GuiDeriveSourceSummary)>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiDeriveEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::DerivedPart(definition) = feature.definition() else {
                return None;
            };
            Some((
                feature.id(),
                GuiDeriveEditSeed {
                    source_path: definition.source_path().to_string(),
                    source_name: definition.source_name().to_string(),
                    style: definition.style(),
                    include_bodies: definition.include_bodies(),
                    include_sketches: definition.include_sketches(),
                    include_work_geometry: definition.include_work_geometry(),
                    include_parameters: definition.include_parameters(),
                    scale: format_scale(definition.scale()),
                    mirror: definition.mirror(),
                    mirror_plane: definition.mirror_plane(),
                    link_face_color: definition.link_face_color(),
                    link_sketch_format: definition.link_sketch_format(),
                    show_all_objects: definition.show_all_objects(),
                    linked: definition.linked(),
                    model_state: definition.model_state().to_string(),
                    design_view: definition.design_view().to_string(),
                },
            ))
        })
        .collect()
}

fn format_scale(value: f64) -> String {
    let text = format!("{value:.6}");
    let trimmed = text.trim_end_matches('0').trim_end_matches('.');
    if trimmed.is_empty() || trimmed == "-0" {
        "1".into()
    } else {
        trimmed.into()
    }
}

fn derive_inputs(request: &GuiDeriveRequest) -> DeriveInputs<'_> {
    DeriveInputs {
        source_path: &request.source_path,
        source_name: &request.source_name,
        style: request.style,
        include_bodies: request.include_bodies,
        include_sketches: request.include_sketches,
        include_work_geometry: request.include_work_geometry,
        include_parameters: request.include_parameters,
        scale: &request.scale,
        mirror: request.mirror,
        mirror_plane: request.mirror_plane,
        link_face_color: request.link_face_color,
        link_sketch_format: request.link_sketch_format,
        show_all_objects: request.show_all_objects,
        linked: request.linked,
        model_state: &request.model_state,
        design_view: &request.design_view,
    }
}

fn summarize_source(path: &str, snapshot: &DocumentSnapshot) -> GuiDeriveSourceSummary {
    let name = Path::new(path)
        .file_name()
        .and_then(|file| file.to_str())
        .unwrap_or("DerivedPart.opd")
        .to_string();
    let mut solid_bodies: Vec<String> = snapshot
        .work_geometry
        .features()
        .iter()
        .enumerate()
        .filter(|(_, feature)| {
            matches!(
                feature.definition(),
                PartFeatureDefinition::SketchPrism(_)
                    | PartFeatureDefinition::SketchRevolve(_)
                    | PartFeatureDefinition::SketchSweep(_)
                    | PartFeatureDefinition::SketchLoft(_)
                    | PartFeatureDefinition::SketchCoil(_)
                    | PartFeatureDefinition::DerivedPart(_)
            )
        })
        .map(|(index, _)| format!("Solid{}", index + 1))
        .collect();
    if solid_bodies.is_empty() && !snapshot.work_geometry.features().is_empty() {
        solid_bodies.push("Solid1".into());
    }
    let sketches = snapshot
        .sketches
        .sketches()
        .iter()
        .map(|sketch| sketch.name().to_string())
        .collect();
    let mut work_geometry = vec![
        "YZ Plane".into(),
        "XZ Plane".into(),
        "XY Plane".into(),
        "X Axis".into(),
        "Y Axis".into(),
        "Z Axis".into(),
        "Center Point".into(),
    ];
    for feature in snapshot.work_geometry.features() {
        if matches!(
            feature.definition(),
            PartFeatureDefinition::Fixed(_)
                | PartFeatureDefinition::Offset { .. }
                | PartFeatureDefinition::Axis(_)
                | PartFeatureDefinition::Point(_)
        ) {
            work_geometry.push(feature.name().to_string());
        }
    }
    let parameters = snapshot
        .parameters
        .ids()
        .iter()
        .filter_map(|id| snapshot.parameters.get(*id).map(|param| param.name.clone()))
        .collect();
    let model_states = snapshot
        .model_states
        .as_ref()
        .map(|states| {
            states
                .active()
                .map(|active| vec![active.name.clone()])
                .unwrap_or_default()
        })
        .filter(|list| !list.is_empty())
        .unwrap_or_else(|| vec!["Primary".into()]);
    let design_views = snapshot
        .part_design_views
        .as_ref()
        .map(|views| {
            use oblikovati_api::contract::PartDesignView as _;
            views
                .list()
                .iter()
                .map(|view| view.name().to_string())
                .collect::<Vec<_>>()
        })
        .filter(|list| !list.is_empty())
        .unwrap_or_else(|| vec!["Primary".into()]);
    GuiDeriveSourceSummary {
        path: path.to_string(),
        name,
        solid_bodies,
        sketches,
        work_geometry,
        parameters,
        model_states,
        design_views,
    }
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn gui_derive_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDerivePreviewState> {
        let preview = self.derive_state.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiDerivePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(_) => GuiDerivePreview::Ready,
                    Err(cause) => GuiDerivePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_derive_source(
        &self,
        document: u64,
    ) -> Option<GuiDeriveSourceSummary> {
        self.derive_state
            .source
            .as_ref()
            .filter(|(doc, _)| *doc == document)
            .map(|(_, summary)| summary.clone())
    }

    pub(in crate::owner) fn derive_command(
        &mut self,
        action: GuiDeriveAction,
    ) -> Result<bool, String> {
        match action {
            GuiDeriveAction::Open | GuiDeriveAction::OpenEdit(_) => Ok(false),
            GuiDeriveAction::Clear => {
                let cleared = self.derive_state.preview.take().is_some()
                    | self.derive_state.source.take().is_some()
                    | self.clear_extrusion_preview();
                Ok(cleared)
            }
            GuiDeriveAction::InspectSource { document, path } => {
                self.require_browser_document(document)?;
                let snapshot = self
                    .files
                    .load_derived_source_snapshot(&path, Some(document))?;
                let summary = summarize_source(&path, &snapshot);
                self.derive_state.source = Some((document, summary));
                Ok(true)
            }
            GuiDeriveAction::OpenSource(feature) => self.open_derive_source(feature),
            GuiDeriveAction::Preview(request) => {
                let request = *request;
                self.require_derive_generation(&request)?;
                let result = self.preview_derive(&request);
                self.derive_state.preview = Some(DerivePreview { request, result });
                Ok(true)
            }
            GuiDeriveAction::Commit(request) => {
                let request = *request;
                self.require_derive_generation(&request)?;
                let commit = self
                    .derive_state
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .ok_or("derive: expected a current certified preview")?
                    .clone();
                self.commit_derive(request.document, &request.source_path, commit)?;
                self.derive_state.preview = None;
                self.derive_state.source = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_derive_generation(&self, request: &GuiDeriveRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("derive: expected the current document generation".into());
        }
        Ok(())
    }

    fn clear_derive_scene(&mut self) {
        if self.extrusion_preview.take().is_some() {
            self.part_scenes.invalidate_presentation();
        }
    }

    fn preview_derive(&mut self, request: &GuiDeriveRequest) -> Result<DeriveCommit, String> {
        self.files
            .sync_derived_sources(request.document, &[&request.source_path])?;
        let outcome = match request.feature {
            Some(feature) => self.preview_derive_edit(request, feature),
            None => self.preview_derive_create(request),
        };
        if outcome.is_err() {
            self.clear_derive_scene();
        }
        outcome
    }

    fn preview_derive_create(
        &mut self,
        request: &GuiDeriveRequest,
    ) -> Result<DeriveCommit, String> {
        let document = request.document;
        let prepared = {
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            prepare_derived_part_definition(snapshot, derive_inputs(request))
                .map_err(|cause| cause.to_string())?
        };
        if request.preview_hidden {
            self.clear_derive_scene();
        } else {
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
        }
        Ok(DeriveCommit::Create(Box::new(prepared)))
    }

    fn preview_derive_edit(
        &mut self,
        request: &GuiDeriveRequest,
        id: FeatureId,
    ) -> Result<DeriveCommit, String> {
        let edited = build_derived_part_definition(derive_inputs(request))
            .map_err(|cause| cause.to_string())?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let projection = session
            .preview_projected(
                |candidate, _| replace_derived_part(candidate, id, edited.clone()),
                |candidate, ()| {
                    Ok(self.part_scenes.project_feature_edit(
                        &self.files,
                        request.document,
                        candidate,
                        id,
                    ))
                },
            )
            .map_err(|cause| cause.to_string())?;
        match projection {
            Ok(scene) => {
                if request.preview_hidden {
                    self.clear_derive_scene();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Replacement,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(DeriveCommit::Edit(id, Box::new(edited)))
            }
            Err(cause) => Err(cause.to_string()),
        }
    }

    fn commit_derive(
        &mut self,
        document: u64,
        source_path: &str,
        commit: DeriveCommit,
    ) -> Result<(), String> {
        self.files.sync_derived_sources(document, &[source_path])?;
        match commit {
            DeriveCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Derive",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            DeriveCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Derived Part",
                    |snapshot, _| replace_derived_part(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }

    fn open_derive_source(&mut self, feature_id: FeatureId) -> Result<bool, String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or("derive open source: expected an active document")?;
        let source_path = {
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            let feature = snapshot
                .work_geometry
                .get(feature_id)
                .map_err(|cause| cause.to_string())?;
            let PartFeatureDefinition::DerivedPart(definition) = feature.definition() else {
                return Err(format!("feature {feature_id:?}: expected a Derived Part"));
            };
            definition.source_path().to_string()
        };
        let resolved = resolve_source_path(&self.files, document, &source_path);
        let opened = self
            .document_host()
            .open_path(&resolved, true)
            .map_err(|cause| cause.to_string())?;
        self.activate_gui_document(&oblikovati_api::wire::ActivateDocumentArgs { id: opened })?;
        Ok(true)
    }
}

fn resolve_source_path<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &oblikovati_session::files::FileWorkspace<O, W, F>,
    document: u64,
    source_path: &str,
) -> PathBuf {
    let raw = PathBuf::from(source_path);
    if raw.is_absolute() {
        return raw;
    }
    if let Ok(owner_path) = files.path(document)
        && let Some(parent) = owner_path.parent()
    {
        return parent.join(raw);
    }
    raw
}
