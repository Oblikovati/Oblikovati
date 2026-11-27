// SPDX-License-Identifier: GPL-2.0-only
use crate::{assets::with_scoped_assets, decode, encode};
use oblikovati_api::{
    Error,
    contract::{Appearance, Material},
    types::*,
    wire::*,
};
use oblikovati_model::assets::{AppearanceAsset, MaterialAsset, ScopedAssets};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
mod prepare;

/// Asset edits with explicit project persistence or document history ownership.
pub const ASSET_COMMAND_METHODS: [&str; 4] = [
    METHOD_MATERIALS_CREATE,
    METHOD_MATERIALS_UPDATE,
    METHOD_APPEARANCES_CREATE,
    METHOD_APPEARANCES_UPDATE,
];
pub(super) enum AssetChange {
    Material(MaterialAsset),
    Appearance(AppearanceAsset),
}
impl AssetChange {
    fn response(&self) -> Result<Vec<u8>, Error> {
        match self {
            Self::Material(asset) => crate::assets::material_response(asset),
            Self::Appearance(asset) => encode(&asset.info()),
        }
    }
    fn source(&self) -> AssetSource {
        match self {
            Self::Material(asset) => asset.source(),
            Self::Appearance(asset) => asset.source(),
        }
    }
}
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let change = with_scoped_assets(files, |materials, appearances| {
        prepare::command(materials, appearances, method, request)
    })?;
    let response = change.response()?;
    match change.source() {
        source if source == ASSET_BUILTIN => (),
        source if source == ASSET_PROJECT => persist_project(files, change)?,
        source if source == ASSET_DOCUMENT => edit_document(files, change)?,
        source => {
            return Err(Error(format!(
                "asset source {source:?}: expected builtin, project or document"
            )));
        }
    }
    Ok(response)
}
fn persist_project<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    change: AssetChange,
) -> Result<(), Error> {
    match change {
        AssetChange::Material(asset) => files.store_project_material(asset),
        AssetChange::Appearance(asset) => files.store_project_appearance(asset),
    }
    .map_err(|cause| Error(cause.to_string()))
}
fn edit_document<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    change: AssetChange,
) -> Result<(), Error> {
    let id = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("asset edit: expected an active document".into()))?;
    files
        .commands(id)
        .map_err(|cause| Error(cause.to_string()))?
        .run("Edit document asset", |snapshot, _| {
            match &change {
                AssetChange::Material(asset) => snapshot.assets.embed_material(asset)?,
                AssetChange::Appearance(asset) => snapshot.assets.embed_appearance(asset)?,
            };
            Ok::<(), String>(())
        })
        .map_err(|cause| Error(cause.to_string()))
}
