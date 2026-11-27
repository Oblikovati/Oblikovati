// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assets::AssetIdentity;

pub(super) fn command(
    materials: &ScopedAssets<'_, MaterialAsset>,
    appearances: &ScopedAssets<'_, AppearanceAsset>,
    method: &str,
    request: &[u8],
) -> Result<AssetChange, Error> {
    match method {
        METHOD_MATERIALS_CREATE => create_material(materials, decode(method, request)?),
        METHOD_APPEARANCES_CREATE => create_appearance(appearances, decode(method, request)?),
        METHOD_MATERIALS_UPDATE => update_material(materials, decode(method, request)?),
        METHOD_APPEARANCES_UPDATE => update_appearance(appearances, decode(method, request)?),
        _ => Err(Error(format!(
            "asset command {method:?}: expected one of {ASSET_COMMAND_METHODS:?}"
        ))),
    }
}
fn create_material(
    scope: &ScopedAssets<'_, MaterialAsset>,
    request: DuplicateAssetArgs,
) -> Result<AssetChange, Error> {
    let base = scope.resolve(&request.base_id).map_err(Error)?;
    let identity = fresh_asset_identity(scope, &request.name);
    base.duplicate(&identity, &request.name, ASSET_PROJECT)
        .map(AssetChange::Material)
        .map_err(Error)
}
fn create_appearance(
    scope: &ScopedAssets<'_, AppearanceAsset>,
    request: CreateAppearanceArgs,
) -> Result<AssetChange, Error> {
    let base = scope.resolve(&request.base_id).map_err(Error)?;
    let identity = fresh_asset_identity(scope, &request.name);
    base.duplicate(&identity, &request.name, ASSET_PROJECT)
        .map(AssetChange::Appearance)
        .map_err(Error)
}
fn update_material(
    scope: &ScopedAssets<'_, MaterialAsset>,
    request: MaterialInfo,
) -> Result<AssetChange, Error> {
    let mut candidate = scope.resolve(&request.id).map_err(Error)?.clone();
    candidate.edit(request).map_err(Error)?;
    Ok(AssetChange::Material(candidate))
}
fn update_appearance(
    scope: &ScopedAssets<'_, AppearanceAsset>,
    request: UpdateAppearanceArgs,
) -> Result<AssetChange, Error> {
    let mut candidate = scope.resolve(&request.id).map_err(Error)?.clone();
    let record = appearance_record(request, candidate.info().source);
    candidate.edit(record).map_err(Error)?;
    Ok(AssetChange::Appearance(candidate))
}
fn appearance_record(request: UpdateAppearanceArgs, source: String) -> AppearanceInfo {
    AppearanceInfo {
        id: request.id,
        source,
        display_name: request.display_name,
        base: request.base,
        specular: request.specular,
        transmission: request.transmission,
        subsurface: request.subsurface,
        coat: request.coat,
        fuzz: request.fuzz,
        thin_film: request.thin_film,
        emission: request.emission,
        geometry: request.geometry,
    }
}
fn fresh_asset_identity<T: AssetIdentity>(scope: &ScopedAssets<'_, T>, name: &str) -> String {
    let slug = asset_slug(name);
    let mut identity = slug.clone();
    let mut suffix = 2_u64;
    while scope.resolve(&identity).is_ok() {
        identity = format!("{slug}-{suffix}");
        suffix += 1;
    }
    identity
}
fn asset_slug(name: &str) -> String {
    // Go strings.ToLower maps each rune once, unlike Rust's expanding string lowercase.
    let text: String = name
        .chars()
        .map(|letter| {
            let lower = letter.to_lowercase().next().unwrap_or(letter);
            if lower.is_ascii_alphanumeric() {
                lower
            } else {
                '-'
            }
        })
        .collect();
    let slug = text.trim_matches('-');
    if slug.is_empty() {
        "asset".into()
    } else {
        slug.into()
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn slug_retains_go_rune_and_separator_rules() {
        for (name, expected) in [
            ("Brushed Aluminum", "brushed-aluminum"),
            (" A  B ", "a--b"),
            ("金", "asset"),
            ("İK", "ik"),
        ] {
            assert_eq!(asset_slug(name), expected);
        }
    }
    #[test]
    fn unknown_command_is_refused() {
        let empty = Default::default();
        let physical = ScopedAssets {
            document: &empty,
            project: &empty,
            builtin: &empty,
        };
        let empty = Default::default();
        let visual = ScopedAssets {
            document: &empty,
            project: &empty,
            builtin: &empty,
        };
        assert!(
            command(&physical, &visual, "unknown", b"{}")
                .err()
                .unwrap()
                .to_string()
                .contains("unknown")
        );
    }
}
