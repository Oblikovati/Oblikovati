// SPDX-License-Identifier: GPL-2.0-only
//! Prepare and commit a generation-checked part-view display switch (#4118).
use crate::{DocumentSession, TransactionObserver};
use oblikovati_api::{
    contract::PartDesignView as _,
    types::{PartDesignViewCamera, ViewOrientationTypeEnum},
};
use std::collections::BTreeSet;

/// A checked switch, before the desktop applies its target camera.
/// Example: `let camera = prepared.camera()` before `commit_part_view_activation(prepared)`.
pub struct PreparedPartViewActivation {
    scope: oblikovati_model::history::EvaluationScope,
    visibility_revision: u64,
    source: u64,
    target: u64,
    current_camera: PartDesignViewCamera,
    target_camera: Option<PartDesignViewCamera>,
    orientation: ViewOrientationTypeEnum,
    target_hidden: BTreeSet<String>,
    available: BTreeSet<String>,
}
impl PreparedPartViewActivation {
    /// Read the target's saved camera; example: `prepared.camera()` before changing the viewport.
    pub fn camera(&self) -> Option<&PartDesignViewCamera> {
        self.target_camera.as_ref()
    }
    /// Read the fallback standard orientation; example: `prepared.orientation()` if there is no camera.
    pub fn orientation(&self) -> ViewOrientationTypeEnum {
        self.orientation
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Check a switch against current resolved bodies without changing the document.
    /// Example: `session.prepare_part_view_activation(6, camera, &resolved_keys)?`.
    pub fn prepare_part_view_activation(
        &self,
        target: u64,
        current_camera: PartDesignViewCamera,
        available: &BTreeSet<String>,
    ) -> Result<PreparedPartViewActivation, String> {
        if self.group.is_some() {
            return Err(
                "part view activation with open transaction: expected no open group".into(),
            );
        }
        oblikovati_model::part_design_view::validate_part_camera(&current_camera)?;
        let views = self
            .snapshot()
            .part_design_views
            .as_ref()
            .ok_or("part view activation: expected part views")?;
        let source = views
            .active()
            .ok_or("part view activation: expected one active view")?
            .id();
        let chosen = views.get(target)?;
        let target_hidden = chosen.hidden_body_keys().clone();
        self.body_visibility
            .prepare_hidden(&target_hidden, available)?;
        let source_hidden = self.hidden_body_keys().clone();
        if let Some(key) = source_hidden.iter().find(|key| !available.contains(*key)) {
            return Err(format!(
                "active body visibility key {key:?}: expected a current resolved body"
            ));
        }
        Ok(PreparedPartViewActivation {
            scope: self.history.evaluation_scope(),
            visibility_revision: self.body_visibility_revision(),
            source,
            target,
            current_camera,
            target_camera: chosen.camera().cloned(),
            orientation: chosen.standard_orientation(),
            target_hidden,
            available: available.clone(),
        })
    }

    /// Commit a prepared switch after the caller applies its camera; restore the camera if this fails.
    /// Example: `session.commit_part_view_activation(prepared, applied_camera)?` after changing the viewport.
    pub fn commit_part_view_activation(
        &mut self,
        prepared: PreparedPartViewActivation,
        applied_camera: PartDesignViewCamera,
    ) -> Result<(), String> {
        self.commit_part_view_activation_projected(prepared, applied_camera, |_, _| Ok(()))?;
        Ok(())
    }

    /// Capture a validated candidate before accepting a display transition.
    /// Example: `session.commit_part_view_activation_projected(prepared, camera, capture)?`.
    pub fn commit_part_view_activation_projected<T>(
        &mut self,
        prepared: PreparedPartViewActivation,
        applied_camera: PartDesignViewCamera,
        project: impl FnOnce(
            &oblikovati_model::history::DocumentSnapshot,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, String>,
    ) -> Result<(T, oblikovati_model::history::EvaluationTransition), String> {
        if self.group.is_some()
            || !prepared.scope.is_current(&self.history)
            || self.body_visibility_revision() != prepared.visibility_revision
        {
            return Err(format!(
                "part view switch generation {} visibility {}: expected unchanged document and display",
                prepared.scope.generation(),
                prepared.visibility_revision
            ));
        }
        oblikovati_model::part_design_view::validate_part_camera(&applied_camera)?;
        if prepared
            .target_camera
            .as_ref()
            .is_some_and(|saved| saved != &applied_camera)
        {
            return Err(format!(
                "applied camera {applied_camera:?}: expected the target's stored camera"
            ));
        }
        let display = oblikovati_model::history::PartViewDisplayTransition {
            before: oblikovati_model::history::PartViewDisplayState {
                hidden_body_keys: self.hidden_body_keys().clone(),
                camera: Some(prepared.current_camera.clone()),
            },
            after: oblikovati_model::history::PartViewDisplayState {
                hidden_body_keys: prepared.target_hidden.clone(),
                camera: Some(applied_camera),
            },
        };
        let old_visibility = self.body_visibility.clone();
        self.body_visibility
            .replace_hidden(&prepared.target_hidden, &prepared.available)?;
        let result = self.run_part_view_edit_projected(
            "Activate Part View",
            display,
            |snapshot, _| apply_prepared_view(snapshot, &prepared),
            project,
        );
        if result.is_err() {
            self.body_visibility = old_visibility;
        }
        result
    }
}
fn apply_prepared_view(
    snapshot: &mut oblikovati_model::history::DocumentSnapshot,
    prepared: &PreparedPartViewActivation,
) -> Result<(), String> {
    let views = snapshot
        .part_design_views
        .as_mut()
        .ok_or("part view activation: expected retained part views")?;
    if views.active().map(|view| view.id()) != Some(prepared.source) {
        return Err(format!(
            "active view {:?}: expected ID {}",
            views.active().map(|view| view.id()),
            prepared.source
        ));
    }
    if prepared.source != prepared.target
        && !views.get(prepared.source)?.locked()
        && views.get(prepared.source)?.autosave_camera()
    {
        views.capture_camera(prepared.source, prepared.current_camera.clone())?;
    }
    views.activate(prepared.target)
}

#[cfg(test)]
#[path = "part_view_activation_tests.rs"]
mod tests;
