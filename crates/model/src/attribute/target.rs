// SPDX-License-Identifier: GPL-2.0-only

use crate::identity::{ReferenceEnvelope, ReferenceEnvelopeError};

/// Canonical reference identity; validation does not prove that geometry can bind.
#[derive(Clone, Debug, Eq, Ord, PartialEq, PartialOrd)]
pub struct AttributeTarget(Vec<u8>);

impl AttributeTarget {
    /// Normalize an encoded reference envelope, retaining context, kind, and payload.
    /// Example: `let target = AttributeTarget::from_encoded(&[0; 16])?;`.
    pub fn from_encoded(encoded: &[u8]) -> Result<Self, ReferenceEnvelopeError> {
        // Go RefKey.Encode writes the current envelope regardless of decoded scheme (#1157).
        ReferenceEnvelope::decode(encoded)?
            .encode_current()
            .map(Self)
    }

    /// Inspect canonical reference bytes for storage; example: `target.encoded()`.
    pub fn encoded(&self) -> &[u8] {
        &self.0
    }
}
