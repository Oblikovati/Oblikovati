// SPDX-License-Identifier: GPL-2.0-only
//! Authored revolve inputs, separate from kernel results and complete feature replay.
mod error;
mod recipe;
mod resolve;
mod validation;
use crate::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        fillet::reference_identity,
        sketch_prism::{PrismBoolean, SketchPrismSelection},
        work_geometry::DatumAngle,
    },
    sketch::SketchEntityId,
};
pub use error::{SketchRevolveCause, SketchRevolveError};
pub use recipe::{RevolveBooleanRecipe, SketchRevolveRecipe};
pub use resolve::ResolvedSketchRevolve;

/// Direction of an angular revolution around its axis.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum SketchRevolveDirection {
    #[default]
    Positive,
    Negative,
    Symmetric,
    Asymmetric,
}

impl SketchRevolveDirection {
    pub(crate) fn is_positive(&self) -> bool {
        *self == Self::Positive
    }
}

/// Validated authored revolve input with explicit line or sole centerline axis, extent, and output.
#[derive(Clone, Debug)]
pub struct SketchRevolveDefinition {
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    axis: Option<SketchEntityId>,
    angle: DatumAngle,
    second_angle: Option<DatumAngle>,
    full: bool,
    direction: SketchRevolveDirection,
    surface: bool,
    boolean_target: Option<(RevolveBooleanRecipe, FeatureId)>,
}

impl SketchRevolveDefinition {
    /// Validate authored profile identities and one angular driver.
    /// Example: `SketchRevolveDefinition::new(sketch, selection, DatumAngle::parameter(3))?`.
    pub fn new(
        sketch: DocumentSketchId,
        selection: SketchPrismSelection,
        angle: DatumAngle,
    ) -> Result<Self, SketchRevolveError> {
        let definition = Self {
            sketch,
            selection,
            axis: None,
            angle,
            second_angle: None,
            full: false,
            direction: SketchRevolveDirection::Positive,
            surface: false,
            boolean_target: None,
        };
        validation::validate_sources(&definition).map_err(|cause| definition.decline(cause))?;
        Ok(definition)
    }

    /// Replace the primary angular driver (for example to bind an owned model parameter).
    /// Example: `definition.with_angle(DatumAngle::parameter(3))?`.
    pub fn with_angle(mut self, angle: DatumAngle) -> Result<Self, SketchRevolveError> {
        self.angle = angle;
        validation::validate_sources(&self).map_err(|cause| self.decline(cause))?;
        Ok(self)
    }

    /// Select an explicit straight sketch line as the revolution axis.
    /// Example: `definition.with_axis(SketchEntityId(6))?`.
    pub fn with_axis(mut self, axis: SketchEntityId) -> Result<Self, SketchRevolveError> {
        self.axis = Some(axis);
        validation::validate_sources(&self).map_err(|cause| self.decline(cause))?;
        Ok(self)
    }

    /// Mark this revolve as a full 360-degree turn.
    /// Example: `definition.with_full_turn()`.
    pub fn with_full_turn(mut self) -> Self {
        self.full = true;
        self.second_angle = None;
        self
    }

    /// Set the angular sweep direction (clears full-turn mode).
    /// Example: `definition.with_direction(SketchRevolveDirection::Symmetric)`.
    pub fn with_direction(mut self, direction: SketchRevolveDirection) -> Self {
        self.direction = direction;
        self
    }

    /// Set the second angular driver for Asymmetric direction.
    /// Example: `definition.with_asymmetric_angle(DatumAngle::constant(45_f64.to_radians())?)?`.
    pub fn with_asymmetric_angle(
        mut self,
        second_angle: DatumAngle,
    ) -> Result<Self, SketchRevolveError> {
        self.full = false;
        self.direction = SketchRevolveDirection::Asymmetric;
        self.second_angle = Some(second_angle);
        validation::validate_sources(&self).map_err(|cause| self.decline(cause))?;
        Ok(self)
    }

