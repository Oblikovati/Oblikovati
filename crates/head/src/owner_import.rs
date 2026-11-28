// SPDX-License-Identifier: GPL-2.0-only
//! Source inspection, preview, edit seeds, link commands, and commit for desktop Import.
use super::*;
use oblikovati_automation::{
    ImportInputs, PreparedImport, build_import_definition, prepare_import_definition,
    replace_imported_model,
};
use oblikovati_gui::{
    GuiImportAction, GuiImportEditSeed, GuiImportPreview, GuiImportPreviewState, GuiImportRequest,
    GuiImportSourceSummary,
};
use oblikovati_martensite::import::{inspect_step, inspect_stl};
use oblikovati_model::{
    feature::{
        FeatureId,
        imported_model::{ImportedFileKind, ImportedModelDefinition},
        part::PartFeatureDefinition,
    },
    history::DocumentSnapshot,
};
use std::{collections::BTreeMap, path::Path};

#[derive(Clone)]
enum ImportCommit {
    Create(Box<PreparedImport>),
    Edit(FeatureId, Box<ImportedModelDefinition>),
}

pub(in crate::owner) struct ImportPreview {
    request: GuiImportRequest,
    result: Result<ImportCommit, String>,
}

#[derive(Default)]
pub(in crate::owner) struct ImportOwnerState {
    pub(in crate::owner) source: Option<(u64, GuiImportSourceSummary)>,
    pub(in crate::owner) preview: Option<ImportPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiImportEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::ImportedModel(definition) = feature.definition() else {
                return None;
            };
            Some((
                feature.id(),
                GuiImportEditSeed {
                    source_path: definition.source_path().into(),
                    source_name: definition.source_name().into(),
                    file_kind: definition.file_kind(),
                    import_mode: definition.import_mode(),
                    body_names: definition.body_names().to_vec(),
                    link_suppressed: definition.link_suppressed(),
                    link_broken: definition.link_broken(),
                    stl_options: definition.stl_options().cloned().unwrap_or_default(),
                    step_options: definition.step_options().cloned().unwrap_or_default(),
                },
            ))
        })
        .collect()
}

