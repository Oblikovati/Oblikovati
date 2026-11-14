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
        face: Option<oblikovati_martensite::PrismFaceRole>,
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
        if let Some(role) = face {
            let key = body.face_key_for_role(history, guid, role)?;
            return body.face_bounds_center(history, guid, &key);
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
    pub(crate) fn visible_body_bounds<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
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
        let mut result = Vec::new();
        for body in self.current(session)?.ordered_bodies() {
            let key = body.body_reference_key(history, guid)?;
            if session.body_visible(&super::bodies::body_reference_text(key)) {
                result.push(body.body_bounds(history, guid, key)?);
            }
        }
        if result.is_empty() {
            return Err("selection fit: expected one or more visible solid bodies".into());
        }
        Ok(result)
    }
}
