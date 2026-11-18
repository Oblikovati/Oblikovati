// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{PartTopologyAccess, PartUpdateAccess, TopologyEntityKind};
use oblikovati_model::{feature::FeatureId, identity::ReferenceEnvelope};
impl DesktopPartScenes {
    /// Resolve a selected face to its current identity before it becomes a datum parent.
    pub(crate) fn selected_face_reference<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        feature: FeatureId,
        face: &[u8],
    ) -> Result<String, String> {
        let key = self.selected_face_key(files, document, feature, face)?;
        Ok(super::bodies::body_reference_text(&key))
    }
    fn selected_face_key<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        feature: FeatureId,
        face: &[u8],
    ) -> Result<Vec<u8>, String> {
        self.ensure_current(files, document)?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|error| error.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let body = self
            .current(session)?
            .body(feature)
            .ok_or_else(|| format!("selected feature {feature:?}: expected a current body"))?;
        body.face_topology(session.history(), guid, face)?;
        Ok(face.to_vec())
    }
    pub(crate) fn sketch_support_face<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<Option<oblikovati_model::feature::work_plane::WorkPlaneFaceReference>, String> {
        self.ensure_current(files, document)?;
        let Some((feature, face)) = self.face_on_line(files, document, line)? else {
            return Ok(None);
        };
        let key = self.selected_face_key(files, document, feature, &face)?;
        oblikovati_model::feature::work_plane::WorkPlaneFaceReference::new(key).map(Some)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> PartUpdateAccess<O, W, F>
    for DesktopPartScenes
{
    fn resolution_policy(&self) -> Option<oblikovati_martensite::PrismResolutionPolicy> {
        self.resolution
    }
    fn ensure_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        self.ensure_current(files, document)
    }
    fn current_part<'a>(
        &'a self,
        files: &'a FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        &'a oblikovati_model::feature::part::PartFeatureEvaluation<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
        String,
    > {
        self.replay.current_part(files, document)
    }
    fn blend_body(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        self.replay.blend_body(files, document, edges)
    }
    fn update_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<oblikovati_api::wire::FeatureError>, String> {
        self.recompute_for_api(files, document)
    }
    fn part_requires_update(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String> {
        self.requires_update_for_api(files, document)
    }
    fn resolve_planar_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        self.ensure_current(files, document)?;
        oblikovati_automation::resolve_slot_face(files, document, self.active_slot()?, face)
    }
    fn datum_evaluation(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        self.ensure_current(files, document)?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        Ok(self.current(session)?.datum_evaluation())
    }
    fn preview_datums(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        candidate: &oblikovati_model::history::DocumentSnapshot,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        oblikovati_automation::preview_part_datums(files, document, candidate, self.resolution)
    }
    fn current_part_datums(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String> {
        self.ensure_current(files, document)?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        self.active_slot()?
            .current_datums(session)
            .map_err(|error| error.to_string())
    }
    fn current_part_datums_for_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String> {
        self.ensure_current(files, document)?;
        oblikovati_automation::slot_datums_for_face(files, document, self.active_slot()?, face)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> PartTopologyAccess<O, W, F>
    for DesktopPartScenes
{
    fn assembly_view_producer(&mut self) -> Result<String, String> {
        self.assembly_producer_digest()
    }
    fn assembly_view(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        (
            Vec<oblikovati_persistence::package::SavedAssemblyTriangle>,
            f64,
            f64,
        ),
        String,
    > {
        let update =
            self.evaluate_assembly_for_view(files, document, AssemblyViewMode::Persisted)?;
        if update.status.unavailable_count() != 0 {
            return Err(format!(
                "assembly {document}: expected complete presentation: {:?}",
                update.status
            ));
        }
        let triangles = update
            .scene
            .triangles
            .into_iter()
            .map(
                |triangle| oblikovati_persistence::package::SavedAssemblyTriangle {
                    positions: triangle.positions,
                    color: triangle.color,
                },
            )
            .collect();
        Ok((
            triangles,
            DISPLAY_MESH.chord_error,
            DISPLAY_MESH.angular_error,
        ))
    }
    fn list_bodies(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<oblikovati_api::wire::BodyInfo>, String> {
        self.ensure_current(files, document)?;
        self.body_infos(files, document)
    }
    fn reference_keys(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_api::wire::ReferenceKeysResult, String> {
        self.ensure_current(files, document)?;
        self.topology_keys(files, document)
    }
    fn physical_properties(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: Option<i64>,
        density: f64,
    ) -> Result<oblikovati_api::wire::MassPropertiesResult, String> {
        self.ensure_current(files, document)?;
        self.body_physical_properties(files, document, index, density)
    }
    fn face_area(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: i64,
        key: &[u8],
    ) -> Result<f64, String> {
        self.ensure_current(files, document)?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.current(session)?;
        let bodies: Vec<_> = evaluation.ordered_bodies().collect();
        let body = super::physical::selected_body(&bodies, index)?;
        body.face_area(session.history(), guid, key)
    }
    fn edge_length(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: i64,
        key: &[u8],
    ) -> Result<f64, String> {
        self.ensure_current(files, document)?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.current(session)?;
        let bodies: Vec<_> = evaluation.ordered_bodies().collect();
        let body = super::physical::selected_body(&bodies, index)?;
        body.edge_length(session.history(), guid, key)
    }
    fn model_health(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_api::wire::ModelHealthResult, String> {
        self.ensure_current(files, document)?;
        self.current_model_health(files, document)
    }
    fn verify(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        key: &[u8],
        kind: TopologyEntityKind,
    ) -> Result<(), String> {
        let envelope = ReferenceEnvelope::decode(key).map_err(|cause| cause.to_string())?;
        self.ensure_current(files, document)?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let feature = FeatureId(envelope.context);
        let body = self.current(session)?.body_from_source(feature)?;
        match kind {
            TopologyEntityKind::Body => body
                .topology_summary(session.history(), guid, key)
                .map(|_| ()),
            TopologyEntityKind::Face => body.face_area(session.history(), guid, key).map(|_| ()),
        }
    }
}
