// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::Error;
use serde::{Serialize, de::DeserializeOwned};
/// Decode canonical object/null arguments; example: `decode_wire_arguments::<wire::AddInRefArgs>(method, bytes)?;`.
pub fn decode_wire_arguments<T: DeserializeOwned + Default>(
    method: &str,
    request: &[u8],
) -> Result<T, Error> {
    serde_json::from_slice::<serde_json::Value>(request).and_then(|value| {
        if !value.is_object() && !value.is_null() {
            return Err(serde::de::Error::custom("expected a JSON object or null"));
        }
        serde_json::from_value::<Option<T>>(value).map(Option::unwrap_or_default)
    }).map_err(|error| {
        Error(format!("method {method:?} request {:?}: expected its canonical JSON argument object: {error}", String::from_utf8_lossy(request)))
    })
}

/// Serialize a canonical reply; example: `encode_wire_reply(&wire::OKResult::default())?;`.
pub fn encode_wire_reply<T: Serialize>(response: &T) -> Result<Vec<u8>, Error> {
    serde_json::to_vec(response).map_err(|error| {
        Error(format!(
            "query response: expected canonical serializable JSON: {error}"
        ))
    })
}
