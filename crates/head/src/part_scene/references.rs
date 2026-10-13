// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{AssetEntityKind, AssetReferenceAccess};
use oblikovati_model::{feature::FeatureId, identity::ReferenceEnvelope};
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> AssetReferenceAccess<O, W, F>
    for DesktopPartScenes
{
    fn verify(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        key: &[u8],
        kind: AssetEntityKind,
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
            AssetEntityKind::Body => body.body_report(session.history(), guid, key).map(|_| ()),
            AssetEntityKind::Face => body.face_report(session.history(), guid, key).map(|_| ()),
        }
    }
}
