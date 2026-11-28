// SPDX-License-Identifier: GPL-2.0-only
//! Use mapping-valued Hole inputs so nested native YAML recipes remain portable (#4093).
use super::{DatumLength, HoleExtent, HolePosition, HoleTerminationSurface};
use serde::{Deserialize, Deserializer, Serialize, Serializer};

#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
enum LengthValue {
    Constant(f64),
    Parameter(u64),
}
impl From<DatumLength> for LengthValue {
    fn from(value: DatumLength) -> Self {
        match value {
            DatumLength::Constant(value) => Self::Constant(value),
            DatumLength::Parameter(id) => Self::Parameter(id),
        }
    }
}
impl From<LengthValue> for DatumLength {
    fn from(value: LengthValue) -> Self {
        match value {
            LengthValue::Constant(value) => Self::Constant(value),
            LengthValue::Parameter(id) => Self::Parameter(id),
        }
    }
}
#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
enum PositionValue {
    FaceCentroid,
    ModelPoint([LengthValue; 3]),
}
#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
enum ExtentValue {
    ThroughAll,
    Distance(LengthValue),
    ToSurface(HoleTerminationSurface),
}

pub(super) mod length {
    use super::*;
    pub(crate) fn serialize<S: Serializer>(
        value: &DatumLength,
        serializer: S,
    ) -> Result<S::Ok, S::Error> {
        LengthValue::from(*value).serialize(serializer)
    }
    pub(crate) fn deserialize<'de, D: Deserializer<'de>>(
        deserializer: D,
    ) -> Result<DatumLength, D::Error> {
        LengthValue::deserialize(deserializer).map(Into::into)
    }
}
pub(super) mod position {
    use super::*;
    pub(crate) fn serialize<S: Serializer>(
        value: &HolePosition,
        serializer: S,
    ) -> Result<S::Ok, S::Error> {
        let value = match value {
            HolePosition::FaceCentroid => PositionValue::FaceCentroid,
            HolePosition::ModelPoint(point) => PositionValue::ModelPoint(point.map(Into::into)),
        };
        value.serialize(serializer)
    }
    pub(crate) fn deserialize<'de, D: Deserializer<'de>>(
        deserializer: D,
    ) -> Result<HolePosition, D::Error> {
        Ok(match PositionValue::deserialize(deserializer)? {
            PositionValue::FaceCentroid => HolePosition::FaceCentroid,
            PositionValue::ModelPoint(point) => HolePosition::ModelPoint(point.map(Into::into)),
        })
    }
}
pub(super) mod extent {
    use super::*;
    pub(crate) fn serialize<S: Serializer>(
        value: &HoleExtent,
        serializer: S,
    ) -> Result<S::Ok, S::Error> {
        let value = match value {
            HoleExtent::ThroughAll => ExtentValue::ThroughAll,
            HoleExtent::Distance(depth) => ExtentValue::Distance((*depth).into()),
            HoleExtent::ToSurface(surface) => ExtentValue::ToSurface(surface.clone()),
        };
        value.serialize(serializer)
    }
    pub(crate) fn deserialize<'de, D: Deserializer<'de>>(
        deserializer: D,
    ) -> Result<HoleExtent, D::Error> {
        Ok(match ExtentValue::deserialize(deserializer)? {
            ExtentValue::ThroughAll => HoleExtent::ThroughAll,
            ExtentValue::Distance(depth) => HoleExtent::Distance(depth.into()),
            ExtentValue::ToSurface(surface) => HoleExtent::ToSurface(surface),
        })
    }
}
