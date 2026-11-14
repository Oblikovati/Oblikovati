// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(super) fn restore_sets(
    package: &DocumentPackage,
    properties: &mut DocumentProperties,
) -> Result<bool, PackageCodecError> {
    let Some(section) = package
        .root
        .get("model")
        .and_then(|model| model.get("customPropertySets"))
    else {
        return Ok(false);
    };
    let names = decode_names(section, package.manifest.schema_version)?;
    insert_names(properties, names)?;
    Ok(true)
}

fn decode_names(section: &Value, schema: u32) -> Result<Vec<String>, PackageCodecError> {
    if schema < 3 {
        return Err(PackageCodecError(format!(
            "customPropertySets in schema {}: expected schema 3",
            schema
        )));
    }
    if !section.is_sequence() {
        return Err(PackageCodecError(format!(
            "customPropertySets {section:?}: expected a sequence of custom set names"
        )));
    }
    serde_yaml_ng::from_value(section.clone()).map_err(codec_error)
}

fn insert_names(
    properties: &mut DocumentProperties,
    names: Vec<String>,
) -> Result<(), PackageCodecError> {
    let mut unique = BTreeSet::new();
    for name in names {
        if name.is_empty() || is_standard(&name) || !unique.insert(name.clone()) {
            return Err(PackageCodecError(format!(
                "customPropertySets entry {name:?}: expected unique nonempty custom names"
            )));
        }
        properties
            .ensure_set(&name)
            .map_err(|error| PackageCodecError(error.to_string()))?;
    }
    Ok(())
}

pub(super) fn capture_sections(
    package: &mut DocumentPackage,
    properties: &DocumentProperties,
    exposure: bool,
) -> Result<(), PackageCodecError> {
    let records = capture_properties(properties, exposure)?;
    let mut model = package
        .root
        .get("model")
        .and_then(Value::as_mapping)
        .cloned()
        .unwrap_or_default();
    model.insert(
        "properties".into(),
        serde_yaml_ng::to_value(records).map_err(codec_error)?,
    );
    capture_names(package, properties, &mut model)?;
    package.root.insert("model".into(), Value::Mapping(model));
    Ok(())
}

fn capture_names(
    package: &mut DocumentPackage,
    properties: &DocumentProperties,
    model: &mut serde_yaml_ng::Mapping,
) -> Result<(), PackageCodecError> {
    let custom: Vec<_> = properties
        .sets()
        .iter()
        .filter(|set| !is_standard(&set.name))
        .collect();
    let extended = custom.iter().any(|set| set.properties().is_empty());
    model.remove("customPropertySets");
    if !extended {
        return Ok(());
    }
    let names: Vec<_> = custom.iter().map(|set| &set.name).collect();
    model.insert(
        "customPropertySets".into(),
        serde_yaml_ng::to_value(names).map_err(codec_error)?,
    );
    package.manifest.schema_version = 3;
    package.root.insert("schemaVersion".into(), 3u32.into());
    Ok(())
}

fn is_standard(name: &str) -> bool {
    [
        SUMMARY_INFORMATION,
        DOCUMENT_SUMMARY,
        DESIGN_TRACKING,
        USER_DEFINED,
    ]
    .contains(&name)
}
