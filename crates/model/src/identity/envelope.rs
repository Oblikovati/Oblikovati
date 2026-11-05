// SPDX-License-Identifier: GPL-2.0-only

use std::{error::Error, fmt};

const MAGIC: &[u8; 4] = b"OKRK";
const TRIPLE_HEADER: usize = 16;
const CURRENT_HEADER: usize = 21;

/// Envelope provenance, distinct from interpretation of the opaque payload.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum ReferenceScheme {
    Legacy,
    Current,
    /// Rust semantic topology payload, distinct from Go lineage.
    Semantic,
}

/// Borrowed identity bytes. Decoding does not prove that a reference can bind.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct ReferenceEnvelope<'payload> {
    pub scheme: ReferenceScheme,
    pub context: u64,
    pub kind: u32,
    pub payload: &'payload [u8],
}

/// A malformed or unsupported persisted reference envelope.
#[derive(Clone, Debug, Eq, PartialEq)]
pub enum ReferenceEnvelopeError {
    ShortHeader { actual: usize, expected: usize },
    UnsupportedScheme(u8),
    PayloadLength { declared: u32, actual: usize },
    PayloadTooLarge(usize),
}

impl<'payload> ReferenceEnvelope<'payload> {
    /// Decode versioned or legacy bytes, selecting by the reserved magic prefix.
    /// A malformed versioned key is never retried as a legacy key.
    ///
    /// ```
    /// use oblikovati_model::identity::ReferenceEnvelope;
    /// let legacy = [0_u8; 16];
    /// assert_eq!(ReferenceEnvelope::decode(&legacy)?.context, 0);
    /// # Ok::<(), oblikovati_model::identity::ReferenceEnvelopeError>(())
    /// ```
    pub fn decode(encoded: &'payload [u8]) -> Result<Self, ReferenceEnvelopeError> {
        if !encoded.starts_with(MAGIC) {
            return Self::decode_legacy(encoded);
        }
        require_header(encoded, CURRENT_HEADER)?;
        let scheme = match encoded[4] {
            1 => ReferenceScheme::Current,
            2 => ReferenceScheme::Semantic,
            other => return Err(ReferenceEnvelopeError::UnsupportedScheme(other)),
        };
        decode_triple(&encoded[5..], scheme)
    }

    /// Decode bytes known by the storage owner to use the unversioned format.
    /// This explicit path handles legacy context bytes that start with `OKRK`.
    ///
    /// ```
    /// use oblikovati_model::identity::{ReferenceEnvelope, ReferenceScheme};
    /// let legacy = [0_u8; 16];
    /// assert_eq!(ReferenceEnvelope::decode_legacy(&legacy)?.scheme, ReferenceScheme::Legacy);
    /// # Ok::<(), oblikovati_model::identity::ReferenceEnvelopeError>(())
    /// ```
    pub fn decode_legacy(encoded: &'payload [u8]) -> Result<Self, ReferenceEnvelopeError> {
        decode_triple(encoded, ReferenceScheme::Legacy)
    }

    /// Upgrade legacy to scheme 1; preserve scheme 2 and every identity byte.
    /// This upgrades only the envelope, not a kernel's naming semantics.
    ///
    /// ```
    /// use oblikovati_model::identity::ReferenceEnvelope;
    /// let legacy = [0_u8; 16];
    /// let current = ReferenceEnvelope::decode_legacy(&legacy)?.encode_current()?;
    /// assert_eq!(&current[..5], b"OKRK\x01");
    /// # Ok::<(), oblikovati_model::identity::ReferenceEnvelopeError>(())
    /// ```
    pub fn encode_current(&self) -> Result<Vec<u8>, ReferenceEnvelopeError> {
        let length = encoded_payload_length(self.payload.len())?;
        let mut encoded = Vec::with_capacity(CURRENT_HEADER + self.payload.len());
        encoded.extend_from_slice(MAGIC);
        encoded.push(if self.scheme == ReferenceScheme::Semantic {
            2
        } else {
            1
        });
        encoded.extend_from_slice(&self.context.to_le_bytes());
        encoded.extend_from_slice(&self.kind.to_le_bytes());
        encoded.extend_from_slice(&length.to_le_bytes());
        encoded.extend_from_slice(self.payload);
        Ok(encoded)
    }
}

fn require_header(encoded: &[u8], expected: usize) -> Result<(), ReferenceEnvelopeError> {
    if encoded.len() < expected {
        return Err(ReferenceEnvelopeError::ShortHeader {
            actual: encoded.len(),
            expected,
        });
    }
    Ok(())
}

fn decode_triple(
    encoded: &[u8],
    scheme: ReferenceScheme,
) -> Result<ReferenceEnvelope<'_>, ReferenceEnvelopeError> {
    require_header(encoded, TRIPLE_HEADER)?;
    let context = u64::from_le_bytes(encoded[..8].try_into().expect("validated header"));
    let kind = u32::from_le_bytes(encoded[8..12].try_into().expect("validated header"));
    let declared = u32::from_le_bytes(encoded[12..16].try_into().expect("validated header"));
    let payload = &encoded[TRIPLE_HEADER..];
    if u64::from(declared) != payload.len() as u64 {
        return Err(ReferenceEnvelopeError::PayloadLength {
            declared,
            actual: payload.len(),
        });
    }
    Ok(ReferenceEnvelope {
        scheme,
        context,
        kind,
        payload,
    })
}

fn encoded_payload_length(length: usize) -> Result<u32, ReferenceEnvelopeError> {
    if length > isize::MAX as usize - CURRENT_HEADER {
        return Err(ReferenceEnvelopeError::PayloadTooLarge(length));
    }
    u32::try_from(length).map_err(|_| ReferenceEnvelopeError::PayloadTooLarge(length))
}

impl fmt::Display for ReferenceEnvelopeError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::ShortHeader { actual, expected } => write!(
                formatter,
                "reference header has {actual} bytes: expected at least {expected}"
            ),
            Self::UnsupportedScheme(scheme) => {
                write!(
                    formatter,
                    "reference scheme {scheme}: expected scheme 1 or 2"
                )
            }
            Self::PayloadLength { declared, actual } => write!(
                formatter,
                "reference payload declares {declared} bytes: expected {actual} trailing bytes"
            ),
            Self::PayloadTooLarge(length) => write!(
                formatter,
                "reference payload length {length}: expected u32 length plus a 21-byte header within isize::MAX"
            ),
        }
    }
}

impl Error for ReferenceEnvelopeError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn size_checks_refuse_unrepresentable_payloads_without_allocating_them() {
        assert_eq!(encoded_payload_length(0), Ok(0));
        assert_eq!(encoded_payload_length(1024), Ok(1024));
        assert!(encoded_payload_length(usize::MAX).is_err());
        if let Ok(too_large) = usize::try_from(u64::from(u32::MAX) + 1) {
            assert!(encoded_payload_length(too_large).is_err());
        }
    }
}
