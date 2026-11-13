// SPDX-License-Identifier: GPL-2.0-only
//! Build a temporary certified extrusion without changing the document.
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_extrusion_preview(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        profile: i64,
        distance: &str,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let sketch_index = self.extrusion_sketch_index(document, sketch)?;
        let request = oblikovati_api::wire::featureargs::Extrude {
            sketch_index,
            profile_index: profile,
            distance: distance.into(),
            ..Default::default()
        };
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
        self.extrusion_preview = Some((document, generation, scene));
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
            .is_some_and(|(document, generation, _)| Some((*document, *generation)) != current)
        {
            self.clear_extrusion_preview();
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
        let Some(mut update) = self.part_scenes.refresh(&mut self.files)? else {
            return Ok(None);
        };
        if let Some((_, _, preview)) = &self.extrusion_preview {
            update
                .scene
                .triangles
                .extend(preview.triangles.iter().copied());
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
                    let mut evaluator =
                        oblikovati_martensite::CertifiedPrismEvaluator::with_policy(backend.0);
                    let result = candidate
                        .evaluate_part_features(&mut evaluator)
                        .map_err(|cause| oblikovati_api::Error(cause.to_string()))?;
                    let body = result.body(id).ok_or_else(|| {
                        oblikovati_api::Error(format!(
                            "preview feature {id:?}: expected a certified body; health {:?}",
                            result.health()
                        ))
                    })?;
                    body.body
                        .presentation(crate::part_scene::DISPLAY_MESH)
                        .map_err(oblikovati_api::Error)
                },
            )
            .map_err(|cause| cause.to_string())
    }
}