pub(in crate::owner) fn third_party_entries(snapshot: &DocumentSnapshot) -> Vec<String> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::ImportedModel(definition) = feature.definition() else {
                return None;
            };
            (definition.file_kind() == ImportedFileKind::Stl).then(|| feature.name().into())
        })
        .collect()
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
    pub(in crate::owner) fn gui_import_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiImportPreviewState> {
        let preview = self.create_features.import.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiImportPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(_) => GuiImportPreview::Ready,
                    Err(cause) => GuiImportPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_import_source(
        &self,
        document: u64,
    ) -> Option<GuiImportSourceSummary> {
        self.create_features
            .import
            .source
            .as_ref()
            .filter(|(owner, _)| *owner == document)
            .map(|(_, summary)| summary.clone())
    }

    pub(in crate::owner) fn import_command(
        &mut self,
        action: GuiImportAction,
    ) -> Result<bool, String> {
        match action {
            GuiImportAction::Open | GuiImportAction::OpenEdit(_) => Ok(false),
            GuiImportAction::Clear => {
                let had_state = self.create_features.import.preview.take().is_some()
                    | self.create_features.import.source.take().is_some();
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiImportAction::InspectSource { document, path } => {
                self.require_browser_document(document)?;
                let summary = inspect_exchange_file(&path)?;
                self.create_features.import.source = Some((document, summary));
                Ok(true)
            }
            GuiImportAction::Preview(request) => {
                let request = *request;
                self.require_import_generation(&request)?;
                let result = self.preview_import(&request);
                self.create_features.import.preview = Some(ImportPreview { request, result });
                Ok(true)
            }
            GuiImportAction::Commit(request) => {
                let request = *request;
                self.require_import_generation(&request)?;
                let commit = if request.file_kind == ImportedFileKind::Stl {
                    self.build_import_commit(&request)?
                } else {
                    self.create_features
                        .import
                        .preview
                        .as_ref()
                        .filter(|preview| preview.request == request)
                        .and_then(|preview| preview.result.as_ref().ok())
                        .cloned()
                        .ok_or("import: expected a current certified preview")?
                };
                self.commit_import(request.document, commit)?;
                self.create_features.import.preview = None;
                self.create_features.import.source = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
            GuiImportAction::SetLinkSuppressed {
                document,
                feature,
                suppressed,
            } => {
                self.update_import_link(document, feature, |def| {
                    def.with_link_suppressed(suppressed)
                })?;
                Ok(true)
            }
            GuiImportAction::BreakLink { document, feature } => {
                self.update_import_link(document, feature, |def| def.with_link_broken())?;
                Ok(true)
            }
        }
    }

    fn require_import_generation(&self, request: &GuiImportRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("import: expected the current document generation".into());
        }
        Ok(())
    }

    fn build_import_commit(&self, request: &GuiImportRequest) -> Result<ImportCommit, String> {
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let template_scale = snapshot.parameters.units().working_scale();
        let unit_scale = match request.file_kind {
            ImportedFileKind::Stl => request.stl_options.units.scale_to_cm(template_scale),
            ImportedFileKind::Step => 0.1,
        };
        let inputs = ImportInputs {
            source_path: &request.source_path,
            source_name: &request.source_name,
            file_kind: request.file_kind,
            import_mode: request.import_mode,
            body_names: request.body_names.clone(),
            link_suppressed: false,
            link_broken: false,
            unit_scale,
            stl_options: (request.file_kind == ImportedFileKind::Stl)
                .then(|| request.stl_options.clone()),
            step_options: (request.file_kind == ImportedFileKind::Step)
                .then(|| request.step_options.clone()),
        };
        match request.feature {
            Some(id) => {
                let definition =
                    build_import_definition(inputs).map_err(|cause| cause.to_string())?;
                Ok(ImportCommit::Edit(id, Box::new(definition)))
            }
            None => {
                let prepared = prepare_import_definition(snapshot, inputs)
                    .map_err(|cause| cause.to_string())?;
                Ok(ImportCommit::Create(Box::new(prepared)))
            }
        }
    }

    fn preview_import(&mut self, request: &GuiImportRequest) -> Result<ImportCommit, String> {
        let commit = self.build_import_commit(request)?;
        if !request.show_viewport_preview {
            self.clear_extrusion_preview();
            return Ok(commit);
        }
        match &commit {
            ImportCommit::Create(prepared) => {
                let preview = prepared.clone();
                self.preview_added_replacement(
                    request.document,
                    super::super::owner_extrusion_preview::PreviewKind::Additive,
                    |candidate, ids| preview.create(candidate, ids),
                )?;
            }
            ImportCommit::Edit(id, definition) => {
                let id = *id;
                let definition = definition.clone();
                let session = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?;
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_imported_model(candidate, id, *definition),
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
                        self.extrusion_preview = Some((
                            request.document,
                            request.generation,
                            scene,
                            super::super::owner_extrusion_preview::PreviewKind::Replacement,
                        ));
                        self.part_scenes.invalidate_presentation();
                    }
                    Err(cause) => return Err(cause.to_string()),
                }
            }
        }
        Ok(commit)
    }

    fn commit_import(&mut self, document: u64, commit: ImportCommit) -> Result<(), String> {
        match commit {
            ImportCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Import",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ImportCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Import",
                    |snapshot, _| replace_imported_model(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }

    fn update_import_link(
        &mut self,
        document: u64,
        feature: FeatureId,
        transform: impl FnOnce(&ImportedModelDefinition) -> ImportedModelDefinition,
    ) -> Result<(), String> {
        self.require_browser_document(document)?;
        let updated = {
            let snapshot = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            let authored = snapshot
                .work_geometry
                .get(feature)
                .map_err(|cause| cause.to_string())?;
            let PartFeatureDefinition::ImportedModel(definition) = authored.definition() else {
                return Err(format!("feature {feature:?}: expected an Imported Model"));
            };
            transform(definition)
        };
        oblikovati_automation::edit_part_with_checkpoints(
            &mut self.files,
            &mut self.part_scenes,
            document,
            "Update Import Link",
            |snapshot, _| replace_imported_model(snapshot, feature, updated),
        )
        .map_err(|cause| cause.to_string())?;
        Ok(())
    }
}

fn inspect_exchange_file(path: &str) -> Result<GuiImportSourceSummary, String> {
    let source = Path::new(path);
    let source_name = source
        .file_name()
        .and_then(|s| s.to_str())
        .unwrap_or("ImportedModel")
        .to_owned();
    let is_step = source
        .extension()
        .and_then(|ext| ext.to_str())
        .is_some_and(|ext| ext.eq_ignore_ascii_case("stp") || ext.eq_ignore_ascii_case("step"));
    if is_step {
        let info = inspect_step(source)?;
        Ok(GuiImportSourceSummary {
            source_path: path.to_owned(),
            source_name,
            file_kind: ImportedFileKind::Step,
            solid_names: info.solid_names,
            bounding_box: [0.0; 3],
            triangle_count: 0,
        })
    } else {
        let info = inspect_stl(source)?;
        Ok(GuiImportSourceSummary {
            source_path: path.to_owned(),
            source_name,
            file_kind: ImportedFileKind::Stl,
            solid_names: vec![info.solid_name],
            bounding_box: info.bounding_box,
            triangle_count: info.triangle_count,
        })
    }
}
