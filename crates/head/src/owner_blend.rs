// SPDX-License-Identifier: GPL-2.0-only
//! Preview and commit blends from exact desktop edge selections (#4045, #4046).
use super::*;
use oblikovati_automation::{PreparedChamfer, PreparedFillet, prepare_chamfer, prepare_fillet};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn require_blend_selection(
        &self,
        request: &oblikovati_gui::GuiFilletRequest,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let edges: Vec<_> = self
            .chosen_model_edges(request.document)
            .iter()
            .map(|edge| edge.key.clone())
            .collect();
        if session.generation() != request.generation || edges != request.edges {
            return Err(format!(
                "blend selection generation {}, edges {}: expected the current document selection",
                request.generation,
                request.edges.len()
            ));
        }
        Ok(())
    }
    fn selected_fillet(&self, document: u64, radius: &str) -> Result<PreparedFillet, String> {
        let (body, edges) = self.selected_blend_inputs(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        prepare_fillet(session.snapshot(), body, edges, radius).map_err(|cause| cause.to_string())
    }

    fn selected_chamfer(&self, document: u64, distance: &str) -> Result<PreparedChamfer, String> {
        let (body, edges) = self.selected_blend_inputs(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        prepare_chamfer(session.snapshot(), body, edges, distance)
            .map_err(|cause| cause.to_string())
    }

    fn selected_blend_inputs(&self, document: u64) -> Result<(Vec<u8>, Vec<Vec<u8>>), String> {
        self.require_browser_document(document)?;
        let selected = self.chosen_model_edges(document);
        let first = selected
            .first()
            .ok_or("blend edges: expected at least one selected edge")?;
        if selected.iter().any(|edge| edge.feature != first.feature) {
            return Err("blend edges: expected edges from one body".into());
        }
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let body = self
            .part_scenes
            .current(session)?
            .body(first.feature)
            .ok_or_else(|| format!("blend feature {:?}: expected a current body", first.feature))?;
        let key = body.body_reference_key(session.history(), guid)?.to_vec();
        Ok((key, selected.iter().map(|edge| edge.key.clone()).collect()))
    }

    pub(super) fn preview_selected_fillet(
        &mut self,
        document: u64,
        radius: &str,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let prepared = self.selected_fillet(document, radius)?;
        self.preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))
    }

    pub(super) fn preview_selected_chamfer(
        &mut self,
        document: u64,
        distance: &str,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let prepared = self.selected_chamfer(document, distance)?;
        self.preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))
    }

    fn preview_selected_blend(
        &mut self,
        document: u64,
        create: impl FnOnce(
            &mut oblikovati_model::history::DocumentSnapshot,
            &mut oblikovati_model::history::RecipeReservations<'_>,
        )
            -> Result<oblikovati_model::feature::FeatureId, oblikovati_api::Error>,
    ) -> Result<bool, String> {
        self.clear_extrusion_preview();
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let scene = session
            .preview_projected(create, |candidate, feature| {
                self.part_scenes
                    .preview_feature_edit(&self.files, document, candidate, feature)
                    .map_err(oblikovati_api::Error)
            })
            .map_err(|cause| cause.to_string())?;
        self.extrusion_preview = Some((
            document,
            generation,
            scene,
            super::owner_extrusion_preview::PreviewKind::BlendInput,
        ));
        self.part_scenes.invalidate_presentation();
        Ok(true)
    }

    pub(super) fn commit_selected_fillet(
        &mut self,
        document: u64,
        radius: &str,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let edges = self
            .chosen_model_edges(document)
            .iter()
            .map(|edge| edge.key.iter().map(|byte| format!("{byte:02x}")).collect())
            .collect();
        let request = oblikovati_api::wire::featureargs::Fillet {
            edge_refs: Some(edges),
            radius: radius.into(),
            ..Default::default()
        };
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .add_feature(&request)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }

    pub(super) fn commit_selected_chamfer(
        &mut self,
        document: u64,
        distance: &str,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let edges = self
            .chosen_model_edges(document)
            .iter()
            .map(|edge| edge.key.iter().map(|byte| format!("{byte:02x}")).collect())
            .collect();
        let request = oblikovati_api::wire::featureargs::Chamfer {
            edge_refs: Some(edges),
            distance: distance.into(),
            ..Default::default()
        };
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .add_feature(&request)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }

    fn require_blend_edit(
        &self,
        request: &oblikovati_gui::GuiFilletEditRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err(format!(
                "blend edit generation {}: expected current generation {}",
                request.generation,
                session.generation()
            ));
        }
        Ok(())
    }

    pub(super) fn preview_edited_fillet(
        &mut self,
        request: &oblikovati_gui::GuiFilletEditRequest,
    ) -> Result<bool, String> {
        self.preview_edited_blend(request, oblikovati_automation::apply_fillet_radius_edit)
    }

    pub(super) fn preview_edited_chamfer(
        &mut self,
        request: &oblikovati_gui::GuiFilletEditRequest,
    ) -> Result<bool, String> {
        self.preview_edited_blend(request, oblikovati_automation::apply_chamfer_distance_edit)
    }

    fn preview_edited_blend(
        &mut self,
        request: &oblikovati_gui::GuiFilletEditRequest,
        apply: fn(
            &mut oblikovati_model::history::DocumentSnapshot,
            &oblikovati_api::wire::EditFeatureArgs,
        ) -> Result<oblikovati_model::feature::FeatureId, oblikovati_api::Error>,
    ) -> Result<bool, String> {
        self.require_blend_edit(request)?;
        self.clear_extrusion_preview();
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let edit = super::owner_extrusion_preview::feature_distance_request(
            request.feature.0,
            &request.radius,
        );
        let scene = session
            .preview_projected(
                |candidate, _| apply(candidate, &edit),
                |candidate, feature| {
                    self.part_scenes
                        .preview_feature_edit(&self.files, request.document, candidate, feature)
                        .map_err(oblikovati_api::Error)
                },
            )
            .map_err(|cause| cause.to_string())?;
        self.extrusion_preview = Some((
            request.document,
            request.generation,
            scene,
            super::owner_extrusion_preview::PreviewKind::Replacement,
        ));
        self.part_scenes.invalidate_presentation();
        Ok(true)
    }

    pub(super) fn commit_edited_fillet(
        &mut self,
        request: &oblikovati_gui::GuiFilletEditRequest,
    ) -> Result<bool, String> {
        self.require_blend_edit(request)?;
        let edit = super::owner_extrusion_preview::feature_distance_request(
            request.feature.0,
            &request.radius,
        );
        oblikovati_api::client::Client::new(self.document_host())
            .features()
            .edit(&edit)
            .map_err(|cause| cause.to_string())?;
        self.clear_extrusion_preview();
        Ok(true)
    }
}
