// SPDX-License-Identifier: GPL-2.0-only
mod color;
mod gamma;
use oblikovati_api::{
    types::{AssetSource, Color3, OpenPBRBase, OpenPBREmission, OpenPBRGeometry, OpenPBRSpecular},
    wire::AppearanceInfo,
};
use serde::Deserialize;
use serde_yaml_ng::Value;
#[derive(Default, Deserialize)]
#[serde(default)]
struct LegacyAppearance {
    id: String,
    name: String,
    albedo: String,
    metallic: f32,
    roughness: f32,
    emissive: String,
    opacity: f32,
}
pub(super) fn decode_legacy_appearance(
    name: &str,
    record: Value,
    source: AssetSource,
) -> Result<AppearanceInfo, String> {
    require_legacy_shape(name, &record)?;
    let legacy: LegacyAppearance = serde_yaml_ng::from_value(record).map_err(|cause| {
        format!("legacy appearance in {name:?}: expected typed scalar fields: {cause}")
    })?;
    require_legacy_scalars(&legacy)?;
    let base = color::linear_hex_color(&legacy.albedo)?;
    let emissive = if legacy.emissive.is_empty() {
        Color3::default()
    } else {
        color::linear_hex_color(&legacy.emissive)?
    };
    let (luminance, emission_color) = color::emission_split(emissive);
    Ok(migrated_record(
        legacy,
        source,
        base,
        luminance,
        emission_color,
    ))
}
fn require_legacy_shape(name: &str, record: &Value) -> Result<(), String> {
    if record.get("source").is_some() {
        return Err(format!(
            "legacy appearance in {name:?} source {:?}: expected scope owned by storage",
            record.get("source")
        ));
    }
    if record.get("albedo").is_none()
        || [
            "base",
            "specular",
            "transmission",
            "subsurface",
            "coat",
            "fuzz",
            "thinFilm",
            "emission",
            "geometry",
        ]
        .iter()
        .any(|group| record.get(*group).is_some())
    {
        return Err(format!(
            "appearance in {name:?} {record:?}: expected one complete legacy shape with albedo, without grouped OpenPBR fields"
        ));
    }
    Ok(())
}
fn require_legacy_scalars(legacy: &LegacyAppearance) -> Result<(), String> {
    for (field, value) in [
        ("metallic", legacy.metallic),
        ("roughness", legacy.roughness),
        ("opacity", legacy.opacity),
    ] {
        if !value.is_finite() || !(0. ..=1.).contains(&value) {
            return Err(format!(
                "legacy appearance {:?} {field}={value}: expected finite [0,1]",
                legacy.id
            ));
        }
    }
    Ok(())
}
fn migrated_record(
    legacy: LegacyAppearance,
    source: AssetSource,
    base: Color3,
    luminance: f32,
    emission_color: Color3,
) -> AppearanceInfo {
    // M46-F04 / Go aa8297d1: keep every absent lobe at its complete zero off-state.
    AppearanceInfo {
        id: legacy.id,
        display_name: legacy.name,
        source: source.0.into_owned(),
        base: OpenPBRBase {
            weight: 1.,
            color: base,
            metalness: legacy.metallic,
            ..Default::default()
        },
        specular: OpenPBRSpecular {
            weight: 1.,
            color: Color3 {
                r: 1.,
                g: 1.,
                b: 1.,
            },
            roughness: legacy.roughness,
            ior: 1.5,
            ..Default::default()
        },
        emission: OpenPBREmission {
            luminance,
            color: emission_color,
        },
        geometry: OpenPBRGeometry {
            opacity: legacy.opacity,
            ..Default::default()
        },
        ..Default::default()
    }
}
