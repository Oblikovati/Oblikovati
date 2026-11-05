// SPDX-License-Identifier: GPL-2.0-only
use super::project::read_project_assets;
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
    package::assets::*,
};
use oblikovati_api::types::ASSET_PROJECT;
use oblikovati_model::assets::ProjectAssetLibrary;
use serde_yaml_ng::{Mapping, Value};
use std::path::Path;

/// The single independently stored library changed by an edit.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ProjectAssetFamily {
    Material,
    Appearance,
}
impl ProjectAssetFamily {
    fn field(self) -> &'static str {
        match self {
            Self::Material => "materials",
            Self::Appearance => "appearances",
        }
    }
    fn other(self) -> Self {
        match self {
            Self::Material => Self::Appearance,
            Self::Appearance => Self::Material,
        }
    }
}

/// Persist one complete family before its owner adopts it; example: `persist_project_asset_family(dir, family, &before, &next, files)?`.
pub fn persist_project_asset_family<F: PackageFileReader + AtomicFiles>(
    directory: &Path,
    family: ProjectAssetFamily,
    before: &ProjectAssetLibrary,
    after: &ProjectAssetLibrary,
    files: &mut F,
) -> Result<(), String> {
    let path = directory.join(format!("{}.yaml", family.field()));
    let contents = prepare_project_asset_family(&path, family, before, after, files)
        .map_err(|cause| format!("project asset file {path:?}: {cause}"))?;
    replace_file(&path, &contents, files).map_err(|cause| cause.to_string())
}

fn prepare_project_asset_family(
    path: &Path,
    family: ProjectAssetFamily,
    before: &ProjectAssetLibrary,
    after: &ProjectAssetLibrary,
    files: &mut dyn PackageFileReader,
) -> Result<Vec<u8>, String> {
    require_single_family_edit(family, before, after)?;
    let original = read_family_records(path, family, files)?;
    let known = family_records(before, family)?;
    let restored = restore_family(&original, family)?;
    if family_records(&restored, family)? != known {
        return Err("library changed on disk: expected explicit reload before an edit".into());
    }
    let replacement = family_records(after, family)?;
    let merged = merge_asset_entries(&original, &known, &replacement);
    if family_records(&restore_family(&merged, family)?, family)? != replacement {
        return Err("library export: expected an exact typed asset round trip".into());
    }
    encode_family(family, merged)
}

fn require_single_family_edit(
    family: ProjectAssetFamily,
    before: &ProjectAssetLibrary,
    after: &ProjectAssetLibrary,
) -> Result<(), String> {
    if family_records(before, family.other())? != family_records(after, family.other())? {
        return Err(format!(
            "{family:?} edit: expected the other asset family to remain unchanged"
        ));
    }
    Ok(())
}

fn read_family_records(
    path: &Path,
    family: ProjectAssetFamily,
    files: &mut dyn PackageFileReader,
) -> Result<Value, String> {
    let file = read_project_assets(path, files)?;
    let (selected, other) = match family {
        ProjectAssetFamily::Material => (file.materials, file.appearances),
        ProjectAssetFamily::Appearance => (file.appearances, file.materials),
    };
    if !other.is_empty() {
        return Err(format!(
            "{path:?}: expected only {} records",
            family.field()
        ));
    }
    Ok(Value::Sequence(selected))
}

fn family_records(
    library: &ProjectAssetLibrary,
    family: ProjectAssetFamily,
) -> Result<Value, String> {
    let records = match family {
        ProjectAssetFamily::Material => library
            .materials()
            .ordered()
            .map(|asset| stored_record(&asset.info(), true))
            .collect::<Result<Vec<_>, _>>()?,
        ProjectAssetFamily::Appearance => library
            .appearances()
            .ordered()
            .map(|asset| stored_record(&asset.info(), false))
            .collect::<Result<Vec<_>, _>>()?,
    };
    Ok(Value::Sequence(records))
}

fn restore_family(
    records: &Value,
    family: ProjectAssetFamily,
) -> Result<ProjectAssetLibrary, String> {
    let entries = records
        .as_sequence()
        .ok_or_else(|| format!("library {records:?}: expected an asset sequence"))?;
    match family {
        ProjectAssetFamily::Material => ProjectAssetLibrary::from_records(
            decode_material_definitions(entries, ASSET_PROJECT)?,
            vec![],
        ),
        ProjectAssetFamily::Appearance => ProjectAssetLibrary::from_records(
            vec![],
            decode_appearance_definitions(entries, ASSET_PROJECT)?,
        ),
    }
}

fn encode_family(family: ProjectAssetFamily, records: Value) -> Result<Vec<u8>, String> {
    let mut root = Mapping::new();
    root.insert(family.field().into(), records);
    serde_yaml_ng::to_string(&root)
        .map(String::into_bytes)
        .map_err(|cause| cause.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::wire::MaterialInfo;
    use oblikovati_model::assets::MaterialAsset;
    #[test]
    fn independent_family_edit_refuses_a_second_family_before_io() {
        let before = ProjectAssetLibrary::default();
        let asset = MaterialAsset::from_info(MaterialInfo {
            id: "second-family".into(),
            source: "project".into(),
            ..Default::default()
        })
        .unwrap();
        let after = before.with_material(asset).unwrap();
        assert!(
            require_single_family_edit(ProjectAssetFamily::Appearance, &before, &after)
                .unwrap_err()
                .contains("other asset family")
        );
    }
}
