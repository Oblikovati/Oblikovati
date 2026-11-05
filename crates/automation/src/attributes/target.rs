// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::Error;
use oblikovati_model::{
    attribute::AttributeTarget,
    identity::{ReferenceEnvelope, ReferenceScheme},
};

pub(super) fn key(target: &str) -> Result<AttributeTarget, Error> {
    // Persisted Go model/identity/entity.go IDs at aa8297d1: document=20, external=21.
    let envelope = ReferenceEnvelope {
        scheme: ReferenceScheme::Current,
        context: 0,
        kind: if target.is_empty() { 20 } else { 21 },
        payload: target.as_bytes(),
    };
    let bytes = envelope
        .encode_current()
        .map_err(|error| Error(error.to_string()))?;
    AttributeTarget::from_encoded(&bytes).map_err(|error| Error(error.to_string()))
}

pub(super) fn name(target: &AttributeTarget) -> Result<String, Error> {
    let envelope =
        ReferenceEnvelope::decode(target.encoded()).map_err(|error| Error(error.to_string()))?;
    if envelope.kind != 21 {
        return Ok(String::new());
    }
    String::from_utf8(envelope.payload.to_vec()).map_err(|error| {
        Error(format!(
            "external attribute target {:?}: expected UTF-8: {error}",
            envelope.payload
        ))
    })
}
