// SPDX-License-Identifier: GPL-2.0-only
//! Structured evaluation failures stay intact until a text-only caller renders them.
use crate::PrismFaceRole;
use oblikovati_model::{
    document_sketch::DocumentSketchId, feature::FeatureId, sketch::SketchError,
};
use rustycad_tkernel::standard::Failure;
use std::{error::Error, fmt};

/// The operation stage that refused or failed an evaluation.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EvaluationPhase {
    Input,
    Placement,
    Profile,
    Prism,
    Validation,
    History,
    Copy,
    Measurement,
    Naming,
}
/// A failure class independent of its displayed text.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum EvaluationFailureKind {
    InvalidInput,
    Kernel,
    InvalidGeometry,
    IncompleteHistory,
    InvalidResult,
    InvalidReference,
}
/// Authored identity or semantic result role; indices are diagnostic, never persistent names.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum EvaluationEntity {
    Body,
    Loop(usize),
    Edges(Vec<u64>),
    Edge(u64),
    Face(PrismFaceRole),
    FaceIndex(usize),
    SketchEntity(u64),
}
#[derive(Clone, Debug, PartialEq)]
enum EvaluationCause {
    Kernel(Failure),
    Sketch(SketchError),
}
#[derive(Clone, PartialEq)]
struct FailureDetails {
    phase: EvaluationPhase,
    kind: EvaluationFailureKind,
    entity: EvaluationEntity,
    feature: Option<FeatureId>,
    sketch: Option<DocumentSketchId>,
    prefix: Option<String>,
    message: String,
    cause: Option<EvaluationCause>,
}
/// One owned typed cause with stage and source context, retained by feature health and scene state.
#[derive(Clone, PartialEq)]
pub struct EvaluationFailure(Box<FailureDetails>);
mod construct;
impl EvaluationFailure {
    /// Inspect the failed stage; example: `failure.phase() == EvaluationPhase::Validation`.
    pub fn phase(&self) -> EvaluationPhase {
        self.0.phase
    }
    /// Inspect the failure class without parsing text; example: `failure.kind() == EvaluationFailureKind::Kernel`.
    pub fn kind(&self) -> EvaluationFailureKind {
        self.0.kind
    }
    /// Inspect the affected source or result role; example: `matches!(failure.entity(), EvaluationEntity::Edge(7))`.
    pub fn entity(&self) -> &EvaluationEntity {
        &self.0.entity
    }
    /// Read the replay feature, when attached; example: `failure.feature() == Some(FeatureId(3))`.
    pub fn feature(&self) -> Option<FeatureId> {
        self.0.feature
    }
    /// Read the source sketch, when attached; example: `failure.sketch() == Some(DocumentSketchId(2))`.
    pub fn sketch(&self) -> Option<DocumentSketchId> {
        self.0.sketch
    }
    pub(crate) fn for_feature(mut self, feature: FeatureId) -> Self {
        self.0.feature = Some(feature);
        self
    }
    pub(crate) fn with_sketch_identity(mut self, sketch: DocumentSketchId) -> Self {
        self.0.sketch = Some(sketch);
        self
    }
    pub(crate) fn for_sketch(mut self, sketch: DocumentSketchId, prefix: String) -> Self {
        self.0.sketch = Some(sketch);
        self.0.prefix = Some(prefix);
        self
    }
}
impl fmt::Display for EvaluationFailure {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if let Some(prefix) = &self.0.prefix {
            write!(f, "{prefix}: ")?;
        }
        f.write_str(&self.0.message)?;
        if let Some(cause) = self.source() {
            if !self.0.message.is_empty() {
                f.write_str(": ")?;
            }
            write!(f, "{cause}")?;
        }
        Ok(())
    }
}
impl fmt::Debug for EvaluationFailure {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // Stored-body declines already expose String debug output. Keep those messages stable;
        // consumers inspect structured fields and Error::source instead of parsing Debug.
        fmt::Debug::fmt(&self.to_string(), f)
    }
}
impl Error for EvaluationFailure {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match &self.0.cause {
            Some(EvaluationCause::Kernel(cause)) => Some(cause),
            Some(EvaluationCause::Sketch(cause)) => Some(cause),
            None => None,
        }
    }
}
#[cfg(test)]
#[path = "../tests/unit/evaluation_failure.rs"]
mod tests;
