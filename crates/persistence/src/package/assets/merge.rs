// SPDX-License-Identifier: GPL-2.0-only
use serde_yaml_ng::Value;
/// Overlay owned fields while retaining opaque extensions, and remove only fields known to the old typed state.
pub(super) fn merge_known_values(original: &Value, before: &Value, after: &Value) -> Value {
    let (Some(original), Some(before), Some(after)) = (
        original.as_mapping(),
        before.as_mapping(),
        after.as_mapping(),
    ) else {
        return after.clone();
    };
    let mut merged = original.clone();
    remove_old_owned_fields(&mut merged, original, before, after);
    for (key, replacement) in after {
        let value = match (original.get(key), before.get(key)) {
            (Some(original), Some(before)) => merge_known_values(original, before, replacement),
            _ => replacement.clone(),
        };
        merged.insert(key.clone(), value);
    }
    Value::Mapping(merged)
}
pub(super) fn merge_asset_entries(original: &Value, before: &Value, after: &Value) -> Value {
    let Some(after) = after.as_sequence() else {
        return after.clone();
    };
    Value::Sequence(
        after
            .iter()
            .map(|record| {
                let identity = record.get("id");
                match (
                    find_record(original, identity),
                    find_record(before, identity),
                ) {
                    (Some(original), Some(before)) => {
                        merge_migrated_entry(original, before, record)
                    }
                    _ => record.clone(),
                }
            })
            .collect(),
    )
}
fn find_record<'a>(records: &'a Value, identity: Option<&Value>) -> Option<&'a Value> {
    records
        .as_sequence()?
        .iter()
        .find(|record| record.get("id") == identity)
}
pub(super) fn merge_asset_recipe(original: &Value, before: &Value, after: &Value) -> Value {
    let mut result = merge_known_values(original, before, after);
    for name in ["materials", "appearances"] {
        if let (Some(original), Some(before), Some(after)) =
            (original.get(name), before.get(name), after.get(name))
            && let Some(fields) = result.as_mapping_mut()
        {
            fields.insert(name.into(), merge_asset_entries(original, before, after));
        }
    }
    result
}

fn remove_old_owned_fields(
    merged: &mut serde_yaml_ng::Mapping,
    original: &serde_yaml_ng::Mapping,
    before: &serde_yaml_ng::Mapping,
    after: &serde_yaml_ng::Mapping,
) {
    for key in before.keys().filter(|key| !after.contains_key(*key)) {
        if let (Some(raw), Some(known)) = (original.get(key), before.get(key)) {
            let retained = merge_known_values(raw, known, &Value::Mapping(Default::default()));
            if retained
                .as_mapping()
                .is_some_and(|fields| !fields.is_empty())
            {
                merged.insert(key.clone(), retained);
                continue;
            }
        }
        merged.remove(key);
    }
}

fn merge_migrated_entry(original: &Value, before: &Value, after: &Value) -> Value {
    let mut original = original.clone();
    if original.get("albedo").is_some()
        && let Some(fields) = original.as_mapping_mut()
    {
        for field in ["albedo", "metallic", "roughness", "emissive", "opacity"] {
            fields.remove(field);
        }
    }
    merge_known_values(&original, before, after)
}
