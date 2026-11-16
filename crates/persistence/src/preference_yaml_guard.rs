// SPDX-License-Identifier: GPL-2.0-only
//! Reuse the document YAML safety check for preference writes.
use serde::Serialize;
use serde_yaml_ng::Value;

pub(crate) fn encode<T: Serialize>(source: &T) -> Result<String, String> {
    let encoded = serde_yaml_ng::to_string(source).map_err(|cause| cause.to_string())?;
    let value: Value = serde_yaml_ng::from_str(&encoded).map_err(|cause| cause.to_string())?;
    crate::package::yaml_write_guard::reject_ambiguous_merge_value(&value)
        .map_err(|cause| cause.to_string())?;
    Ok(encoded)
}

#[cfg(test)]
mod tests {
    use super::encode;
    use serde::Serialize;
    use std::collections::BTreeMap;

    #[derive(Serialize)]
    enum ExampleMode {
        Off,
        Custom { count: u32 },
    }

    #[test]
    fn valid_preferences_keep_original_serde_output() {
        for mode in [ExampleMode::Off, ExampleMode::Custom { count: 7 }] {
            let source = BTreeMap::from([("mode", mode)]);
            assert_eq!(
                encode(&source).unwrap(),
                serde_yaml_ng::to_string(&source).unwrap()
            );
        }
    }
}
