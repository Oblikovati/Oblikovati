// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentIdentityIndex, DocumentSaveError, DocumentSaveTarget};
use crate::identity::{FileIdentity, GuidSource, IdentityTransitionError, RecipeDigest};

impl DocumentIdentityIndex {
    /// Persist an independent copy without changing the registered source or claiming the copy.
    /// Example: `let copy_identity = index.save_copy(id, digest, "alpha", source, target)?;`.
    pub fn save_copy<T: DocumentSaveTarget>(
        &self,
        id: u64,
        digest: Option<RecipeDigest>,
        version: &str,
        source: &mut dyn GuidSource,
        target: &mut T,
    ) -> Result<FileIdentity, DocumentSaveError<T::Failure>> {
        let current = self
            .documents
            .get(&id)
            .ok_or(DocumentSaveError::MissingDocument(id))?;
        let candidate = current
            .prepare_copy_for_recipe(digest, version, source)
            .map_err(DocumentSaveError::Preparation)?;
        if self.owners.contains_key(&candidate.internal_name) {
            return Err(DocumentSaveError::Preparation(
                IdentityTransitionError::RepeatedGuid(candidate.internal_name),
            ));
        }
        target
            .persist(&candidate)
            .map_err(DocumentSaveError::Storage)?;
        Ok(candidate)
    }
}
