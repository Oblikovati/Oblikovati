// SPDX-License-Identifier: GPL-2.0-only

//! Single-owner edits over owned recipe state. Kernel handles are not snapshot contents.
use crate::{
    identity::{RecipeIdAllocator, RecipeIdError},
    parameter::ParameterCollection,
    property::DocumentProperties,
};
mod checkpoints;
mod derived_ids;
mod error;
mod navigation;
mod refresh;
pub use error::{EditError, HistoryError};
pub use refresh::PreparedRefresh;
use std::sync::Arc;

/// Owned document recipe and metadata; identity allocation and file identity are outside history.
#[derive(Clone, Debug)]
pub struct DocumentSnapshot {
    pub attributes: crate::attribute::DocumentAttributes,
    pub interests: crate::interest::DocumentInterests,
    pub resources: std::collections::BTreeMap<String, crate::resource::DocumentResource>,
    pub parameters: ParameterCollection<u64>,
    pub properties: DocumentProperties,
}

/// Bind an immutable recipe to storage before its save identity is prepared.
pub trait DocumentSnapshotTarget: crate::identity::DocumentSaveTarget {
    /// Prepare without replacing storage; return the exact candidate recipe digest.
    /// Example: `let digest = target.prepare(&snapshot, table_mark, parameter_mark)?;` before identity preparation.
    fn prepare(
        &mut self,
        snapshot: &DocumentSnapshot,
        derived_high_water: i64,
        parameter_high_water: u64,
    ) -> Result<Option<crate::identity::RecipeDigest>, Self::Failure>;
}

/// A command can reserve identities but cannot replace or rewind the document allocator.
pub struct RecipeReservations<'a> {
    ids: &'a mut RecipeIdAllocator,
    derived_high_water: &'a mut i64,
}
impl RecipeReservations<'_> {
    /// Permanently reserve an identity; example: `let id = reservations.allocate()?;`.
    pub fn allocate(&mut self) -> Result<u64, RecipeIdError> {
        self.ids.allocate()
    }
}

/// A monotonically assigned transaction handle, valid only while it is the active top frame.
#[must_use]
#[derive(Clone, Debug)]
pub struct TransactionToken {
    sequence: u64,
    owner: Arc<()>,
}
impl PartialEq for TransactionToken {
    fn eq(&self, other: &Self) -> bool {
        self.sequence == other.sequence && Arc::ptr_eq(&self.owner, &other.owner)
    }
}
impl Eq for TransactionToken {}

#[derive(Debug)]
struct OpenTransaction {
    token: TransactionToken,
    label: String,
    before: DocumentSnapshot,
}
#[derive(Debug)]
struct HistoryStep {
    label: String,
    saved: bool,
    before: DocumentSnapshot,
    after: DocumentSnapshot,
}

/// Owns current recipe state, permanent ID reservations, and nested edit history.
#[derive(Debug)]
pub struct DocumentEditHistory {
    current: DocumentSnapshot,
    ids: RecipeIdAllocator,
    derived_high_water: i64,
    open: Vec<OpenTransaction>,
    done: Vec<HistoryStep>,
    undone: Vec<HistoryStep>,
    owner: Arc<()>,
    next_token: u64,
    generation: u64,
}

impl DocumentEditHistory {
    /// Take ownership of loaded state and its allocator; example: `DocumentEditHistory::new(snapshot, ids)?`.
    pub fn new(snapshot: DocumentSnapshot, ids: RecipeIdAllocator) -> Result<Self, RecipeIdError> {
        RecipeIdAllocator::restore(Some(ids.high_water()), snapshot.parameters.ids())?;
        let derived_high_water = snapshot
            .parameters
            .derived_tables()
            .iter()
            .map(|table| table.id)
            .max()
            .unwrap_or(0);
        Ok(Self {
            current: snapshot,
            derived_high_water,
            ids,
            open: Vec::new(),
            done: Vec::new(),
            undone: Vec::new(),
            owner: Arc::new(()),
            next_token: 0,
            generation: 0,
        })
    }

    /// Inspect committed or preview state without bypassing edits; example: `history.snapshot()`.
    pub fn snapshot(&self) -> &DocumentSnapshot {
        &self.current
    }

    /// Read the generation used to reject stale evaluation results; example: `history.generation()`.
    pub fn generation(&self) -> u64 {
        self.generation
    }

    /// Inspect permanent ID reservations; example: `history.high_water()` after undo.
    pub fn high_water(&self) -> u64 {
        self.ids.high_water()
    }

