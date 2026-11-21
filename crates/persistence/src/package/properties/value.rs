// SPDX-License-Identifier: GPL-2.0-only

use super::{PackageCodecError, PropertyRecipe};
use base64::{Engine, engine::general_purpose::STANDARD};
use oblikovati_api::{json::Bytes, types::VariantValue};

pub(super) fn decode(record: &PropertyRecipe) -> Result<VariantValue, PackageCodecError> {
    let value = &record.value;
    let decoded = match record.kind.as_str() {
        "boolean" => parse_boolean(value).map(VariantValue::Boolean),
        "integer" => value.parse().ok().map(VariantValue::Integer),
        "double" => value
            .parse::<f64>()
            .ok()
            .filter(|value| value.is_finite())
            .map(VariantValue::Double),
        "string" => Some(VariantValue::String(value.clone())),
        "bytes" => return decode_bytes(record),
        _ => None,
    };
    decoded.ok_or_else(|| PackageCodecError(format!("property {:?}/{:?} type {:?}, value {value:?}: expected a valid boolean, integer, finite double, string, or bytes record", record.set, record.name, record.kind)))
}

fn decode_bytes(record: &PropertyRecipe) -> Result<VariantValue, PackageCodecError> {
    let owner = format!("property {:?}/{:?}", record.set, record.name);
    let bytes = super::super::binary_text::decode_base64(&owner, &record.value)?;
    Ok(VariantValue::Bytes(Bytes(
        (!bytes.is_empty()).then_some(bytes),
    )))
}

fn parse_boolean(value: &str) -> Option<bool> {
    match value {
        "1" | "t" | "T" | "TRUE" | "true" | "True" => Some(true),
        "0" | "f" | "F" | "FALSE" | "false" | "False" => Some(false),
        _ => None,
    }
}

pub(super) fn encode(value: &VariantValue) -> Result<(&'static str, String), PackageCodecError> {
    match value {
        VariantValue::Boolean(value) => Ok(("boolean", value.to_string())),
        VariantValue::Integer(value) => Ok(("integer", value.to_string())),
        VariantValue::Double(value) if value.is_finite() => Ok(("double", value.to_string())),
        VariantValue::String(value) => Ok(("string", value.clone())),
        VariantValue::Bytes(value) => Ok((
            "bytes",
            STANDARD.encode(value.0.as_deref().unwrap_or_default()),
        )),
        value => Err(PackageCodecError(format!(
            "property value {value:?}: expected an initialized finite API value"
        ))),
    }
}
