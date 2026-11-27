// SPDX-License-Identifier: GPL-2.0-only

use super::DocumentGuid;
use std::{error::Error, fmt};

/// Supplies fresh GUIDs; callers inject this boundary for repeatable save tests.
pub trait GuidSource {
    /// Generate one GUID, reporting failure without substituting an identity.
    ///
    /// ```
    /// use oblikovati_model::identity::{GuidSource, SystemGuidSource};
    /// let guid = SystemGuidSource.generate()?;
    /// assert_eq!(guid.to_string().len(), 36);
    /// # Ok::<(), oblikovati_model::identity::IdentityTransitionError>(())
    /// ```
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError>;
}

/// Operating-system randomness behind the model-owned GUID source interface.
pub struct SystemGuidSource;

impl GuidSource for SystemGuidSource {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        let mut bytes = [0_u8; 16];
        getrandom::fill(&mut bytes)
            .map_err(|error| IdentityTransitionError::Randomness(error.to_string()))?;
        Ok(DocumentGuid(
            uuid::Builder::from_random_bytes(bytes).into_uuid(),
        ))
    }
}

/// A refused save identity transition; the original identity is unchanged.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum IdentityTransitionError {
    Randomness(String),
    RepeatedGuid(DocumentGuid),
    CounterOverflow(u64),
}

impl fmt::Display for IdentityTransitionError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Randomness(reason) => write!(
                formatter,
                "GUID entropy {reason:?}: expected 16 random bytes"
            ),
            Self::RepeatedGuid(guid) => write!(
                formatter,
                "generated GUID {guid}: expected a fresh identity"
            ),
            Self::CounterOverflow(count) => write!(
                formatter,
                "save counter {count}: expected a value below {}",
                u64::MAX
            ),
        }
    }
}

impl Error for IdentityTransitionError {}

pub(super) struct GuidIssuer<'source> {
    source: &'source mut dyn GuidSource,
    excluded: Vec<DocumentGuid>,
}

impl<'source> GuidIssuer<'source> {
    pub(super) fn new(source: &'source mut dyn GuidSource, excluded: Vec<DocumentGuid>) -> Self {
        Self { source, excluded }
    }

    pub(super) fn issue(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        let guid = self.source.generate()?;
        if self.excluded.contains(&guid) {
            return Err(IdentityTransitionError::RepeatedGuid(guid));
        }
        self.excluded.push(guid);
        Ok(guid)
    }
}
