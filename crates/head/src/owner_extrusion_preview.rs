// SPDX-License-Identifier: GPL-2.0-only
//! Build a temporary certified extrusion without changing the document.
use super::*;
use oblikovati_martensite::PrismPresentationSource;
pub(super) struct ComponentPreview {
    assembly: u64,
    generations: Vec<(u64, u64)>,
    scene: oblikovati_viewport::ViewportScene,
    owned_source: Option<u64>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum PreviewKind {
    Additive,
    Replacement,
    BlendInput,
}
pub(super) type ScenePreview = (u64, u64, oblikovati_viewport::ViewportScene, PreviewKind);
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_component_preview(
        &mut self,
        request: Option<(u64, u64)>,
    ) -> Result<bool, String> {
        let changed = self.clear_component_preview()?;
        self.part_scenes.invalidate_presentation();
        let Some((assembly, source)) = request else {
            return Ok(changed);
        };
        if self.files.workspace().active_document() != Some(assembly) || assembly == source {
            return Err(format!(
                "assembly {assembly} component {source}: expected an active assembly and a different component"
            ));
        }
        let root = self
            .files
            .workspace()
            .session(assembly)
            .map_err(|error| error.to_string())?;
        if root.snapshot().assembly.is_none() {
            return Err(format!("document {assembly}: expected an assembly"));
        }
        let generations = self.component_preview_generations();
        let identity = oblikovati_api::types::Matrix {
            cells: [
                1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
            ],
        };
        let scene = self
            .part_scenes
            .component_preview(&mut self.files, source, &identity)?;
        self.component_preview = Some(ComponentPreview {
            assembly,
            generations,
            scene,
            owned_source: None,
        });
        Ok(true)
    }
    pub(super) fn set_component_file_preview(
        &mut self,
        assembly: u64,
        path: &str,
    ) -> Result<bool, String> {
        self.clear_component_preview()?;
        self.part_scenes.invalidate_presentation();
        if self.files.workspace().active_document() != Some(assembly) {
            return Err(format!("assembly {assembly}: expected the active assembly"));
        }
        let (source, opened) = self
            .document_host()
            .open_component_preview(std::path::Path::new(path))
            .map_err(|error| error.to_string())?;
        if let Err(error) = self.set_component_preview(Some((assembly, source))) {
            if opened {
                self.files
                    .close_discarding_changes(source)
                    .map_err(|cleanup| {
                        format!(
                            "preview failed: {error}; component {source} cleanup failed: {cleanup}"
                        )
                    })?;
            }
            return Err(error);
        }
        if opened && let Some(preview) = &mut self.component_preview {
            preview.owned_source = Some(source);
        }
        Ok(true)
    }
    fn clear_component_preview(&mut self) -> Result<bool, String> {
        let Some(preview) = self.component_preview.take() else {
            return Ok(false);
        };
        if let Some(source) = preview.owned_source {
            let visible = self
                .files
                .document_info(source)
                .map_err(|error| error.to_string())?
                .visible;
            if !visible && self.files.has_reference_dependents(source) {
                self.files
                    .keep_referenced_background_model(source)
                    .map_err(|error| error.to_string())?;
            } else if !visible {
                self.files
                    .close_discarding_changes(source)
                    .map_err(|error| error.to_string())?;
            }
        }
        Ok(true)
    }
    fn component_preview_generations(&self) -> Vec<(u64, u64)> {
        self.files
            .workspace()
            .documents()
            .iter()
            .filter_map(|&id| {
                self.files
                    .workspace()
                    .session(id)
                    .ok()
                    .map(|session| (id, session.generation()))
            })
            .collect()
    }
    pub(super) fn set_extrusion_preview(
        &mut self,
        preview: &oblikovati_gui::GuiExtrudePreview,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let document = preview.document;
        let request = self.extrusion_request(
            document,
            preview.sketch,
            preview.profile,
            &preview.distance,
            preview.flipped,
        )?;
        let presentation = self.preview_extrusion_from_gui(document, &request)?;
        let scene = oblikovati_viewport::ViewportScene::from_prism(
            &presentation,
            [0., 0., 1.],
            [1., 0.65, 0.1, 1.],
        )?;
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        self.extrusion_preview = Some((document, generation, scene, PreviewKind::Additive));
        self.part_scenes.invalidate_presentation();
        Ok(true)
    }
    pub(super) fn clear_extrusion_preview(&mut self) -> bool {
        if self.extrusion_preview.take().is_none() {
            return false;
        }
        self.part_scenes.invalidate_presentation();
        true
    }
    pub(super) fn preview_edited_extrusion(
        &mut self,
        document: u64,
        feature: u64,
        distance: &str,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let scene = self.edited_extrusion_scene(document, feature, distance)?;
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        self.extrusion_preview = Some((document, generation, scene, PreviewKind::Replacement));
        self.part_scenes.invalidate_presentation();
        Ok(true)
    }
    pub(super) fn edited_extrusion_scene(
        &self,
        document: u64,
        feature: u64,
        distance: &str,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        self.require_browser_document(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let request = feature_distance_request(feature, distance);
        session
            .preview_projected(
                |candidate, _| {
                    oblikovati_automation::apply_feature_distance_edit(candidate, &request)
                },
                |candidate, ()| self.project_edited_scene(document, feature, candidate),
            )
            .map_err(|cause| cause.to_string())
    }
    fn project_edited_scene(
        &self,
        document: u64,
        feature: u64,
        candidate: &oblikovati_model::history::DocumentSnapshot,
    ) -> Result<oblikovati_viewport::ViewportScene, oblikovati_api::Error> {
        self.part_scenes
            .preview_feature_edit(
                &self.files,
                document,
                candidate,
                oblikovati_model::feature::FeatureId(feature),
            )
            .map_err(oblikovati_api::Error)
    }
    pub(super) fn feature_edit_is_current(
        &self,
        request: &oblikovati_gui::GuiExtrudeEditRequest,
    ) -> Result<bool, String> {
        let actual = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .generation();
        Ok(request.generation.is_none_or(|expected| expected == actual))
    }
    pub(super) fn commit_edited_extrusion(
        &mut self,
        request: &oblikovati_gui::GuiExtrudeEditRequest,
    ) -> Result<bool, String> {
        if !self.feature_edit_is_current(request)? {
            return Err(format!(
                "extrusion edit generation {:?}: expected the current generation",
                request.generation
            ));
        }
        self.edited_extrusion_scene(request.document, request.feature.0, &request.distance)?;
        let edit = feature_distance_request(request.feature.0, &request.distance);
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .edit(&edit)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }
    pub(super) fn take_scene_with_extrusion_preview(
        &mut self,
    ) -> Result<Option<crate::HeadSceneUpdate>, String> {
        let current = self
            .files
            .workspace()
            .active_document()
            .and_then(|document| {
                self.files
                    .workspace()
                    .session(document)
                    .ok()
                    .map(|session| (document, session.generation()))
            });
        if self
            .extrusion_preview
            .as_ref()
            .is_some_and(|(document, generation, _, _)| Some((*document, *generation)) != current)
        {
            self.clear_extrusion_preview();
        }
        let stale = self.component_preview.as_ref().is_some_and(|preview| {
            current.map(|(document, _)| document) != Some(preview.assembly)
                || preview.generations != self.component_preview_generations()
        });
        if stale {
            self.set_component_preview(None)?;
        }
        let selected = self
            .selected_occurrence
            .as_ref()
            .and_then(|(document, path)| {
                current
                    .filter(|(active, _)| active == document)
                    .map(|_| path.clone())
            });
        self.part_scenes.set_occurrence_selection(selected);
        let selected_faces = current.and_then(|(document, generation)| {
            let faces: Vec<_> = self
                .current_face_appearance_picks(document)
                .iter()
                .map(|pick| (pick.face.feature, pick.face.key.clone()))
                .collect();
            if faces.is_empty() || self.active_sketch_edit().is_some() {
                return None;
            }
            let color = &self.color_schemes.active().secondary_select;
            Some((document, generation, faces, [color.r, color.g, color.b]))
        });
        self.part_scenes.set_selected_faces(selected_faces);
        let Some(mut update) = self.part_scenes.refresh(&mut self.files)? else {
            return Ok(None);
        };
        if let Some((_, _, preview, kind)) = &self.extrusion_preview {
            if *kind != PreviewKind::Additive {
                update.scene = preview.clone();
            } else {
                update
                    .scene
                    .triangles
                    .extend(preview.triangles.iter().copied());
            }
        }
        if let Some(preview) = &self.component_preview {
            update
                .scene
                .triangles
                .extend(preview.scene.triangles.iter().copied());
        }
        Ok(Some(update))
    }
    pub(super) fn preview_extrusion_from_gui(
        &self,
        document: u64,
        request: &oblikovati_api::wire::featureargs::Extrude,
    ) -> Result<oblikovati_martensite::PrismPresentation, String> {
        let backend = self.extrusion_access.as_ref().ok_or_else(|| {
            "extrusion preview: expected a certified geometry service".to_string()
        })?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let prepared =
            oblikovati_automation::prepare_extrusion(session.snapshot(), backend, request)
                .map_err(|cause| cause.to_string())?;
        session
            .preview_projected(
                |snapshot, ids| prepared.create(snapshot, ids),
                |candidate, id| {
                    let (presentation, reused) = self
                        .files
                        .workspace()
                        .inspect_owned_append(document, Some(backend.0), candidate, id, |body| {
                            extrusion_preview_presentation(body, id)
                        })
                        .map_err(|error| oblikovati_api::Error(error.to_string()))?;
                    #[cfg(test)]
                    if reused {
                        self.part_scenes.note_extrusion_prefix_preview();
                    }
                    #[cfg(not(test))]
                    let _ = reused;
                    presentation
                },
            )
            .map_err(|cause| cause.to_string())
    }
}
fn extrusion_preview_presentation(
    body: &oblikovati_martensite::NamedSolid,
    id: oblikovati_model::feature::FeatureId,
) -> Result<oblikovati_martensite::PrismPresentation, oblikovati_api::Error> {
    body.as_prism()
        .ok_or_else(|| {
            oblikovati_api::Error(format!("preview feature {id:?}: expected a prism result"))
        })?
        .presentation(crate::part_scene::DISPLAY_MESH)
        .map_err(oblikovati_api::Error)
}
pub(super) fn feature_distance_request(
    feature: u64,
    distance: &str,
) -> oblikovati_api::wire::EditFeatureArgs {
    oblikovati_api::wire::EditFeatureArgs {
        id: feature,
        scalars: Some(vec![oblikovati_api::wire::ScalarEdit {
            index: 0,
            value: distance.into(),
        }]),
        ..Default::default()
    }
}
