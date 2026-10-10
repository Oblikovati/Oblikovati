// SPDX-License-Identifier: GPL-2.0-only

//! Ordered open-document ownership without file I/O or UI dependencies.
mod commands;
mod derived_edit;
pub use commands::SessionCommands;
mod copy;
mod error;
mod interests;
mod lifecycle;
mod save;
use crate::{DocumentSession, TransactionObserver};
pub use error::WorkspaceError;
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{
        DocumentGuid, DocumentIdentityIndex, FileIdentity, GuidSource, IdentityRegistration,
        IdentityRegistrationError, IdentityTransitionError,
    },
};
pub use save::WorkspaceSaveError;
use std::collections::BTreeMap;

/// Policy and observations at the workspace boundary; callbacks cannot mutate it.
pub trait WorkspaceObserver {
    /// Match an exact registered add-in ID or connected client name.
    /// Example: return whether the live host registry contains `client`.
    fn interest_client_present(&self, client: &str) -> bool;
    /// Report unavailable clients after an open completes; this cannot veto the open.
    /// Example: show `clients` in the status bar for `display_name`.
    fn unavailable_interest_clients(
        &mut self,
        document: u64,
        display_name: &str,
        clients: &[String],
    );
    /// Observe a new unsaved document; example: add a tab with its dirty marker set.
    fn created(&mut self, document: u64, registration: IdentityRegistration);
    /// Observe a successful Save As binding; example: update the document's tab path.
    fn path_changed(
        &mut self,
        document: u64,
        previous: &std::path::Path,
        current: &std::path::Path,
    );
    /// Refuse a new file open before reading it; example: return `Err("project is locked".into())`.
    fn before_open(&mut self, path: &std::path::Path) -> Result<(), String>;
    /// Refuse a save before preparation or I/O; example: return `Err("document is locked".into())`.
    fn before_save(&mut self, document: u64, snapshot: &DocumentSnapshot) -> Result<(), String>;
    /// Observe a completed save after identity and baseline adoption; example: clear the dirty marker.
    fn saved(&mut self, document: u64);
    /// Report an open and any required identity notice; example: display a `Reassigned` outcome.
    fn opened(&mut self, document: u64, registration: IdentityRegistration);
    /// Observe explicit activation; example: select the tab for `document`.
    fn activated(&mut self, document: u64);
    /// Refuse explicit discard-close before state changes; example: return `Err("tool is busy".into())`.
    fn before_close(&mut self, document: u64, snapshot: &DocumentSnapshot) -> Result<(), String>;
    /// Observe removal and its remaining active document; example: remove the closed tab.
    fn closed(&mut self, document: u64, active: Option<u64>);
}

/// Whether registration changes focus; background loads preserve the existing focus.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum OpenActivation {
    Activate,
    Background,
}

/// Both the session ID and identity reassignment must reach the caller.
#[must_use]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct RegisteredDocument {
    pub session_id: u64,
    pub identity: IdentityRegistration,
}

struct WorkspaceEntry<O> {
    session: DocumentSession<O>,
    baseline: DocumentSnapshot,
    baseline_derived_high_water: i64,
    baseline_parameter_high_water: u64,
    baseline_sketch_marks: oblikovati_model::history::SketchReservationMarks,
    identity_dirty: bool,
    display_name: String,
}

