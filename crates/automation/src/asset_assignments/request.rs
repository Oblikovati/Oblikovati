// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::identity::ReferenceEnvelope;
pub(super) enum AssignmentChange {
    Material(MaterialAssignmentTarget, String),
    Appearance(AppearanceAssignmentTarget, String),
}
pub(super) fn parse(method: &str, encoded: &[u8]) -> Result<AssignmentChange, Error> {
    match method {
        METHOD_MODEL_ASSIGN_MATERIAL => material(decode(method, encoded)?),
        METHOD_MODEL_ASSIGN_APPEARANCE => appearance(decode(method, encoded)?),
        other => Err(Error(format!(
            "assignment method {other:?}: expected {ASSET_ASSIGNMENT_METHODS:?}"
        ))),
    }
}
fn material(args: AssignMaterialArgs) -> Result<AssignmentChange, Error> {
    require_asset_id(&args.material_id)?;
    let target = if args.body_key.is_empty() {
        MaterialAssignmentTarget::Part
    } else {
        MaterialAssignmentTarget::Body(reference(&args.body_key)?)
    };
    Ok(AssignmentChange::Material(target, args.material_id))
}
fn appearance(args: AssignAppearanceArgs) -> Result<AssignmentChange, Error> {
    require_asset_id(&args.appearance_id)?;
    let target = match args.scope.as_str() {
        "part" if args.key.is_empty() => AppearanceAssignmentTarget::Part,
        "body" => AppearanceAssignmentTarget::Body(reference(&args.key)?),
        "face" => AppearanceAssignmentTarget::Face(reference(&args.key)?),
        _ => {
            return Err(Error(format!(
                "appearance scope {:?}, key {:?}: expected part with empty key, or body/face with a hex reference",
                args.scope, args.key
            )));
        }
    };
    Ok(AssignmentChange::Appearance(target, args.appearance_id))
}
fn reference(text: &str) -> Result<AssignmentReference, Error> {
    let bytes = hex_bytes(text)?;
    let canonical = ReferenceEnvelope::decode(&bytes)
        .and_then(|key| key.encode_current())
        .map_err(|cause| Error(cause.to_string()))?;
    let encoded = canonical
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect::<String>();
    AssignmentReference::new(&encoded, vec![]).map_err(Error)
}
fn hex_bytes(text: &str) -> Result<Vec<u8>, Error> {
    if text.is_empty()
        || !text.len().is_multiple_of(2)
        || !text.bytes().all(|byte| byte.is_ascii_hexdigit())
    {
        return Err(Error(format!(
            "reference key {text:?}: expected nonempty hexadecimal byte pairs"
        )));
    }
    (0..text.len())
        .step_by(2)
        .map(|index| {
            u8::from_str_radix(&text[index..index + 2], 16).map_err(|cause| {
                Error(format!(
                    "reference key {text:?}: expected hexadecimal bytes: {cause}"
                ))
            })
        })
        .collect()
}
impl AssignmentChange {
    pub(super) fn entity(&self) -> Result<Option<(Vec<u8>, AssetEntityKind)>, Error> {
        let (reference, kind) = match self {
            Self::Material(MaterialAssignmentTarget::Body(reference), _)
            | Self::Appearance(AppearanceAssignmentTarget::Body(reference), _) => {
                (reference, AssetEntityKind::Body)
            }
            Self::Appearance(AppearanceAssignmentTarget::Face(reference), _) => {
                (reference, AssetEntityKind::Face)
            }
            _ => return Ok(None),
        };
        Ok(Some((hex_bytes(reference.reference_key())?, kind)))
    }
}

fn require_asset_id(id: &str) -> Result<(), Error> {
    if id.is_empty() {
        return Err(Error(format!(
            "assignment asset ID {id:?}: expected an existing nonempty asset ID"
        )));
    }
    Ok(())
}
