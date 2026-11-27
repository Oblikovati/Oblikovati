// SPDX-License-Identifier: GPL-2.0-only
//! Serializable UI snapshot extraction from the active `WebSession`.

use super::WebSession;
use crate::{
    WebCameraSnapshot, WebDocumentSummary, WebFeatureSummary, WebParameterSummary, WebStateSnapshot,
};
use oblikovati_model::quantity::Dimension;
use oblikovati_viewport::web::web_worker_pool_size;

impl WebSession {
    /// Build a serializable UI state snapshot for the browser shell.
    /// Example: `let state = session.snapshot();`.
    pub fn snapshot(&self) -> WebStateSnapshot {
        let ws = self.workspace.workspace();
        let active = ws.active_document();
        let documents = ws
            .documents()
            .iter()
            .map(|&id| WebDocumentSummary {
                id,
                name: ws.display_name(id).unwrap_or("Document").to_owned(),
                active: active == Some(id),
                dirty: ws.needs_save(id).unwrap_or(false),
            })
            .collect();
        let (features, parameters, can_undo, can_redo) = active
            .and_then(|id| ws.session(id).ok())
            .map(|session| {
                let snap = session.snapshot();
                let features = snap
                    .work_geometry
                    .features()
                    .iter()
                    .map(|feat| WebFeatureSummary {
                        id: feat.id().0,
                        name: feat.name().to_owned(),
                        kind: "Feature".into(),
                        status: if feat.suppressed() {
                            "Suppressed".into()
                        } else {
                            "Active".into()
                        },
                    })
                    .collect();
                let unit = snap
                    .parameters
                    .units()
                    .preferred_name(Dimension::Length)
                    .to_owned();
                let parameters = snap
                    .parameters
                    .ids()
                    .iter()
                    .filter_map(|&pid| {
                        let state = snap.parameters.get(pid)?;
                        let value = state.current_value().map(|q| q.value);
                        Some(WebParameterSummary {
                            id: pid,
                            name: state.name.clone(),
                            expression: state.source.clone(),
                            value,
                            unit: unit.clone(),
                        })
                    })
                    .collect();
                (
                    features,
                    parameters,
                    session.history().committed_undo_label().is_some(),
                    session.history().committed_redo_label().is_some(),
                )
            })
            .unwrap_or_default();
        let camera = self.viewport.camera_state().unwrap_or_default();
        let validation_errors = self
            .runtime
            .validation_errors()
            .map(|v| v.len())
            .unwrap_or(0);
        WebStateSnapshot {
            active_document: active,
            documents,
            features,
            parameters,
            visible_bodies: self.visible_bodies,
            triangle_count: self
                .current_scene
                .as_ref()
                .map_or(0, |scene| scene.triangles.len()),
            can_undo,
            can_redo,
            theme: self.theme,
            perspective: camera.perspective,
            normal_debug: self.normal_debug,
            camera: WebCameraSnapshot::from(camera),
            device_name: self.runtime.device_name(),
            active_workers: web_worker_pool_size(),
            validation_errors,
            status_message: self.status_message.clone(),
        }
    }

    /// Serialize the current UI state snapshot as JSON.
    /// Example: `let json = session.snapshot_json()?;`.
    pub fn snapshot_json(&self) -> Result<String, String> {
        serde_json::to_string(&self.snapshot()).map_err(|err| err.to_string())
    }
}
