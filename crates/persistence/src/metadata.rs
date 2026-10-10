// SPDX-License-Identifier: GPL-2.0-only

//! Legacy binary attribute/property values, independent of collection ownership.
mod reader;
use oblikovati_api::types::VariantValue;
use std::{error::Error, fmt};

/// A binary metadata value cannot be represented without loss.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct MetadataCodecError(String);
impl fmt::Display for MetadataCodecError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(output)
    }
}
impl Error for MetadataCodecError {}

/// Encode one Go-compatible value; example: `encode_value(&VariantValue::Integer(-1))?`.
pub fn encode_value(value: &VariantValue) -> Result<Vec<u8>, MetadataCodecError> {
    match value {
        VariantValue::Boolean(value) => Ok(vec![0, u8::from(*value)]),
        VariantValue::Integer(value) => Ok(fixed(1, value.to_le_bytes())),
        VariantValue::Double(value) if value.is_finite() => Ok(fixed(2, value.to_le_bytes())),
        VariantValue::String(value) => blob(3, value.as_bytes()),
        VariantValue::Bytes(value) => blob(4, value.0.as_deref().unwrap_or_default()),
        _ => Err(MetadataCodecError(format!(
            "metadata value {value:?}: expected a Boolean, integer, finite double, string, or bytes"
        ))),
    }
}

/// Decode one value and return the unread suffix for a collection decoder.
/// Example: `let (value, rest) = decode_value(&[0, 1])?;`.
pub fn decode_value(encoded: &[u8]) -> Result<(VariantValue, &[u8]), MetadataCodecError> {
    let mut reader = reader::ValueReader::new(encoded);
    let value = reader.value()?;
    Ok((value, reader.remaining()))
}

fn fixed(tag: u8, bytes: [u8; 8]) -> Vec<u8> {
    let mut encoded = Vec::with_capacity(9);
    encoded.push(tag);
    encoded.extend_from_slice(&bytes);
    encoded
}

fn blob(tag: u8, bytes: &[u8]) -> Result<Vec<u8>, MetadataCodecError> {
    let length = blob_length(bytes.len())?;
    let mut encoded = vec![tag];
    encoded.extend_from_slice(&length.to_le_bytes());
    encoded.extend_from_slice(bytes);
    Ok(encoded)
}

fn blob_length(length: usize) -> Result<u32, MetadataCodecError> {
    u32::try_from(length).map_err(|_| {
        MetadataCodecError(format!(
            "metadata payload length {length}: expected at most {} bytes",
            u32::MAX
        ))
    })
}

#[cfg(test)]
mod tests {
    #[test]
    fn blob_length_checks_the_persisted_integer_range() {
        assert_eq!(super::blob_length(u32::MAX as usize).unwrap(), u32::MAX);
        if let Some(too_large) = (u32::MAX as usize).checked_add(1) {
            assert!(super::blob_length(too_large).is_err());
        }
    }
}
