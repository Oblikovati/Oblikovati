// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentGuid, FileIdentity, GuidSource, IdentityTransitionError};
use std::{collections::BTreeMap, error::Error, fmt};

mod copy;
mod save;
pub use save::{DocumentSaveError, DocumentSaveTarget};

/// Owns the identity claims of live documents, independently of file I/O.
#[derive(Debug, Default)]
pub struct DocumentIdentityIndex {
    documents: BTreeMap<u64, FileIdentity>,
    owners: BTreeMap<DocumentGuid, u64>,
}

/// An open result the host must inspect before exposing a loaded document.
/// Reassignment requires a dirty document and a visible identity-change notice.
#[must_use]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum IdentityRegistration {
    Preserved,
    Reassigned {
        previous: DocumentGuid,
        current: DocumentGuid,
    },
}

/// An identity registration failure. Existing claims remain unchanged.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum IdentityRegistrationError {
    InvalidSessionId(u64),
    AlreadyRegistered(u64),
    MissingDocument(u64),
    Generation(IdentityTransitionError),
}

impl DocumentIdentityIndex {
    /// Register a loaded identity, explicitly reporting a duplicate-file reassignment.
    ///
    /// ```
    /// use oblikovati_model::identity::{DocumentIdentityIndex, FileIdentity, SystemGuidSource};
    /// let mut index = DocumentIdentityIndex::default();
    /// let identity = FileIdentity::new("alpha", &mut SystemGuidSource)?;
    /// let outcome = index.register(1, identity, &mut SystemGuidSource)?;
    /// # Ok::<(), Box<dyn std::error::Error>>(())
    /// ```
    pub fn register(
        &mut self,
        session_id: u64,
        mut identity: FileIdentity,
        source: &mut dyn GuidSource,
    ) -> Result<IdentityRegistration, IdentityRegistrationError> {
        self.validate_session_id(session_id)?;
        let outcome = self.resolve_collision(&mut identity, source)?;
        self.owners.insert(identity.internal_name, session_id);
        self.documents.insert(session_id, identity);
        Ok(outcome)
    }

    /// Inspect an identity without allowing edits that bypass the GUID index.
    ///
    /// ```
    /// use oblikovati_model::identity::DocumentIdentityIndex;
    /// assert!(DocumentIdentityIndex::default().identity(1).is_none());
    /// ```
    pub fn identity(&self, session_id: u64) -> Option<&FileIdentity> {
        self.documents.get(&session_id)
    }

    /// Find the live owner of a persistent document identity.
    ///
    /// ```
    /// use oblikovati_model::identity::{DocumentGuid, DocumentIdentityIndex};
    /// let guid: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse()?;
    /// assert_eq!(DocumentIdentityIndex::default().owner(guid), None);
    /// # Ok::<(), oblikovati_model::identity::DocumentGuidError>(())
    /// ```
    pub fn owner(&self, guid: DocumentGuid) -> Option<u64> {
        self.owners.get(&guid).copied()
    }

    /// Release a closed document's claim; the original file can then reopen.
    ///
    /// ```
    /// use oblikovati_model::identity::DocumentIdentityIndex;
    /// assert!(DocumentIdentityIndex::default().close(1).is_err());
    /// ```
    pub fn close(&mut self, session_id: u64) -> Result<FileIdentity, IdentityRegistrationError> {
        let identity = self
            .documents
            .remove(&session_id)
            .ok_or(IdentityRegistrationError::MissingDocument(session_id))?;
        self.owners.remove(&identity.internal_name);
        Ok(identity)
    }

    fn validate_session_id(&self, session_id: u64) -> Result<(), IdentityRegistrationError> {
        if session_id == 0 {
            return Err(IdentityRegistrationError::InvalidSessionId(session_id));
        }
        if self.documents.contains_key(&session_id) {
            return Err(IdentityRegistrationError::AlreadyRegistered(session_id));
        }
        Ok(())
    }

    fn resolve_collision(
        &self,
        identity: &mut FileIdentity,
        source: &mut dyn GuidSource,
    ) -> Result<IdentityRegistration, IdentityRegistrationError> {
        let previous = identity.internal_name;
        if !self.owners.contains_key(&previous) {
            return Ok(IdentityRegistration::Preserved);
        }
        let current = source
            .generate()
            .map_err(IdentityRegistrationError::Generation)?;
        if self.owners.contains_key(&current)
            || [identity.revision_id, identity.database_revision_id].contains(&Some(current))
        {
            return Err(IdentityRegistrationError::Generation(
                IdentityTransitionError::RepeatedGuid(current),
            ));
        }
        identity.internal_name = current;
        Ok(IdentityRegistration::Reassigned { previous, current })
    }
}

impl fmt::Display for IdentityRegistrationError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InvalidSessionId(id) => {
                write!(formatter, "document session ID {id}: expected nonzero")
            }
            Self::AlreadyRegistered(id) => write!(
                formatter,
                "document session ID {id}: expected an unregistered ID"
            ),
            Self::MissingDocument(id) => write!(
                formatter,
                "document session ID {id}: expected an open document"
            ),
            Self::Generation(error) => error.fmt(formatter),
        }
    }
}

impl Error for IdentityRegistrationError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Generation(error) => Some(error),
            _ => None,
        }
    }
}
