// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkernel::standard::Failure;
use std::{error::Error, fmt};
/// Failure classes can be inspected without parsing a displayed native exception.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PlanarCurveFailureKind {
    InvalidInput,
    Kernel,
    Incomplete,
    InvalidResult,
}
/// The original native exception is retained as the error source.
#[derive(Clone, Debug, PartialEq)]
pub struct PlanarCurveFailure {
    kind: PlanarCurveFailureKind,
    message: String,
    cause: Option<Box<Failure>>,
}
impl PlanarCurveFailure {
    pub(super) fn with_input(self, operand: &str, input: &super::PlanarCurveDefinition) -> Self {
        self.with_context(format!("{operand} curve {input:?}"))
    }

    pub(super) fn with_context(mut self, context: String) -> Self {
        self.message = format!("{context}: {}", self.message);
        self
    }

    /// Inspect the failure class; example: `failure.kind() == PlanarCurveFailureKind::Incomplete`.
    pub fn kind(&self) -> PlanarCurveFailureKind {
        self.kind
    }
    pub(super) fn input(message: String) -> Self {
        Self {
            kind: PlanarCurveFailureKind::InvalidInput,
            message,
            cause: None,
        }
    }
    pub(super) fn result(message: String) -> Self {
        Self {
            kind: PlanarCurveFailureKind::InvalidResult,
            message,
            cause: None,
        }
    }
    pub(super) fn incomplete() -> Self {
        Self {
            kind: PlanarCurveFailureKind::Incomplete,
            message: "planar intersection is not done: expected a completed native result".into(),
            cause: None,
        }
    }
}
impl From<Failure> for PlanarCurveFailure {
    fn from(cause: Failure) -> Self {
        Self {
            kind: PlanarCurveFailureKind::Kernel,
            message: "planar curve operation: expected a successful native result".into(),
            cause: Some(Box::new(cause)),
        }
    }
}
impl fmt::Display for PlanarCurveFailure {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(&self.message)?;
        if let Some(cause) = &self.cause {
            write!(formatter, ": {cause}")?;
        }
        Ok(())
    }
}
impl Error for PlanarCurveFailure {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        self.cause
            .as_deref()
            .map(|cause| cause as &(dyn Error + 'static))
    }
}
