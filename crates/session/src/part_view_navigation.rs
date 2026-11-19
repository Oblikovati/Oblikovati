// SPDX-License-Identifier: GPL-2.0-only
//! Prepare tagged display history against the current body and document generation (#4118).
use crate::{DocumentSession, TransactionObserver};
use oblikovati_api::types::PartDesignViewCamera;
use oblikovati_model::history::{EvaluationScope, PartViewDisplayState};
use std::collections::BTreeSet;

/// The direction of one tagged view history step; example: `PartViewNavigationDirection::Undo`.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PartViewNavigationDirection {
    Undo,
    Redo,
}
/// A validated history display, before the viewport applies its camera.
/// Example: `prepared.camera()` before committing the navigation.
pub struct PreparedPartViewNavigation {
    scope: EvaluationScope,
    visibility_revision: u64,
    direction: PartViewNavigationDirection,
    target: PartViewDisplayState,
    available: BTreeSet<String>,
}
impl PreparedPartViewNavigation {
    /// Read the camera to apply, if the step changed it; example: `prepared.camera()`.
    pub fn camera(&self) -> Option<&PartDesignViewCamera> {
        self.target.camera.as_ref()
    }
}
impl<O: TransactionObserver> DocumentSession<O> {
    /// Prepare a tagged view history step with freshly resolved bodies.
    /// Example: `session.prepare_part_view_navigation(PartViewNavigationDirection::Undo, &keys)?`.
    pub fn prepare_part_view_navigation(
        &self,
        direction: PartViewNavigationDirection,
        available: &BTreeSet<String>,
    ) -> Result<PreparedPartViewNavigation, String> {
        let transition = match direction {
            PartViewNavigationDirection::Undo => {
                self.history
                    .undo_label()
                    .map_err(|cause| cause.to_string())?;
                self.history.undo_part_view_display()
            }
            PartViewNavigationDirection::Redo => {
                self.history
                    .redo_label()
                    .map_err(|cause| cause.to_string())?;
                self.history.redo_part_view_display()
            }
        }
        .ok_or("part view history: expected a tagged display step")?;
        let target = match direction {
            PartViewNavigationDirection::Undo => &transition.before,
            PartViewNavigationDirection::Redo => &transition.after,
        };
        if let Some(key) = self
            .hidden_body_keys()
            .iter()
            .find(|key| !available.contains(*key))
        {
            return Err(format!(
                "current body visibility key {key:?}: expected a current resolved body"
            ));
        }
        self.body_visibility
            .prepare_hidden(&target.hidden_body_keys, available)?;
        if let Some(camera) = &target.camera {
            oblikovati_model::part_design_view::validate_part_camera(camera)?;
        }
        Ok(PreparedPartViewNavigation {
            scope: self.history.evaluation_scope(),
            visibility_revision: self.body_visibility_revision(),
            direction,
            target: target.clone(),
            available: available.clone(),
        })
    }

    /// Commit a prepared history step after its optional camera is applied; restore that camera if this fails.
    /// Example: `session.commit_part_view_navigation(prepared, applied_camera)?`.
    pub fn commit_part_view_navigation(
        &mut self,
        prepared: PreparedPartViewNavigation,
        applied_camera: Option<&PartDesignViewCamera>,
    ) -> Result<(), String> {
        self.commit_part_view_navigation_projected(prepared, applied_camera, |_, _| Ok(()))?;
        Ok(())
    }

    /// Capture the target recipe before a validated display history step.
    /// Example: `session.commit_part_view_navigation_projected(prepared, camera, capture)?`.
    pub fn commit_part_view_navigation_projected<T>(
        &mut self,
        prepared: PreparedPartViewNavigation,
        applied_camera: Option<&PartDesignViewCamera>,
        project: impl FnOnce(
            &oblikovati_model::history::DocumentSnapshot,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, String>,
    ) -> Result<(T, oblikovati_model::history::EvaluationTransition), String> {
        if !prepared.scope.is_current(&self.history)
            || self.body_visibility_revision() != prepared.visibility_revision
        {
            return Err(
                "part view history: expected the same document and display generation".into(),
            );
        }
        if prepared.target.camera.as_ref() != applied_camera {
            return Err(format!(
                "applied history camera {applied_camera:?}: expected {:?}",
                prepared.target.camera
            ));
        }
        let candidate = match prepared.direction {
            PartViewNavigationDirection::Undo => self.history.undo_candidate(),
            PartViewNavigationDirection::Redo => self.history.redo_candidate(),
        }
        .map_err(|cause| cause.to_string())?;
        let captured = project(candidate, &self.history)?;
        let old = self.body_visibility.clone();
        self.body_visibility
            .replace_hidden(&prepared.target.hidden_body_keys, &prepared.available)?;
        let result = match prepared.direction {
            PartViewNavigationDirection::Undo => self.undo_display_transition(),
            PartViewNavigationDirection::Redo => self.redo_display_transition(),
        };
        match result {
            Ok(transition) => Ok((captured, transition)),
            Err(cause) => {
                self.body_visibility = old;
                Err(cause.to_string())
            }
        }
    }
}
