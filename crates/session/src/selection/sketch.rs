// SPDX-License-Identifier: GPL-2.0-only
use super::{Selectable, SelectionKind, SelectionSet};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchConstraintId, SketchDeletionSelection, SketchEntityId, SketchError},
};

/// A sketch pick preserves the distinction between geometry, dimension labels, and markers.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum SketchPickTarget {
    Entity(SketchEntityId),
    Dimension(SketchConstraintId),
    GeometricConstraint(SketchConstraintId),
}
/// A checked session pick, valid only for the document generation that produced it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchPick {
    document: u64,
    generation: u64,
    sketch: DocumentSketchId,
    target: SketchPickTarget,
}
impl SketchPick {
    pub(crate) fn checked(
        document: u64,
        generation: u64,
        sketch: DocumentSketchId,
        target: SketchPickTarget,
    ) -> Self {
        Self {
            document,
            generation,
            sketch,
            target,
        }
    }
    /// Read the typed model identity without exposing mutable geometry.
    /// Example: `let target = pick.target();`.
    pub fn target(&self) -> SketchPickTarget {
        self.target
    }
    fn validate_owner(
        &self,
        document: u64,
        generation: u64,
        sketch: DocumentSketchId,
    ) -> Result<(), SketchError> {
        let supplied = (self.document, self.generation, self.sketch);
        let expected = (document, generation, sketch);
        if supplied != expected {
            return Err(SketchError::Definition(format!(
                "sketch pick {supplied:?}: expected current document, generation and sketch {expected:?}"
            )));
        }
        Ok(())
    }
}
impl Selectable for SketchPick {
    fn selection_kind(&self) -> SelectionKind {
        match self.target {
            SketchPickTarget::Entity(_) => SelectionKind::SketchEntity,
            SketchPickTarget::Dimension(_) => SelectionKind::SketchDimension,
            SketchPickTarget::GeometricConstraint(_) => SelectionKind::SketchConstraint,
        }
    }
}
impl SelectionSet<SketchPick> {
    pub(crate) fn deletion_label(&self) -> &'static str {
        let entities = self
            .items()
            .iter()
            .any(|pick| matches!(pick.target, SketchPickTarget::Entity(_)));
        let dimensions = self
            .items()
            .iter()
            .any(|pick| matches!(pick.target, SketchPickTarget::Dimension(_)));
        let constraints = self
            .items()
            .iter()
            .any(|pick| matches!(pick.target, SketchPickTarget::GeometricConstraint(_)));
        match (entities, dimensions, constraints) {
            (true, false, false) => "Delete Sketch Entities",
            (false, true, false) => "Delete Sketch Dimensions",
            (false, false, true) => "Delete Sketch Constraints",
            _ => "Delete Sketch Geometry",
        }
    }
    pub(crate) fn deletion_intent(
        &self,
        document: u64,
        generation: u64,
        sketch: DocumentSketchId,
    ) -> Result<SketchDeletionSelection, SketchError> {
        let mut intent = SketchDeletionSelection::default();
        for pick in self.items() {
            pick.validate_owner(document, generation, sketch)?;
            match pick.target {
                SketchPickTarget::Entity(id) => intent.entities.push(id),
                SketchPickTarget::Dimension(id) => intent.dimensions.push(id),
                SketchPickTarget::GeometricConstraint(id) => intent.geometric_constraints.push(id),
            }
        }
        Ok(intent)
    }
}
