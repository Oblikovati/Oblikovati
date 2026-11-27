// SPDX-License-Identifier: GPL-2.0-only
//! Resolve the current selected entity before asking the kernel adapter for exact bounds.
use super::*;
use oblikovati_model::feature::FeatureId;

impl DesktopPartScenes {
    pub(crate) fn selection_center<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        feature: FeatureId,
        face: Option<&[u8]>,
        expected_body: Option<&str>,
    ) -> Result<[f64; 3], String> {
        self.ensure_current(files, document)?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let history = session.history();
        let body = self
            .current(session)?
            .body(feature)
            .ok_or_else(|| format!("selected feature {feature:?}: expected a current body"))?;
        if let Some(key) = face {
            return body.face_bounds_center(history, guid, key);
        }
        let key = body.body_reference_key(history, guid)?;
        let actual = super::bodies::body_reference_text(key);
        if expected_body != Some(actual.as_str()) {
            return Err(format!(
                "selected body {expected_body:?}: expected current name {actual:?}"
            ));
        }
        body.body_bounds_center(history, guid, key)
    }
}
