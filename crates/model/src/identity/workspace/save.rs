// SPDX-License-Identifier: GPL-2.0-only

use super::DocumentIdentityIndex;
use crate::identity::{FileIdentity, GuidSource, IdentityTransitionError, RecipeDigest};
use std::{error::Error, fmt};

/// Storage boundary for a prepared recipe and its candidate identity.
/// The implementation must commit both atomically, or return an error.
pub trait DocumentSaveTarget {
    type Failure: Error + 'static;

    /// Persist the candidate with the recipe snapshot owned by this target.
    /// The caller-supplied digest must describe that same snapshot.
    ///
    /// ```text
    /// target.persist(&candidate)?; // success permits adopting candidate in memory
    /// ```
    fn persist(&mut self, candidate: &FileIdentity) -> Result<(), Self::Failure>;
}

/// A refused save or a storage failure; no candidate was adopted in memory.
#[derive(Debug)]
pub enum DocumentSaveError<Failure> {
    MissingDocument(u64),
    Preparation(IdentityTransitionError),
    Storage(Failure),
}

impl DocumentIdentityIndex {
    /// Prepare and persist a save before replacing the registered identity.
    /// The exclusive borrow prevents overlapping save adoption in this owner.
    ///
    /// ```text
    /// index.save(session_id, recipe_digest, version, guid_source, save_target)?;
    /// // Registered revisions now describe the successfully written snapshot.
    /// ```
    pub fn save<Target: DocumentSaveTarget>(
        &mut self,
        session_id: u64,
        digest: Option<RecipeDigest>,
        version: &str,
        source: &mut dyn GuidSource,
        target: &mut Target,
    ) -> Result<(), DocumentSaveError<Target::Failure>> {
        let current = self
            .documents
            .get_mut(&session_id)
            .ok_or(DocumentSaveError::MissingDocument(session_id))?;
        let candidate = current
            .prepare_save(digest, version, source)
            .map_err(DocumentSaveError::Preparation)?;
        target
            .persist(&candidate)
            .map_err(DocumentSaveError::Storage)?;
        // prepare_save preserves the GUID, so the reverse identity index is unchanged.
        *current = candidate;
        Ok(())
    }
}

impl<Failure: fmt::Display> fmt::Display for DocumentSaveError<Failure> {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::MissingDocument(id) => write!(
                formatter,
                "save document {id}: expected a registered session ID"
            ),
            Self::Preparation(error) => write!(formatter, "prepare document save: {error}"),
            Self::Storage(error) => write!(formatter, "persist document save: {error}"),
        }
    }
}

impl<Failure: Error + 'static> Error for DocumentSaveError<Failure> {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::MissingDocument(_) => None,
            Self::Preparation(error) => Some(error),
            Self::Storage(error) => Some(error),
        }
    }
}
