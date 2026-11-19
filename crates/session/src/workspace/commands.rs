// SPDX-License-Identifier: GPL-2.0-only

use crate::{CommandError, DocumentSession, SessionError, TransactionObserver};
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};

/// A borrowed command capability. It cannot replace a workspace's document owner.
pub struct SessionCommands<'a, O> {
    pub(super) session: &'a mut DocumentSession<O>,
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Create a sketch with current part datums and project its reply.
    /// Example: `commands.create_sketch_with_part_datums(&request, &datums, project)?`.
    pub fn create_sketch_with_part_datums<R, E>(
        &mut self,
        request: &oblikovati_model::history::DocumentSketchCreation,
        datums: &crate::part_evaluation::CurrentPartDatums,
        project: impl FnOnce(crate::SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, crate::PartSketchCreateError<E>> {
        self.session
            .create_automatic_sketch_with_part_datums_projected(request, datums, project)
    }
    /// Create a named sketch using current part datums.
    /// Example: `commands.create_named_sketch_with_part_datums(&request, &datums, project)?`.
    pub fn create_named_sketch_with_part_datums<R, E>(
        &mut self,
        request: &oblikovati_model::history::DocumentSketchCreation,
        datums: &crate::part_evaluation::CurrentPartDatums,
        project: impl FnOnce(crate::SketchCreationProjectionInput) -> Result<R, E>,
    ) -> Result<R, crate::PartSketchCreateError<E>> {
        self.session
            .create_sketch_with_part_datums_projected(request, datums, project)
    }
    /// Mark the latest desktop sketch creation as entering edit mode; example: `commands.tag_sketch_creation(id)?`.
    pub fn tag_sketch_creation(
        &mut self,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<(), SessionError> {
        self.session.tag_sketch_creation(sketch)
    }
    /// Record a reversible editor finish; example: `commands.finish_sketch_history(id)?`.
    pub fn finish_sketch_history(
        &mut self,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<(), SessionError> {
        self.session.finish_sketch_history(sketch)
    }
    /// Finish an editor using generation-checked part datums.
    /// Example: `commands.finish_sketch_history_with_datums(id, &datums)?`.
    pub fn finish_sketch_history_with_datums(
        &mut self,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
        datums: &crate::part_evaluation::CurrentPartDatums,
    ) -> Result<(), SessionError> {
        self.session
            .finish_sketch_history_with_datums(sketch, datums)
    }
    /// Consume the last editor target after history navigation; example: `commands.take_sketch_navigation()`.
    pub fn take_sketch_navigation(&mut self) -> Option<crate::SketchEditNavigation> {
        self.session.take_sketch_navigation()
    }
    /// Start a bounded group on this document; example: `commands.begin("batch")?;`.
    pub fn begin(&mut self, label: &str) -> Result<(), SessionError> {
        self.session.begin(label)
    }

    /// Finish one bounded level on this document; example: `commands.end()?;`.
    pub fn end(&mut self) -> Result<(), SessionError> {
        self.session.end()
    }

    /// Cancel this document's whole group; example: `commands.abort()?;`.
    pub fn abort(&mut self) -> Result<(), SessionError> {
        self.session.abort()
    }

    /// Undo this document's latest step; example: `commands.undo()?;`.
    pub fn undo(&mut self) -> Result<(), SessionError> {
        self.session.undo()
    }

    /// Redo this document's next step; example: `commands.redo()?;`.
    pub fn redo(&mut self) -> Result<(), SessionError> {
        self.session.redo()
    }

    /// Undo an ordinary step with its accepted generation proof.
    /// Example: `commands.undo_with_transition()?` after capturing a candidate.
    pub fn undo_with_transition(
        &mut self,
    ) -> Result<oblikovati_model::history::EvaluationTransition, SessionError> {
        self.session.undo_with_transition()
    }
    /// Redo an ordinary step with its accepted generation proof.
    /// Example: `commands.redo_with_transition()?` after capturing a candidate.
    pub fn redo_with_transition(
        &mut self,
    ) -> Result<oblikovati_model::history::EvaluationTransition, SessionError> {
        self.session.redo_with_transition()
    }
    /// Verify a recomputed candidate and prepare its reply before adoption.
    /// Example: `commands.run_projected("Create", create, verify_and_encode)?`.
    pub fn run_projected<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(&DocumentSnapshot, R) -> Result<T, E>,
    ) -> Result<T, CommandError<E>> {
        self.session.run_projected(label, command, project)
    }
    /// Validate against the unchanged source history; example: `commands.run_projected_with_view(label, edit, validate)?`.
    pub fn run_projected_with_view<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(
            &DocumentSnapshot,
            R,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, E>,
    ) -> Result<T, CommandError<E>> {
        self.session
            .run_projected_with_view(label, command, project)
    }
    /// Return proof of an accepted generation change; example: `commands.run_projected_with_transition(label, edit, project)?`.
    pub fn run_projected_with_transition<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(
            &DocumentSnapshot,
            R,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, E>,
    ) -> Result<(T, oblikovati_model::history::EvaluationTransition), CommandError<E>> {
        self.session
            .run_projected_with_transition(label, command, project)
    }
    /// Validate an unexposed candidate before adopting its identities (#4108).
    /// Example: `commands.run_private_candidate("Edit", edit, validate)?`.
    pub fn run_private_candidate<R, T, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        project: impl FnOnce(
            &DocumentSnapshot,
            R,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, E>,
    ) -> Result<(T, oblikovati_model::history::EvaluationTransition), CommandError<E>> {
        self.session.run_private_candidate(label, command, project)
    }
    /// Run through this document's normal transaction path.
    /// Example: `commands.run("width", |snapshot, _| snapshot.parameters.set_expression(1, "2 cm"))?;`.
    pub fn run<R, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session.run(label, command)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Move this document's cursor without activating it; each step can be vetoed.
    /// Example: `commands.jump_to(0)?` navigates to the opened recipe.
    pub fn jump_to(&mut self, position: i64) -> Result<(), SessionError> {
        self.session.jump_to(position)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Publish a source consequence without an authored transaction.
    /// Example: `commands.refresh_derived_table(7, source.as_ref())?`.
    pub fn refresh_derived_table(
        &mut self,
        table: i64,
        source: Option<&std::collections::BTreeMap<String, oblikovati_model::quantity::Quantity>>,
    ) -> Result<bool, CommandError<oblikovati_model::parameter::ParameterError<u64>>> {
        self.session.refresh_derived_table(table, source)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Choose a body row; a transient choice, so it makes no undo step.
    /// Example: `commands.set_browser_body_selection(key, feature);`.
    pub fn set_browser_body_selection(
        &mut self,
        key: &str,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        self.session.set_browser_body_selection(key, feature);
    }
    /// Apply a modified body click without a history step.
    /// Example: `commands.apply_browser_body_click(key, feature, mode)`.
    pub fn apply_browser_body_click(
        &mut self,
        key: &str,
        feature: oblikovati_model::feature::FeatureId,
        mode: crate::selection::SelectionClickMode,
    ) {
        self.session.apply_browser_body_click(key, feature, mode);
    }
    /// Choose the document's own row; a transient choice, so it makes no undo step.
    /// Example: `commands.set_browser_root_selection();`.
    pub fn set_browser_root_selection(&mut self) {
        self.session.set_browser_root_selection();
    }
    /// Change view state for a resolved body without editing history or saved content.
    /// Example: `commands.set_body_visible(current_body_key, false)?;` returns whether display changed.
    pub fn set_body_visible(&mut self, key: &str, visible: bool) -> Result<bool, String> {
        self.session.body_visibility.set(key, visible)
    }
    /// Change resolved bodies in the active view; example: `commands.set_part_view_body_visibility(&keys, &available, false)?`.
    pub fn set_part_view_body_visibility(
        &mut self,
        keys: &[String],
        available: &std::collections::BTreeSet<String>,
        visible: bool,
    ) -> Result<bool, String> {
        self.session
            .set_part_view_body_visibility(keys, available, visible)
    }
    /// Capture the candidate when body visibility is stored in a part view.
    /// Example: `commands.set_part_view_body_visibility_projected(&keys, &available, false, capture)?`.
    pub fn set_part_view_body_visibility_projected<T>(
        &mut self,
        keys: &[String],
        available: &std::collections::BTreeSet<String>,
        visible: bool,
        project: impl FnOnce(
            &oblikovati_model::history::DocumentSnapshot,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, String>,
    ) -> Result<
        (
            bool,
            Option<(T, oblikovati_model::history::EvaluationTransition)>,
        ),
        String,
    > {
        self.session
            .set_part_view_body_visibility_projected(keys, available, visible, project)
    }
    /// Prepare a view switch against resolved bodies; example: `commands.prepare_part_view_activation(id, camera, &keys)?`.
    pub fn prepare_part_view_activation(
        &self,
        id: u64,
        camera: oblikovati_api::types::PartDesignViewCamera,
        keys: &std::collections::BTreeSet<String>,
    ) -> Result<crate::PreparedPartViewActivation, String> {
        self.session.prepare_part_view_activation(id, camera, keys)
    }
    /// Commit a prepared view switch after its camera is applied; example: `commands.commit_part_view_activation(prepared, camera)?`.
    pub fn commit_part_view_activation(
        &mut self,
        prepared: crate::PreparedPartViewActivation,
        camera: oblikovati_api::types::PartDesignViewCamera,
    ) -> Result<(), String> {
        self.session.commit_part_view_activation(prepared, camera)
    }
    /// Capture a candidate while committing a display switch.
    /// Example: `commands.commit_part_view_activation_projected(prepared, camera, capture)?`.
    pub fn commit_part_view_activation_projected<T>(
        &mut self,
        prepared: crate::PreparedPartViewActivation,
        camera: oblikovati_api::types::PartDesignViewCamera,
        project: impl FnOnce(
            &oblikovati_model::history::DocumentSnapshot,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, String>,
    ) -> Result<(T, oblikovati_model::history::EvaluationTransition), String> {
        self.session
            .commit_part_view_activation_projected(prepared, camera, project)
    }
    /// Prepare one tagged history step; example: `commands.prepare_part_view_navigation(direction, &keys)?`.
    pub fn prepare_part_view_navigation(
        &self,
        direction: crate::PartViewNavigationDirection,
        keys: &std::collections::BTreeSet<String>,
    ) -> Result<crate::PreparedPartViewNavigation, String> {
        self.session.prepare_part_view_navigation(direction, keys)
    }
    /// Capture the target recipe while committing a display history step.
    /// Example: `commands.commit_part_view_navigation_projected(prepared, camera, capture)?`.
    pub fn commit_part_view_navigation_projected<T>(
        &mut self,
        prepared: crate::PreparedPartViewNavigation,
        camera: Option<&oblikovati_api::types::PartDesignViewCamera>,
        project: impl FnOnce(
            &oblikovati_model::history::DocumentSnapshot,
            &oblikovati_model::history::DocumentEditHistory,
        ) -> Result<T, String>,
    ) -> Result<(T, oblikovati_model::history::EvaluationTransition), String> {
        self.session
            .commit_part_view_navigation_projected(prepared, camera, project)
    }
    /// Commit a prepared display history step; example: `commands.commit_part_view_navigation(prepared, camera)?`.
    pub fn commit_part_view_navigation(
        &mut self,
        prepared: crate::PreparedPartViewNavigation,
        camera: Option<&oblikovati_api::types::PartDesignViewCamera>,
    ) -> Result<(), String> {
        self.session.commit_part_view_navigation(prepared, camera)
    }
}
