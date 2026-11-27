// SPDX-License-Identifier: GPL-2.0-only
//! Persist the selected origin plane in the shared application options file.
use oblikovati_model::feature::work_plane::WorkPlaneOrigin;
use serde::{Deserialize, Deserializer, Serializer, de::Error};

pub(super) fn serialize<S: Serializer>(
    plane: &Option<WorkPlaneOrigin>,
    serializer: S,
) -> Result<S::Ok, S::Error> {
    serializer.serialize_str(match plane {
        None => "none",
        Some(WorkPlaneOrigin::XY) => "xy",
        Some(WorkPlaneOrigin::YZ) => "yz",
        Some(WorkPlaneOrigin::XZ) => "xz",
    })
}
pub(super) fn deserialize<'de, D: Deserializer<'de>>(
    deserializer: D,
) -> Result<Option<WorkPlaneOrigin>, D::Error> {
    let value = Option::<String>::deserialize(deserializer)?;
    match value.as_deref() {
        None | Some("none") => Ok(None),
        Some("xy") => Ok(Some(WorkPlaneOrigin::XY)),
        Some("yz") => Ok(Some(WorkPlaneOrigin::YZ)),
        Some("xz") => Ok(Some(WorkPlaneOrigin::XZ)),
        Some(other) => Err(D::Error::custom(format!(
            "startup sketch plane {other:?}: expected none, xy, yz, or xz"
        ))),
    }
}
