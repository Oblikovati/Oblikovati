// SPDX-License-Identifier: GPL-2.0-only
//! Authored coordinate-frame changes retain their source identities (#4107).
use super::{
    DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe, SketchAxisReference,
    SketchAxisResolver,
};
use crate::{
    feature::{FeatureReadKey, work_geometry::WorkPointReference},
    sketch::{SketchError, SketchFrameAlignment, SketchFrameAxis, SketchPlane},
};

/// An origin is either an absolute point or an associative datum.
/// Example: `SketchFrameOrigin::Position([1., 2., 0.])`.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum SketchFrameOrigin {
    Position([f64; 3]),
    Datum(WorkPointReference),
    Topology(super::SketchFrameTopologyOrigin),
}
/// An in-plane axis follows its source reference.
/// Example: `SketchFrameAxisEdit { reference: SketchAxisReference::OriginX, axis: SketchFrameAxis::X, reverse: false }`.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct SketchFrameAxisEdit {
    pub reference: SketchAxisReference,
    pub axis: SketchFrameAxis,
    pub reverse: bool,
}
/// Independent authored changes to a sketch's existing frame.
/// Example: `SketchFrameEditDefinition { origin: None, alignment: Some(axis) }`.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchFrameEditDefinition {
    pub origin: Option<SketchFrameOrigin>,
    pub alignment: Option<SketchFrameAxisEdit>,
}
impl PlacedSketch {
    /// Inspect authored frame changes; example: `sketch.frame_edit()`.
    pub fn frame_edit(&self) -> Option<&SketchFrameEditDefinition> {
        self.frame_edit.as_ref()
    }
}
impl DocumentSketches {
    /// Apply supplied aspects without losing existing ones; example: `sketches.patch_frame_edit(id, patch)?`.
    pub fn patch_frame_edit(
        &mut self,
        id: DocumentSketchId,
        patch: SketchFrameEditDefinition,
    ) -> Result<(), SketchError> {
        patch.validate()?;
        let previous = self.get(id)?.frame_edit();
        let edit = SketchFrameEditDefinition {
            origin: patch
                .origin
                .or_else(|| previous.and_then(|value| value.origin.clone())),
            alignment: patch
                .alignment
                .or_else(|| previous.and_then(|value| value.alignment)),
        };
        self.set_frame_edit(id, Some(edit))
    }
    /// Replace the authored edit after validation; example: `sketches.set_frame_edit(id, Some(edit))?`.
    pub fn set_frame_edit(
        &mut self,
        id: DocumentSketchId,
        edit: Option<SketchFrameEditDefinition>,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        if let Some(value) = &edit {
            value.validate()?;
        }
        self.sketches[index].frame_edit = edit;
        Ok(())
    }
}
impl SketchFrameEditDefinition {
    fn validate(&self) -> Result<(), SketchError> {
        if self.origin.is_none() && self.alignment.is_none() {
            return Err(SketchError::Definition(
                "empty frame edit: expected an origin or axis alignment".into(),
            ));
        }
        self.validate_origin()?;
        self.validate_alignment()
    }
    fn validate_origin(&self) -> Result<(), SketchError> {
        match &self.origin {
            Some(SketchFrameOrigin::Position(origin))
                if !origin.iter().all(|value| value.is_finite()) =>
            {
                Err(SketchError::Definition(format!(
                    "frame origin {origin:?}: expected three finite coordinates"
                )))
            }
            Some(SketchFrameOrigin::Datum(WorkPointReference::Feature(
                crate::feature::FeatureId(0),
            ))) => Err(SketchError::Definition(
                "frame origin feature 0: expected a nonzero datum reference".into(),
            )),
            _ => Ok(()),
        }
    }
    fn validate_alignment(&self) -> Result<(), SketchError> {
        if self.alignment.is_some_and(|axis| {
            axis.reference == SketchAxisReference::Feature(crate::feature::FeatureId(0))
        }) {
            return Err(SketchError::Definition(format!(
                "frame alignment {:?}: expected a nonzero datum reference",
                self.alignment
            )));
        }
        Ok(())
    }
    pub(crate) fn requires_datums(&self) -> bool {
        !self.read_keys().is_empty()
    }
    pub(super) fn read_keys(&self) -> Vec<FeatureReadKey> {
        let origin = match &self.origin {
            Some(SketchFrameOrigin::Datum(WorkPointReference::Feature(id))) => {
                Some(FeatureReadKey::Feature(*id))
            }
            Some(SketchFrameOrigin::Topology(reference)) => {
                Some(FeatureReadKey::Feature(reference.source()))
            }
            _ => None,
        };
        let axis = self
            .alignment
            .and_then(|alignment| match alignment.reference {
                SketchAxisReference::Feature(id) => Some(FeatureReadKey::Feature(id)),
                _ => None,
            });
        origin.into_iter().chain(axis).collect()
    }
    pub(super) fn resolve(
        &self,
        base: SketchPlane,
        resolver: &dyn SketchAxisResolver,
    ) -> Result<SketchPlane, SketchError> {
        let origin = match &self.origin {
            Some(SketchFrameOrigin::Position(at)) => Some(*at),
            Some(SketchFrameOrigin::Datum(reference)) => Some(resolver.frame_origin(*reference)?),
            Some(SketchFrameOrigin::Topology(reference)) => {
                Some(resolver.frame_topology_origin(reference)?)
            }
            None => None,
        };
        let alignment = self
            .alignment
            .map(|axis| axis.resolve(resolver))
            .transpose()?;
        base.with_frame_edit(origin, alignment)
    }
}
impl SketchFrameAxisEdit {
    fn resolve(
        self,
        resolver: &dyn SketchAxisResolver,
    ) -> Result<SketchFrameAlignment, SketchError> {
        let direction = super::orientation::resolve_axis_reference(self.reference, resolver)?;
        Ok(SketchFrameAlignment {
            direction,
            axis: self.axis,
            reverse: self.reverse,
        })
    }
}
pub(super) fn restore(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if recipe.frame_edit.is_some() && version < 15 {
        return Err(SketchError::Definition(format!(
            "sketch {:?} frame edit in recipe version {version}: expected version 15",
            recipe.id
        )));
    }
    sketches.set_frame_edit(recipe.id, recipe.frame_edit.clone())
}

pub(super) fn same(
    first: Option<&SketchFrameEditDefinition>,
    second: Option<&SketchFrameEditDefinition>,
) -> bool {
    let (Some(first), Some(second)) = (first, second) else {
        return first.is_none() && second.is_none();
    };
    let origins_match = match (&first.origin, &second.origin) {
        (Some(SketchFrameOrigin::Position(left)), Some(SketchFrameOrigin::Position(right))) => {
            left.map(f64::to_bits) == right.map(f64::to_bits)
        }
        (left, right) => left == right,
    };
    origins_match && first.alignment == second.alignment
}
