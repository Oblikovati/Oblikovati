// SPDX-License-Identifier: GPL-2.0-only
//! Convert the iProperties editor's typed text to API values.
use oblikovati_api::{json::Bytes, types::VariantValue};

pub(super) const TYPES: [&str; 5] = ["Text", "Integer", "Number", "Boolean", "Bytes (hex)"];

pub(super) fn parse_value(kind: usize, input: &str) -> Result<VariantValue, &'static str> {
    match kind {
        0 => Ok(VariantValue::String(input.into())),
        1 => input
            .trim()
            .parse::<i64>()
            .map(VariantValue::Integer)
            .map_err(|_| "Enter a whole number within the 64-bit range."),
        2 => parse_number(input),
        3 => match input.trim() {
            "true" => Ok(VariantValue::Boolean(true)),
            "false" => Ok(VariantValue::Boolean(false)),
            _ => Err("Enter true or false."),
        },
        4 => parse_hex(input),
        _ => Err("Select a supported property type."),
    }
}

fn parse_number(input: &str) -> Result<VariantValue, &'static str> {
    let value: f64 = input.trim().parse().map_err(|_| "Enter a finite number.")?;
    if !value.is_finite() {
        return Err("Enter a finite number.");
    }
    Ok(VariantValue::Double(value))
}

fn parse_hex(input: &str) -> Result<VariantValue, &'static str> {
    let bytes = input.trim().as_bytes();
    if !bytes.len().is_multiple_of(2) || !bytes.iter().all(u8::is_ascii_hexdigit) {
        return Err("Enter hexadecimal bytes as pairs of digits, without spaces.");
    }
    let values = bytes
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| {
            let digits = std::str::from_utf8(pair).expect("validated ASCII hex");
            u8::from_str_radix(digits, 16).expect("validated hex digits")
        })
        .collect();
    Ok(VariantValue::Bytes(Bytes(Some(values))))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn parses_each_supported_type_without_coercion() {
        assert_eq!(parse_value(0, "12"), Ok(VariantValue::String("12".into())));
        assert_eq!(parse_value(1, " -12 "), Ok(VariantValue::Integer(-12)));
        assert_eq!(parse_value(2, "1.25"), Ok(VariantValue::Double(1.25)));
        assert_eq!(parse_value(3, "false"), Ok(VariantValue::Boolean(false)));
        assert_eq!(
            parse_value(4, "00aF"),
            Ok(VariantValue::Bytes(Bytes(Some(vec![0, 175]))))
        );
        assert_eq!(
            parse_value(4, ""),
            Ok(VariantValue::Bytes(Bytes(Some(vec![]))))
        );
    }
    #[test]
    fn invalid_values_and_unknown_types_are_refused() {
        for (kind, value) in [
            (1, "9223372036854775808"),
            (2, "NaN"),
            (2, "inf"),
            (3, "yes"),
            (4, "abc"),
            (4, "g0"),
            (4, "aa bb"),
            (5, "x"),
        ] {
            assert!(
                parse_value(kind, value).is_err(),
                "kind={kind} value={value:?}"
            );
        }
    }
}
