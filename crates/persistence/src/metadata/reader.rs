// SPDX-License-Identifier: GPL-2.0-only

use super::{MetadataCodecError, VariantValue};

pub(super) struct ValueReader<'bytes> {
    remaining: &'bytes [u8],
}

impl<'bytes> ValueReader<'bytes> {
    pub(super) fn new(encoded: &'bytes [u8]) -> Self {
        Self { remaining: encoded }
    }

    pub(super) fn remaining(&self) -> &'bytes [u8] {
        self.remaining
    }

    pub(super) fn value(&mut self) -> Result<VariantValue, MetadataCodecError> {
        match self.take(1)?[0] {
            0 => self.boolean(),
            1 => Ok(VariantValue::Integer(i64::from_le_bytes(self.fixed()?))),
            2 => self.double(),
            3 => self.string(),
            4 => self.bytes(),
            tag => Err(MetadataCodecError(format!(
                "metadata type code {tag}: expected 0..=4"
            ))),
        }
    }

    fn bytes(&mut self) -> Result<VariantValue, MetadataCodecError> {
        let bytes = self.blob()?.to_vec();
        Ok(VariantValue::Bytes(oblikovati_api::json::Bytes(
            (!bytes.is_empty()).then_some(bytes),
        )))
    }

    fn boolean(&mut self) -> Result<VariantValue, MetadataCodecError> {
        match self.take(1)?[0] {
            0 => Ok(VariantValue::Boolean(false)),
            1 => Ok(VariantValue::Boolean(true)),
            value => Err(MetadataCodecError(format!(
                "metadata Boolean byte {value}: expected 0 or 1"
            ))),
        }
    }

    fn double(&mut self) -> Result<VariantValue, MetadataCodecError> {
        let value = f64::from_le_bytes(self.fixed()?);
        if !value.is_finite() {
            return Err(MetadataCodecError(format!(
                "metadata double {value}: expected a finite value"
            )));
        }
        Ok(VariantValue::Double(value))
    }

    fn string(&mut self) -> Result<VariantValue, MetadataCodecError> {
        let bytes = self.blob()?;
        let text = std::str::from_utf8(bytes).map_err(|error| {
            MetadataCodecError(format!(
                "metadata string at byte {}: expected valid UTF-8 ({error})",
                error.valid_up_to()
            ))
        })?;
        Ok(VariantValue::String(text.into()))
    }

    fn blob(&mut self) -> Result<&'bytes [u8], MetadataCodecError> {
        let length = u32::from_le_bytes(self.fixed()?) as usize;
        self.take(length)
    }

    fn fixed<const COUNT: usize>(&mut self) -> Result<[u8; COUNT], MetadataCodecError> {
        Ok(self
            .take(COUNT)?
            .try_into()
            .expect("checked metadata length"))
    }

    fn take(&mut self, count: usize) -> Result<&'bytes [u8], MetadataCodecError> {
        if count > self.remaining.len() {
            return Err(MetadataCodecError(format!(
                "metadata payload has {} bytes: expected at least {count}",
                self.remaining.len()
            )));
        }
        let (prefix, remaining) = self.remaining.split_at(count);
        self.remaining = remaining;
        Ok(prefix)
    }
}