/// Own all open sessions and their GUID claims. IDs are never reused within this workspace.
pub struct WorkspaceSession<O, W> {
    entries: BTreeMap<u64, WorkspaceEntry<O>>,
    order: Vec<u64>,
    active: Option<u64>,
    identities: DocumentIdentityIndex,
    next_id: u64,
    observer: W,
    clients: crate::clients::ConnectedClients,
}

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    pub(crate) fn notify_path_changed(
        &mut self,
        id: u64,
        previous: &std::path::Path,
        current: &std::path::Path,
    ) {
        self.observer.path_changed(id, previous, current);
    }
    pub(crate) fn approve_open(&mut self, path: &std::path::Path) -> Result<(), String> {
        self.observer.before_open(path)
    }
    /// Inject workspace policy; example: `let workspace = WorkspaceSession::new(observer);`.
    pub fn new(observer: W) -> Self {
        Self {
            entries: BTreeMap::new(),
            order: Vec::new(),
            active: None,
            identities: DocumentIdentityIndex::default(),
            next_id: 0,
            observer,
            clients: crate::clients::ConnectedClients::default(),
        }
    }

    /// Register validated loaded state; example: `workspace.register(history, identity, observer, source, OpenActivation::Activate)?`.
    pub fn register(
        &mut self,
        history: DocumentEditHistory,
        identity: FileIdentity,
        observer: O,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
    ) -> Result<RegisteredDocument, WorkspaceError> {
        let registration = self.adopt_document(history, identity, observer, source)?;
        self.observer
            .opened(registration.session_id, registration.identity);
        if activation == OpenActivation::Activate {
            self.activate(registration.session_id)?;
        }
        Ok(registration)
    }

    pub(crate) fn register_created(
        &mut self,
        history: DocumentEditHistory,
        identity: FileIdentity,
        observer: O,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, WorkspaceError> {
        if self.identities.owner(identity.internal_name).is_some() {
            return Err(WorkspaceError::Identity(
                IdentityRegistrationError::Generation(IdentityTransitionError::RepeatedGuid(
                    identity.internal_name,
                )),
            ));
        }
        let registration = self.adopt_document(history, identity, observer, source)?;
        self.entries
            .get_mut(&registration.session_id)
            .expect("adopted document")
            .identity_dirty = true;
        self.observer
            .created(registration.session_id, registration.identity);
        self.activate(registration.session_id)?;
        Ok(registration)
    }

    fn adopt_document(
        &mut self,
        history: DocumentEditHistory,
        identity: FileIdentity,
        observer: O,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, WorkspaceError> {
        let id = next_session_id(self.next_id)?;
        self.next_id = id;
        let identity = self.identities.register(id, identity, source)?;
        let entry = WorkspaceEntry {
            baseline: history.snapshot().clone(),
            baseline_derived_high_water: history.derived_high_water(),
            baseline_parameter_high_water: history.high_water(),
            baseline_sketch_marks: history.sketch_reservations(),
            display_name: String::new(),
            session: DocumentSession::new(id, history, observer),
            identity_dirty: identity != IdentityRegistration::Preserved,
        };
        self.entries.insert(id, entry);
        self.order.push(id);
        Ok(RegisteredDocument {
            session_id: id,
            identity,
        })
    }

    /// Inspect documents in open order; example: `workspace.documents()`.
    pub fn documents(&self) -> &[u64] {
        &self.order
    }

    /// Read the current focus; example: `workspace.active_document()`.
    pub fn active_document(&self) -> Option<u64> {
        self.active
    }

    /// Inspect workspace observations; example: `workspace.observer().events()`.
    pub fn observer(&self) -> &W {
        &self.observer
    }

    /// Inspect one named document; example: `workspace.session(id)?.snapshot()`.
    pub fn session(&self, id: u64) -> Result<&DocumentSession<O>, WorkspaceError> {
        self.entries
            .get(&id)
            .map(|entry| &entry.session)
            .ok_or(WorkspaceError::Missing(id))
    }

    /// Route commands to a fixed document across focus changes; example: `workspace.commands(id)?.end()?`.
    pub fn commands(&mut self, id: u64) -> Result<SessionCommands<'_, O>, WorkspaceError> {
        self.entries
            .get_mut(&id)
            .map(|entry| SessionCommands {
                session: &mut entry.session,
            })
            .ok_or(WorkspaceError::Missing(id))
    }

    /// Resolve focus for a new command; example: `workspace.active_commands()?.undo()?`.
    pub fn active_commands(&mut self) -> Result<SessionCommands<'_, O>, WorkspaceError> {
        let id = self.active.ok_or(WorkspaceError::NoActive)?;
        self.commands(id)
    }

    /// Inspect persistent identity without changing its index; example: `workspace.identity(id)?`.
    pub fn identity(&self, id: u64) -> Result<&FileIdentity, WorkspaceError> {
        self.identities
            .identity(id)
            .ok_or(WorkspaceError::Missing(id))
    }

    /// Find an open GUID owner; example: `workspace.owner(guid)`.
    pub fn owner(&self, guid: DocumentGuid) -> Option<u64> {
        self.identities.owner(guid)
    }

    /// Compare current recipe with loaded state; identity reassignment remains dirty across undo.
    /// Example: `workspace.needs_save(id)?` before prompting to close.
    pub fn needs_save(&self, id: u64) -> Result<bool, WorkspaceError> {
        let entry = self.entries.get(&id).ok_or(WorkspaceError::Missing(id))?;
        let current = entry.session.snapshot();
        Ok(entry.identity_dirty
            || entry.baseline_sketch_marks != entry.session.history.sketch_reservations()
            || !entry.baseline.sketches.same_recipe(&current.sketches)
            || entry.baseline_parameter_high_water != entry.session.high_water()
            || entry.baseline_derived_high_water != entry.session.history.derived_high_water()
            || !entry.baseline.parameters.same_recipe(&current.parameters)
            || !entry.baseline.properties.same_recipe(&current.properties)
            || entry.baseline.attachments != current.attachments
            || entry.baseline.resources != current.resources
            || entry.baseline.interests != current.interests
            || !entry.baseline.attributes.same_recipe(&current.attributes))
    }
}

fn next_session_id(previous: u64) -> Result<u64, WorkspaceError> {
    previous.checked_add(1).ok_or(WorkspaceError::Exhausted)
}

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Read the file owner's display name without changing focus.
    /// Example: `workspace.display_name(id)?` for a history browser title.
    pub fn display_name(&self, id: u64) -> Result<&str, WorkspaceError> {
        self.entries
            .get(&id)
            .map(|entry| entry.display_name.as_str())
            .ok_or(WorkspaceError::Missing(id))
    }

    pub(crate) fn set_file_display_name(&mut self, id: u64, name: String) {
        self.entries
            .get_mut(&id)
            .expect("registered file owner")
            .display_name = name;
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Inspect external connections without changing document state; example: `workspace.connected_clients().entries()`.
    pub fn connected_clients(&self) -> &crate::clients::ConnectedClients {
        &self.clients
    }
    /// Change external connections independently from document history; example: `workspace.connected_clients_mut().register("cam")?`.
    pub fn connected_clients_mut(&mut self) -> &mut crate::clients::ConnectedClients {
        &mut self.clients
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn session_ids_never_wrap_to_zero() {
        assert_eq!(next_session_id(0), Ok(1));
        assert_eq!(next_session_id(u64::MAX - 1), Ok(u64::MAX));
        let error = next_session_id(u64::MAX).unwrap_err();
        assert_eq!(error, WorkspaceError::Exhausted);
        assert!(error.to_string().contains("u64::MAX"));
    }
}
