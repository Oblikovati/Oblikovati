// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentEditHistory, DocumentSnapshot, EditError, HistoryError, RecipeReservations};
use crate::identity::RecipeIdAllocator;
/// A snapshot-backed evaluator cannot allocate or publish recipe identities (#4080).
///
/// Example: `SnapshotReplay::new(&snapshot)?.history()` supplies an immutable replay history.
///
/// ```compile_fail
/// use oblikovati_model::history::{DocumentSnapshot, SnapshotReplay};
/// fn mutate(snapshot: &DocumentSnapshot) {
///     let replay = SnapshotReplay::new(snapshot).unwrap();
///     replay.history().begin("edit").unwrap();
/// }
/// ```
pub struct SnapshotReplay(DocumentEditHistory);

impl SnapshotReplay {
    /// Reserve live IDs for evaluation; example: `SnapshotReplay::new(&snapshot)?`.
    pub fn new(snapshot: &DocumentSnapshot) -> Result<Self, HistoryError> {
        let highest = snapshot
            .parameters
            .ids()
            .iter()
            .copied()
            .chain(super::model_state_ids::authored_ids(snapshot))
            .max()
            .unwrap_or(0);
        let ids = RecipeIdAllocator::restore(Some(highest), snapshot.parameters.ids())?;
        Ok(Self(DocumentEditHistory::new(snapshot.clone(), ids)?))
    }

    /// Borrow the immutable history for an evaluator; example: `replay.history().snapshot()`.
    pub fn history(&self) -> &DocumentEditHistory {
        &self.0
    }
}

/// A validated private edit holds its owner until it is published or discarded (#4122).
/// Dropping it preserves the live generation, history and reserved identities.
/// Example: `let (candidate, value) = history.prepare_private_edit("width", edit)?;`.
pub struct PreparedPrivateEdit<'a> {
    owner: &'a mut DocumentEditHistory,
    candidate: DocumentEditHistory,
    generation: u64,
    source: super::EvaluationScope,
    action: PreparedAction,
    seal: std::sync::Arc<()>,
}

enum PreparedAction {
    Edit(String, super::HistoryTransitionKind),
    Undo,
    Redo,
}

/// Select the history transition to prepare without changing the live cursor.
/// Example: `history.prepare_private_navigation(PreparedNavigation::Undo)?`.
#[derive(Clone, Copy)]
pub enum PreparedNavigation {
    Undo,
    Redo,
}

/// Identifies one private candidate, including one later abandoned (#4122).
/// Example: `let seal = prepared.seal();` before preparing dependent results.
#[derive(Clone)]
pub struct PreparedCandidateSeal(std::sync::Arc<()>);

impl PreparedCandidateSeal {
    /// Check the exact private candidate; example: `seal.matches(&prepared)`.
    pub fn matches(&self, prepared: &PreparedPrivateEdit<'_>) -> bool {
        std::sync::Arc::ptr_eq(&self.0, &prepared.seal)
    }
}

impl PreparedPrivateEdit<'_> {
    /// Issue an identity for this exact candidate; example: `let seal = prepared.seal();`.
    pub fn seal(&self) -> PreparedCandidateSeal {
        PreparedCandidateSeal(self.seal.clone())
    }

    /// Read the validated candidate; example: `prepared.snapshot()`.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        self.candidate.snapshot()
    }

    /// Read the unchanged live source; example: `prepared.source()` before replay capture.
    pub fn source(&self) -> &DocumentEditHistory {
        self.owner
    }

    /// Check for an authored change; example: `if prepared.changed() { notify(); }`.
    pub fn changed(&self) -> bool {
        !super::sketch_consequence::same_history_recipe(
            &self.owner.current,
            &self.candidate.current,
        )
    }

    /// Classify an edit before publication; example: `prepared.set_transition_kind(kind)`.
    pub fn set_transition_kind(&mut self, kind: super::HistoryTransitionKind) {
        if let PreparedAction::Edit(_, current) = &mut self.action {
            *current = kind;
        }
    }

    /// Publish the validated edit as one undo step without further validation.
    /// Example: `let transition = prepared.publish();` after all observers accept.
    pub fn publish(self) -> super::EvaluationTransition {
        let changed = self.changed();
        match self.action {
            PreparedAction::Edit(label, kind) if changed => {
                self.owner.done.push(super::HistoryStep {
                    kind,
                    label,
                    saved: false,
                    before: self.owner.current.clone(),
                    after: self.candidate.current.clone(),
                    sketch_edit: None,
                });
                self.owner.undone.clear();
            }
            PreparedAction::Edit(_, _) => {}
            PreparedAction::Undo => {
                let step = self.owner.done.pop().expect("prepared undo step");
                self.owner.undone.push(step);
            }
            PreparedAction::Redo => {
                let step = self.owner.undone.pop().expect("prepared redo step");
                self.owner.done.push(step);
            }
        }
        self.owner
            .adopt_private_candidate(self.candidate, self.generation);
        super::EvaluationTransition::new(self.source, self.owner.evaluation_scope())
    }
}

