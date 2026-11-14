// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{PartTopologyAccess, PartUpdateAccess, TopologyEntityKind};
use oblikovati_model::{feature::FeatureId, identity::ReferenceEnvelope};
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> PartUpdateAccess<O, W, F>
    for DesktopPartScenes
{
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
        oblikovati_automation::resolve_slot_face(files, document, &self.slot, face)
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
        self.slot
            .current_datums(session)
            .map_err(|error| error.to_string())
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> PartTopologyAccess<O, W, F>
    for DesktopPartScenes
{
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
        Ok(body.face_report(session.history(), guid, key)?.area)
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
        let body = self.current(session)?.body(feature).ok_or_else(|| {
            format!("assignment feature {feature:?}: expected a current successful body")
        })?;
        match kind {
            TopologyEntityKind::Body => body.body_report(session.history(), guid, key).map(|_| ()),
            TopologyEntityKind::Face => body.face_report(session.history(), guid, key).map(|_| ()),
        }
    }
}