    /// Begin an isolated nested undo unit; example: `let token = history.begin("Edit width")?;`.
    pub fn begin(&mut self, label: &str) -> Result<TransactionToken, HistoryError> {
        let next = self
            .next_token
            .checked_add(1)
            .ok_or(HistoryError::Exhausted("transaction tokens"))?;
        let token = TransactionToken {
            sequence: next,
            owner: Arc::clone(&self.owner),
        };
        self.open.push(OpenTransaction {
            token: token.clone(),
            label: label.into(),
            before: self.current.clone(),
        });
        self.next_token = next;
        Ok(token)
    }

    /// Apply a fallible edit to a candidate; allocated IDs remain reserved even on failure.
    /// Example: `history.edit(&token, |snapshot, ids| edit_parameters(snapshot, ids))?;`.
    pub fn edit<R, E>(
        &mut self,
        token: &TransactionToken,
        edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, EditError<E>> {
        self.require_top(token).map_err(EditError::History)?;
        let generation = self.next_generation().map_err(EditError::History)?;
        let mut candidate = self.current.clone();
        let reserved_before = self.ids.high_water();
        let tables_before = self.derived_high_water;
        let result = edit(
            &mut candidate,
            &mut RecipeReservations {
                ids: &mut self.ids,
                derived_high_water: &mut self.derived_high_water,
            },
        )
        .map_err(EditError::Mutation)?;
        self.validate_candidate_ids(&candidate, reserved_before)
            .map_err(EditError::History)?;
        self.validate_derived_ids(&candidate, tables_before)
            .map_err(EditError::History)?;
        self.current = candidate;
        self.generation = generation;
        Ok(result)
    }

    /// Commit the active frame; nested commits fold into their parent.
    /// Example: `history.commit(&token, false)?;`.
    pub fn commit(
        &mut self,
        token: &TransactionToken,
        merge_previous: bool,
    ) -> Result<(), HistoryError> {
        self.require_top(token)?;
        let frame = self.open.pop().expect("validated frame");
        if !self.open.is_empty() {
            return Ok(());
        }
        if merge_previous && let Some(previous) = self.done.last_mut() {
            previous.after = self.current.clone();
            previous.saved = false;
        } else {
            self.done.push(HistoryStep {
                label: frame.label,
                saved: false,
                before: frame.before,
                after: self.current.clone(),
            });
        }
        self.undone.clear();
        Ok(())
    }

    /// Revert the active frame without releasing IDs; example: `history.abort(&token)?;`.
    pub fn abort(&mut self, token: &TransactionToken) -> Result<(), HistoryError> {
        self.validate_abort(token)?;
        let generation = self.next_generation()?;
        self.current = self.open.pop().expect("validated frame").before;
        self.generation = generation;
        Ok(())
    }

    /// Check rollback before a vetoable event; example: `history.validate_abort(&token)?;`.
    pub fn validate_abort(&self, token: &TransactionToken) -> Result<(), HistoryError> {
        self.require_top(token)?;
        self.next_generation()?;
        Ok(())
    }

    /// Close an unchanged frame without an undo step or redo loss.
    /// Example: `if history.discard_unchanged(&token)? { return Ok(()); }`.
    pub fn discard_unchanged(&mut self, token: &TransactionToken) -> Result<bool, HistoryError> {
        self.require_top(token)?;
        let before = &self.open.last().expect("validated frame").before;
        if !before.parameters.same_recipe(&self.current.parameters)
            || !before.properties.same_recipe(&self.current.properties)
            || before.resources != self.current.resources
            || before.interests != self.current.interests
            || !before.attributes.same_recipe(&self.current.attributes)
        {
            return Ok(false);
        }
        self.open.pop();
        Ok(true)
    }

    fn require_top(&self, token: &TransactionToken) -> Result<(), HistoryError> {
        if !Arc::ptr_eq(&self.owner, &token.owner) {
            return Err(HistoryError::ForeignTransaction);
        }
        let actual = self.open.last().map(|frame| &frame.token);
        if actual != Some(token) {
            return Err(HistoryError::Inactive {
                supplied: token.clone(),
                active: actual.cloned(),
            });
        }
        Ok(())
    }

    fn next_generation(&self) -> Result<u64, HistoryError> {
        self.generation
            .checked_add(1)
            .ok_or(HistoryError::Exhausted("evaluation generations"))
    }

    fn validate_candidate_ids(
        &self,
        candidate: &DocumentSnapshot,
        reserved_before: u64,
    ) -> Result<(), HistoryError> {
        for &id in candidate.parameters.ids() {
            if self.current.parameters.get(id).is_none()
                && (id <= reserved_before || id > self.ids.high_water())
            {
                return Err(HistoryError::UnreservedIdentity(id));
            }
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests;
