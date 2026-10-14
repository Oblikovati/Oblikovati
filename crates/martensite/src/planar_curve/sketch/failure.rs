// SPDX-License-Identifier: GPL-2.0-only
use super::super::PlanarCurveFailure;
use oblikovati_model::sketch::{SketchEntityId, SketchError};
use std::{error::Error, fmt};
/// Original model or kernel-adapter failure, without string-only conversion.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchCurveFailureCause {
    Model(Box<SketchError>),
    Curve(Box<PlanarCurveFailure>),
}
/// The offending sketch identity and the original typed failure.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchCurveFailure {
    pub entity: SketchEntityId,
    pub cause: SketchCurveFailureCause,
}
impl SketchCurveFailure {
    pub(super) fn model(entity: SketchEntityId, cause: SketchError) -> Self {
        Self {
            entity,
            cause: SketchCurveFailureCause::Model(Box::new(cause)),
        }
    }
    pub(super) fn curve(entity: SketchEntityId, cause: PlanarCurveFailure) -> Self {
        Self {
            entity,
            cause: SketchCurveFailureCause::Curve(Box::new(cause)),
        }
    }
    pub(super) fn input(entity: SketchEntityId, message: String) -> Self {
        Self::curve(entity, PlanarCurveFailure::input(message))
    }
}
impl fmt::Display for SketchCurveFailure {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(formatter, "sketch curve {:?}: ", self.entity)?;
        match &self.cause {
            SketchCurveFailureCause::Model(cause) => cause.fmt(formatter),
            SketchCurveFailureCause::Curve(cause) => cause.fmt(formatter),
        }
    }
}
impl Error for SketchCurveFailure {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match &self.cause {
            SketchCurveFailureCause::Model(cause) => Some(cause.as_ref()),
            SketchCurveFailureCause::Curve(cause) => Some(cause.as_ref()),
        }
    }
}
