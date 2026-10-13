// SPDX-License-Identifier: GPL-2.0-only

//! Document-scoped recipe identity, independent of session IDs and geometry.

use std::{error::Error, fmt, str::FromStr};
use uuid::Uuid;

mod allocation;
mod bindings;
pub use bindings::ExactReferenceBindings;
mod envelope;
mod file;
mod record;
mod source;
mod workspace;
pub use allocation::{RecipeIdAllocator, RecipeIdError};
pub use envelope::{ReferenceEnvelope, ReferenceEnvelopeError, ReferenceScheme};
pub use file::{FileIdentity, RecipeDigest};
pub use record::{FileIdentityRecord, IdentityRecordError};
pub use source::{GuidSource, IdentityTransitionError, SystemGuidSource};
pub use workspace::{
    DocumentIdentityIndex, DocumentSaveError, DocumentSaveTarget, IdentityRegistration,
    IdentityRegistrationError,
};

/// A saved document GUID used as a namespace for persistent sketch references.
#[derive(Clone, Copy, Debug, Eq, Hash, Ord, PartialEq, PartialOrd)]
pub struct DocumentGuid(Uuid);

/// A derived sketch reference UUID. This value does not resolve geometry.
#[derive(Clone, Copy, Debug, Eq, Hash, Ord, PartialEq, PartialOrd)]
pub struct SketchReferenceUuid(Uuid);

/// Invalid document GUID text, retained for a useful persistence diagnostic.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct DocumentGuidError(String);

impl DocumentGuid {
    /// Derive the saved sketch's name from its persistent document-local ID.
    ///
    /// ```
    /// use oblikovati_model::identity::DocumentGuid;
    /// let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse()?;
    /// assert_eq!(doc.sketch_key(42), doc.sketch_key(42));
    /// # Ok::<(), oblikovati_model::identity::DocumentGuidError>(())
    /// ```
    pub fn sketch_key(self, local_id: u64) -> SketchReferenceUuid {
        self.derive_reference("sketch", local_id)
    }

    /// Derive an entity name in a namespace distinct from sketch names.
    ///
    /// ```
    /// use oblikovati_model::identity::DocumentGuid;
    /// let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse()?;
    /// assert_ne!(doc.sketch_key(42), doc.sketch_entity_key(42));
    /// # Ok::<(), oblikovati_model::identity::DocumentGuidError>(())
    /// ```
    pub fn sketch_entity_key(self, local_id: u64) -> SketchReferenceUuid {
        self.derive_reference("sketchEntity", local_id)
    }

    fn derive_reference(self, kind: &str, local_id: u64) -> SketchReferenceUuid {
        // #3585: UUIDv5 and these exact tags preserve the Go file reference values.
        // SHA-1 here defines identity compatibility; it is not authentication.
        let name = format!("{kind}:{local_id}");
        SketchReferenceUuid(Uuid::new_v5(&self.0, name.as_bytes()))
    }
}

impl FromStr for DocumentGuid {
    type Err = DocumentGuidError;

    fn from_str(source: &str) -> Result<Self, Self::Err> {
        // The UUID library also accepts URNs and braces, which are not file GUIDs.
        if !matches!(source.len(), 32 | 36) {
            return Err(DocumentGuidError(source.to_owned()));
        }
        Uuid::parse_str(source)
            .map(Self)
            .map_err(|_| DocumentGuidError(source.to_owned()))
    }
}

impl fmt::Display for DocumentGuid {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.hyphenated().fmt(formatter)
    }
}

impl fmt::Display for SketchReferenceUuid {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.hyphenated().fmt(formatter)
    }
}

impl fmt::Display for DocumentGuidError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            formatter,
            "document GUID {:?}: expected 32 hexadecimal digits or an 8-4-4-4-12 GUID",
            self.0
        )
    }
}

impl Error for DocumentGuidError {}
