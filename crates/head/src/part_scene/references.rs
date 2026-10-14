// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{PartTopologyAccess, TopologyEntityKind};
use oblikovati_model::{feature::FeatureId, identity::ReferenceEnvelope};
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
        self.current_reference_keys(files, document)
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
    fn model_health(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_api::wire::ModelHealthResult, String> {
        self.ensure_current(files, document)?;
        self.current_model_health(files, document)
    }
    fn requires_update(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String> {
        self.part_requires_update(files, document)
    }
    fn update(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        rebuild: bool,
    ) -> Result<Vec<oblikovati_api::wire::FeatureError>, String> {
        self.update_part(files, document, rebuild)
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
