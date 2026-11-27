// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn encode_reference(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<String, Error> {
    let reference = format!("datum/v1/{}", id.0);
    verify_reference(snapshot, id, &reference)?;
    Ok(reference)
}
pub(super) fn decode_reference(
    snapshot: &DocumentSnapshot,
    reference: &str,
) -> Result<FeatureId, Error> {
    let id = parse_reference(reference)?;
    snapshot.work_geometry.get(id).map_err(api_error)?;
    Ok(id)
}
fn parse_reference(reference: &str) -> Result<FeatureId, Error> {
    let digits = reference
        .strip_prefix("datum/v1/")
        .ok_or_else(|| malformed_reference(reference))?;
    let id = digits
        .parse::<u64>()
        .map_err(|_| malformed_reference(reference))?;
    if id == 0 || digits != id.to_string() {
        return Err(malformed_reference(reference));
    }
    Ok(FeatureId(id))
}
pub(super) fn verify_reference(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
    reference: &str,
) -> Result<(), Error> {
    let resolved = decode_reference(snapshot, reference)?;
    if resolved != id {
        return Err(Error(format!(
            "datum reference {reference:?} resolves to {resolved:?}: expected {id:?}"
        )));
    }
    Ok(())
}
fn malformed_reference(reference: &str) -> Error {
    Error(format!(
        "datum reference {reference:?}: expected datum/v1/<positive canonical u64 feature ID> in the active document"
    ))
}