    /// Toggle surface output (clears any solid Boolean target when enabled).
    /// Example: `definition.with_surface_output(true)`.
    pub fn with_surface_output(mut self, surface: bool) -> Self {
        self.surface = surface;
        if surface {
            self.boolean_target = None;
        }
        self
    }

    /// Bind a solid Boolean operation (Join, Cut, Intersect) to an exact target body key.
    /// Example: `definition.with_boolean_target(PrismBoolean::Cut, key)?`.
    pub fn with_boolean_target(
        mut self,
        operation: PrismBoolean,
        key: Vec<u8>,
    ) -> Result<Self, SketchRevolveError> {
        let (_, source) = reference_identity(&key, 4, "revolve Boolean target")
            .map_err(|cause| self.decline(SketchRevolveCause::Definition(cause)))?;
        self.surface = false;
        self.boolean_target = Some((RevolveBooleanRecipe { operation, key }, source));
        Ok(self)
    }

    /// Clear any solid Boolean target so the revolve creates an independent solid.
    pub fn without_boolean_target(mut self) -> Self {
        self.boolean_target = None;
        self
    }

    /// Inspect the stable source sketch; example: `definition.sketch()` before dependency analysis.
    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }

    /// Inspect explicit profile traversal; example: `definition.selection().outer`.
    pub fn selection(&self) -> &SketchPrismSelection {
        &self.selection
    }

    /// Inspect the optional explicit axis entity; `None` uses the sketch's sole centerline.
    pub fn axis(&self) -> Option<SketchEntityId> {
        self.axis
    }

    /// Inspect the retained primary angular driver; example: `definition.angle()`.
    pub fn angle(&self) -> DatumAngle {
        self.angle
    }

    /// Inspect the optional secondary angular driver for Asymmetric mode.
    pub fn second_angle(&self) -> Option<DatumAngle> {
        self.second_angle
    }

    /// Return whether the authored extent is a full 360-degree revolution.
    pub fn is_full(&self) -> bool {
        self.full
    }

    /// Return the authored sweep direction.
    pub fn direction(&self) -> SketchRevolveDirection {
        self.direction
    }

    /// Return whether this revolve outputs an open/closed surface shell rather than a solid.
    pub fn is_surface(&self) -> bool {
        self.surface
    }

    /// Read the solid Boolean operation, target reference key, and producing feature if bound.
    pub fn boolean_target(&self) -> Option<(PrismBoolean, &[u8], FeatureId)> {
        self.boolean_target
            .as_ref()
            .map(|(recipe, source)| (recipe.operation, recipe.key.as_slice(), *source))
    }

    /// Direct sketch, target body, and angle parameter dependencies without resolving sketch internals.
    pub fn source_reads(&self) -> FeatureReadFootprint {
        let mut keys = vec![FeatureReadKey::Sketch(self.sketch.0)];
        if let Some(parameter) = self.angle.parameter_id() {
            keys.push(FeatureReadKey::Parameter(parameter));
        }
        if let Some(parameter) = self.second_angle.and_then(DatumAngle::parameter_id) {
            keys.push(FeatureReadKey::Parameter(parameter));
        }
        if let Some((_, _, source)) = self.boolean_target() {
            keys.push(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Compare only authored sources, including exact scalar representation.
    /// Example: `before.same_recipe(&after)` before a no-op edit is accepted.
    pub fn same_recipe(&self, other: &Self) -> bool {
        let same_second = match (self.second_angle, other.second_angle) {
            (Some(a), Some(b)) => a.same_recipe(b),
            (None, None) => true,
            _ => false,
        };
        self.sketch == other.sketch
            && self.selection == other.selection
            && self.axis == other.axis
            && self.full == other.full
            && self.direction == other.direction
            && self.surface == other.surface
            && self.boolean_target == other.boolean_target
            && self.angle.same_recipe(other.angle)
            && same_second
    }

    fn decline(&self, cause: SketchRevolveCause) -> SketchRevolveError {
        SketchRevolveError {
            sketch: self.sketch,
            cause: Box::new(cause),
        }
    }
}

impl PartialEq for SketchRevolveDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
