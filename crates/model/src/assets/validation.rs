// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{ASSET_BUILTIN, ASSET_DOCUMENT, ASSET_PROJECT, AssetSource};
use serde::Serialize;
pub(super) fn validate_asset_record<T: Serialize>(
    id: &str,
    source: &str,
    record: &T,
) -> Result<(), String> {
    validate_asset_identity(id)?;
    validate_asset_origin(id, source)?;
    serde_json::to_vec(record)
        .map(|_| ())
        .map_err(|cause| format!("asset {id:?}: expected finite API property values: {cause}"))
}
pub(super) fn editable_asset_source(source: &AssetSource) -> bool {
    source == &ASSET_DOCUMENT || source == &ASSET_PROJECT
}

fn validate_asset_identity(id: &str) -> Result<(), String> {
    if id.is_empty() || id.contains('\0') {
        return Err(format!(
            "asset identity {id:?}: expected nonempty text without NUL"
        ));
    }
    Ok(())
}

fn validate_asset_origin(id: &str, source: &str) -> Result<(), String> {
    if ![
        ASSET_BUILTIN.0.as_ref(),
        ASSET_PROJECT.0.as_ref(),
        ASSET_DOCUMENT.0.as_ref(),
    ]
    .contains(&source)
    {
        return Err(format!(
            "asset {id:?} source {source:?}: expected builtin, project, or document"
        ));
    }
    Ok(())
}
