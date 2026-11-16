// SPDX-License-Identifier: GPL-2.0-only
//! Refuse YAML keys whose original merge semantics were lost during decoding.
use super::{Mapping, PackageCodecError, Value};

pub(crate) fn reject_ambiguous_merge_keys(root: &Mapping) -> Result<(), PackageCodecError> {
    let mut pending = Vec::new();
    collect_mapping(root, &mut pending)?;
    validate_pending(pending)
}

pub(crate) fn reject_ambiguous_merge_value(value: &Value) -> Result<(), PackageCodecError> {
    validate_pending(vec![value])
}

fn validate_pending(mut pending: Vec<&Value>) -> Result<(), PackageCodecError> {
    while let Some(value) = pending.pop() {
        collect_children(value, &mut pending)?;
    }
    Ok(())
}

fn collect_children<'a>(
    value: &'a Value,
    pending: &mut Vec<&'a Value>,
) -> Result<(), PackageCodecError> {
    match value {
        Value::Mapping(mapping) => collect_mapping(mapping, pending),
        Value::Sequence(sequence) => {
            pending.extend(sequence);
            Ok(())
        }
        Value::Tagged(tagged) => {
            pending.push(&tagged.value);
            Ok(())
        }
        _ => Ok(()),
    }
}

fn collect_mapping<'a>(
    mapping: &'a Mapping,
    pending: &mut Vec<&'a Value>,
) -> Result<(), PackageCodecError> {
    for (key, value) in mapping {
        if is_ambiguous_key(key) {
            return Err(PackageCodecError(
                "YAML mapping key <<: expected a key with unambiguous meaning (#4006)".into(),
            ));
        }
        pending.extend([key, value]);
    }
    Ok(())
}

fn is_ambiguous_key(key: &Value) -> bool {
    match key {
        Value::String(value) => value == "<<",
        Value::Tagged(tagged) => is_ambiguous_key(&tagged.value),
        _ => false,
    }
}
