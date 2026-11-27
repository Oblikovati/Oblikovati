// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::VariantValue;

pub(crate) fn validate(value: &VariantValue) -> Result<(), &'static str> {
    if matches!(value, VariantValue::Uninitialized)
        || matches!(value, VariantValue::Double(number) if !number.is_finite())
    {
        return Err("a Boolean, integer, finite number, string, or byte array");
    }
    Ok(())
}

pub(crate) fn normalize(value: VariantValue) -> VariantValue {
    // Go BytesValue copies onto a nil slice; empty input therefore has one canonical form.
    match value {
        VariantValue::Bytes(bytes) if bytes.is_empty() => VariantValue::Bytes(Default::default()),
        value => value,
    }
}

pub(crate) fn same(left: &VariantValue, right: &VariantValue) -> bool {
    match (left, right) {
        (VariantValue::Double(left), VariantValue::Double(right)) => {
            left.to_bits() == right.to_bits()
        }
        _ => left == right,
    }
}
