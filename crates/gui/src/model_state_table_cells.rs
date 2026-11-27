// SPDX-License-Identifier: GPL-2.0-only
//! Parse typed property cells in the model-state table.
use oblikovati_api::types::{Variant, VariantValue};

pub(crate) fn parse_cell(original: Option<&Variant>, text: &str) -> Option<Variant> {
    let value = match original.map(|value| &value.value) {
        None | Some(VariantValue::String(_)) => VariantValue::String(text.into()),
        Some(VariantValue::Integer(_)) => VariantValue::Integer(text.parse().ok()?),
        Some(VariantValue::Double(_)) => {
            let value: f64 = text.parse().ok()?;
            if !value.is_finite() {
                return None;
            }
            VariantValue::Double(value)
        }
        Some(VariantValue::Boolean(_)) => VariantValue::Boolean(text.parse().ok()?),
        _ => return None,
    };
    Some(Variant {
        value,
        unit: String::new(),
    })
}
pub(crate) fn property_text(value: &Variant) -> String {
    match &value.value {
        VariantValue::String(text) => text.clone(),
        VariantValue::Integer(value) => value.to_string(),
        VariantValue::Double(value) => value.to_string(),
        VariantValue::Boolean(value) => value.to_string(),
        VariantValue::Bytes(_) => "<binary>".into(),
        VariantValue::Uninitialized => String::new(),
    }
}
