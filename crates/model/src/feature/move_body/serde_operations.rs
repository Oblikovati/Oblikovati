// SPDX-License-Identifier: GPL-2.0-only
//! Stable mapping form for body-motion inputs in YAML and JSON (#4097).
use super::*;
use serde::{Deserialize, Deserializer, Serialize, Serializer, de::Error};

#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
enum Scalar {
    Constant(f64),
    Parameter(u64),
}
impl From<DatumLength> for Scalar {
    fn from(value: DatumLength) -> Self {
        match value {
            DatumLength::Constant(number) => Self::Constant(number),
            DatumLength::Parameter(id) => Self::Parameter(id),
        }
    }
}
impl From<Scalar> for DatumLength {
    fn from(value: Scalar) -> Self {
        match value {
            Scalar::Constant(number) => Self::Constant(number),
            Scalar::Parameter(id) => Self::Parameter(id),
        }
    }
}
#[derive(Serialize, Deserialize)]
#[serde(tag = "kind", content = "value", deny_unknown_fields)]
enum StoredOperation {
    Translation([Scalar; 3]),
    AlongRay {
        direction: [f64; 3],
        distance: Scalar,
    },
    Rotation {
        origin: [Scalar; 3],
        direction: [f64; 3],
        angle: Scalar,
    },
}
impl From<MoveBodyOperation> for StoredOperation {
    fn from(value: MoveBodyOperation) -> Self {
        match value {
            MoveBodyOperation::Translation(values) => Self::Translation(values.map(Into::into)),
            MoveBodyOperation::AlongRay {
                direction,
                distance,
            } => Self::AlongRay {
                direction,
                distance: distance.into(),
            },
            MoveBodyOperation::Rotation {
                origin,
                direction,
                angle,
            } => Self::Rotation {
                origin: origin.map(Into::into),
                direction,
                angle: angle.source().into(),
            },
        }
    }
}
impl StoredOperation {
    fn restore(self) -> Result<MoveBodyOperation, String> {
        match self {
            Self::Translation(values) => Ok(MoveBodyOperation::Translation(values.map(Into::into))),
            Self::AlongRay {
                direction,
                distance,
            } => Ok(MoveBodyOperation::AlongRay {
                direction,
                distance: distance.into(),
            }),
            Self::Rotation {
                origin,
                direction,
                angle,
            } => Ok(MoveBodyOperation::Rotation {
                origin: origin.map(Into::into),
                direction,
                angle: restore_angle(angle)?,
            }),
        }
    }
}
fn restore_angle(value: Scalar) -> Result<DatumAngle, String> {
    match value {
        Scalar::Constant(value) => DatumAngle::constant(value).map_err(|error| error.to_string()),
        Scalar::Parameter(id) => Ok(DatumAngle::parameter(id)),
    }
}
pub(super) fn serialize<S: Serializer>(
    values: &[MoveBodyOperation],
    serializer: S,
) -> Result<S::Ok, S::Error> {
    values
        .iter()
        .copied()
        .map(StoredOperation::from)
        .collect::<Vec<_>>()
        .serialize(serializer)
}
pub(super) fn deserialize<'de, D: Deserializer<'de>>(
    deserializer: D,
) -> Result<Vec<MoveBodyOperation>, D::Error> {
    Vec::<StoredOperation>::deserialize(deserializer)?
        .into_iter()
        .map(|value| value.restore().map_err(D::Error::custom))
        .collect()
}