impl DocumentEditHistory {
    /// Prepare an edit without changing live history or reserving live identities (#4122).
    /// Example: `let (prepared, result) = history.prepare_private_edit("edit", command)?;`.
    pub fn prepare_private_edit<R, E>(
        &mut self,
        label: &str,
        edit: impl FnOnce(
            &mut DocumentSnapshot,
            &mut RecipeReservations<'_>,
            &DocumentEditHistory,
        ) -> Result<R, E>,
    ) -> Result<(PreparedPrivateEdit<'_>, R), EditError<E>> {
        self.require_closed().map_err(EditError::History)?;
        let generation = self.next_generation().map_err(EditError::History)?;
        let source = self.evaluation_scope();
        let mut candidate = self.preview_owner().map_err(EditError::History)?;
        let token = candidate.begin(label).map_err(EditError::History)?;
        let result = candidate.edit(&token, |snapshot, ids| edit(snapshot, ids, self))?;
        Ok((
            PreparedPrivateEdit {
                owner: self,
                candidate,
                generation,
                source,
                action: PreparedAction::Edit(label.into(), super::HistoryTransitionKind::Ordinary),
                seal: std::sync::Arc::new(()),
            },
            result,
        ))
    }

    /// Prepare undo or redo while retaining the live cursor and generation (#4122).
    /// Example: `let candidate = history.prepare_private_navigation(PreparedNavigation::Undo)?;`.
    pub fn prepare_private_navigation(
        &mut self,
        direction: PreparedNavigation,
    ) -> Result<PreparedPrivateEdit<'_>, HistoryError> {
        let snapshot = match direction {
            PreparedNavigation::Undo => {
                self.undo_label()?;
                self.done.last().expect("validated undo").before.clone()
            }
            PreparedNavigation::Redo => {
                self.redo_label()?;
                self.undone.last().expect("validated redo").after.clone()
            }
        };
        let generation = self.next_generation()?;
        let source = self.evaluation_scope();
        let mut candidate = self.preview_owner()?;
        candidate.current = snapshot;
        let action = match direction {
            PreparedNavigation::Undo => PreparedAction::Undo,
            PreparedNavigation::Redo => PreparedAction::Redo,
        };
        Ok(PreparedPrivateEdit {
            owner: self,
            candidate,
            generation,
            source,
            action,
            seal: std::sync::Arc::new(()),
        })
    }

    /// Adopt an unexposed candidate and its reservations only after validation succeeds (#4108).
    ///
    /// The edit must not publish candidate identities before returning success.
    /// Example: `history.edit_private_with_view_transition(&token, |candidate, ids, source| validate(candidate, ids, source))?`.
    pub fn edit_private_with_view_transition<R, E>(
        &mut self,
        token: &super::TransactionToken,
        edit: impl FnOnce(
            &mut DocumentSnapshot,
            &mut RecipeReservations<'_>,
            &DocumentEditHistory,
        ) -> Result<R, E>,
    ) -> Result<(R, super::EvaluationTransition), EditError<E>> {
        self.require_top(token).map_err(EditError::History)?;
        let generation = self.next_generation().map_err(EditError::History)?;
        let source = self.evaluation_scope();
        let mut view = self.preview_owner().map_err(EditError::History)?;
        view.generation = self.generation.clone();
        let mut private = self.preview_owner().map_err(EditError::History)?;
        let private_token = private
            .begin("Private candidate")
            .map_err(EditError::History)?;
        let result = private.edit(&private_token, |candidate, ids| edit(candidate, ids, &view))?;
        self.adopt_private_candidate(private, generation);
        Ok((
            result,
            super::EvaluationTransition::new(source, self.evaluation_scope()),
        ))
    }

    fn adopt_private_candidate(&mut self, candidate: Self, generation: u64) {
        self.current = candidate.current;
        self.ids = candidate.ids;
        self.sketch_ids = candidate.sketch_ids;
        self.feature_ids = candidate.feature_ids;
        self.occurrence_ids = candidate.occurrence_ids;
        self.sketch_entity_ids = candidate.sketch_entity_ids;
        self.sketch_constraint_ids = candidate.sketch_constraint_ids;
        self.dimension_names = candidate.dimension_names;
        self.derived_high_water = candidate.derived_high_water;
        self.generation
            .store(generation, std::sync::atomic::Ordering::Release);
    }
    /// Return a generation transition only after a source-view edit is accepted.
    /// Example: `let (value, transition) = history.edit_with_view_transition(&token, edit)?;`.
    pub fn edit_with_view_transition<R, E>(
        &mut self,
        token: &super::TransactionToken,
        edit: impl FnOnce(
            &mut DocumentSnapshot,
            &mut RecipeReservations<'_>,
            &DocumentEditHistory,
        ) -> Result<R, E>,
    ) -> Result<(R, super::EvaluationTransition), EditError<E>> {
        let source = self.evaluation_scope();
        let result = self.edit_with_view(token, edit)?;
        Ok((
            result,
            super::EvaluationTransition::new(source, self.evaluation_scope()),
        ))
    }
    /// Edit while a read-only view keeps the source generation available to validation.
    /// Example: `history.edit_with_view(&token, |candidate, ids, source| validate(candidate, source))?`.
    pub fn edit_with_view<R, E>(
        &mut self,
        token: &super::TransactionToken,
        edit: impl FnOnce(
            &mut DocumentSnapshot,
            &mut RecipeReservations<'_>,
            &DocumentEditHistory,
        ) -> Result<R, E>,
    ) -> Result<R, EditError<E>> {
        let mut view = self.preview_owner().map_err(EditError::History)?;
        view.generation = self.generation.clone();
        self.edit(token, |candidate, ids| edit(candidate, ids, &view))
    }
    /// Validate a refresh on a private owner without publishing it (#309).
    /// Example: `history.preview_refresh(|snapshot| refresh(snapshot))?`.
    pub fn preview_refresh<E>(
        &self,
        refresh: impl FnOnce(&mut DocumentSnapshot) -> Result<(), E>,
    ) -> Result<(), EditError<E>> {
        self.require_closed().map_err(EditError::History)?;
        let mut private = self.preview_owner().map_err(EditError::History)?;
        private.prepare_refresh(refresh).map(|_| ())
    }

    /// Validate an edit on a private current-state copy without consuming owner IDs or history.
    /// Example: `history.preview(|snapshot, ids| snapshot.create_sketch_primitive(sketch, &input, ids))?;`.
    pub fn preview<R, E>(
        &self,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, EditError<E>> {
        let mut private = self.preview_owner().map_err(EditError::History)?;
        let token = private
            .begin("Transient preview")
            .map_err(EditError::History)?;
        private.edit(&token, command)
    }
    fn preview_owner(&self) -> Result<Self, HistoryError> {
        let ids =
            RecipeIdAllocator::restore(Some(self.high_water()), self.current.parameters.ids())?;
        Self::new(self.current.clone(), ids)?
            .with_sketch_reservations(self.sketch_reservations())?
            .with_feature_reservations(self.feature_high_water())?
            .with_occurrence_reservations(self.occurrence_high_water())?
            .with_derived_reservations(self.derived_high_water())
            .map_err(HistoryError::from)
    }
}
