// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentGuid, GuidSource, IdentityTransitionError, source::GuidIssuer};
use sha2::{Digest, Sha256};

/// SHA-256 fingerprint of serialized recipe bytes, not evaluated geometry.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct RecipeDigest(pub(super) [u8; 32]);

impl RecipeDigest {
    /// Fingerprint the exact recipe encoding supplied by its persistence owner.
    ///
    /// ```
    /// use oblikovati_model::identity::RecipeDigest;
    /// assert_eq!(RecipeDigest::of_recipe(b"recipe"), RecipeDigest::of_recipe(b"recipe"));
    /// ```
    pub fn of_recipe(recipe: &[u8]) -> Self {
        Self(Sha256::digest(recipe).into())
    }
}

impl std::fmt::Display for RecipeDigest {
    fn fmt(&self, formatter: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        for byte in self.0 {
            write!(formatter, "{byte:02x}")?;
        }
        Ok(())
    }
}

/// Identity block for persistence. Candidate transitions never mutate the source.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FileIdentity {
    pub internal_name: DocumentGuid,
    pub revision_id: Option<DocumentGuid>,
    pub database_revision_id: Option<DocumentGuid>,
    pub save_counter: u64,
    pub version_created: String,
    pub version_saved: String,
    pub model_digest: Option<RecipeDigest>,
}

impl FileIdentity {
    /// Create an unsaved document identity with an injected software version.
    ///
    /// ```
    /// use oblikovati_model::identity::{FileIdentity, SystemGuidSource};
    /// let identity = FileIdentity::new("0.1.0", &mut SystemGuidSource)?;
    /// assert_eq!(identity.save_counter, 0);
    /// # Ok::<(), oblikovati_model::identity::IdentityTransitionError>(())
    /// ```
    pub fn new(
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<Self, IdentityTransitionError> {
        Ok(Self {
            internal_name: source.generate()?,
            revision_id: None,
            database_revision_id: None,
            save_counter: 0,
            version_created: version.to_owned(),
            version_saved: String::new(),
            model_digest: None,
        })
    }

    /// Prepare an ordinary save; adopt the returned identity only after storage succeeds.
    /// Unknown recipe fingerprints always produce a fresh database revision.
    ///
    /// ```
    /// use oblikovati_model::identity::{FileIdentity, SystemGuidSource};
    /// let original = FileIdentity::new("0.1.0", &mut SystemGuidSource)?;
    /// let candidate = original.prepare_save(None, "0.1.1", &mut SystemGuidSource)?;
    /// assert_eq!(original.save_counter, 0);
    /// assert_eq!(candidate.save_counter, 1);
    /// # Ok::<(), oblikovati_model::identity::IdentityTransitionError>(())
    /// ```
    pub fn prepare_save(
        &self,
        digest: Option<RecipeDigest>,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<Self, IdentityTransitionError> {
        let mut next = self.clone();
        next.save_counter = self
            .save_counter
            .checked_add(1)
            .ok_or(IdentityTransitionError::CounterOverflow(self.save_counter))?;
        let mut issuer = self.issuer(source);
        next.revision_id = Some(issuer.issue()?);
        next.version_saved = version.to_owned();
        if self.database_revision_id.is_none() || digest.is_none() || digest != self.model_digest {
            next.database_revision_id = Some(issuer.issue()?);
        }
        next.model_digest = digest;
        Ok(next)
    }

    /// Prepare a saved copy with a new document identity and the same model revision.
    ///
    /// ```
    /// use oblikovati_model::identity::{FileIdentity, SystemGuidSource};
    /// let original = FileIdentity::new("0.1.0", &mut SystemGuidSource)?;
    /// let copy = original.prepare_copy("0.1.1", &mut SystemGuidSource)?;
    /// assert_ne!(original.internal_name, copy.internal_name);
    /// # Ok::<(), oblikovati_model::identity::IdentityTransitionError>(())
    /// ```
    pub fn prepare_copy(
        &self,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<Self, IdentityTransitionError> {
        self.copy_identity(self.model_digest, true, version, source)
    }

    /// Copy an exact prepared recipe, preserving its model revision only when the digest matches.
    /// Example: `let copy = identity.prepare_copy_for_recipe(digest, "alpha", source)?;`.
    pub fn prepare_copy_for_recipe(
        &self,
        digest: Option<RecipeDigest>,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<Self, IdentityTransitionError> {
        let preserve_model = digest.is_some() && digest == self.model_digest;
        self.copy_identity(digest, preserve_model, version, source)
    }

    fn copy_identity(
        &self,
        digest: Option<RecipeDigest>,
        preserve_model: bool,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<Self, IdentityTransitionError> {
        let mut issuer = self.issuer(source);
        let mut copy = self.clone();
        copy.internal_name = issuer.issue()?;
        copy.revision_id = Some(issuer.issue()?);
        copy.save_counter = 1;
        copy.version_created = version.to_owned();
        copy.version_saved = version.to_owned();
        if !preserve_model || copy.database_revision_id.is_none() {
            copy.database_revision_id = Some(issuer.issue()?);
        }
        copy.model_digest = digest;
        Ok(copy)
    }

    fn issuer<'source>(&self, source: &'source mut dyn GuidSource) -> GuidIssuer<'source> {
        let excluded = [
            Some(self.internal_name),
            self.revision_id,
            self.database_revision_id,
        ]
        .into_iter()
        .flatten()
        .collect();
        GuidIssuer::new(source, excluded)
    }
}
