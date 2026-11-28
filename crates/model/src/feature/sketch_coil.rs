// SPDX-License-Identifier: GPL-2.0-only
//! Authored Coil inputs (helical and spiral sweeps), separate from kernel results and replay.
mod resolve;
pub use resolve::ResolvedSketchCoil;

use crate::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        fillet::reference_identity,
        sketch_prism::{PrismBoolean, SketchPrismSelection},
        sketch_revolve::RevolveBooleanRecipe,
        work_axis::WorkAxisReference,
        work_geometry::{DatumAngle, DatumLength},
    },
    sketch::SketchEntityId,
};

/// Coil sizing method (two of pitch/revolutions/height, or flat outward spiral).
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum CoilMethod {
    #[default]
    PitchAndRevolution,
    RevolutionAndHeight,
    PitchAndHeight,
    Spiral,
}

/// Winding sense about the coil axis.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum CoilHandedness {
    #[default]
    Left,
    Right,
}

/// Optional flat and transition sweep angles grounding one spring end.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct CoilEndTreatment {
    pub flat_angle: DatumAngle,
    pub transition_angle: DatumAngle,
}

impl CoilEndTreatment {
    pub fn same_recipe(self, other: Self) -> bool {
        self.flat_angle.same_recipe(other.flat_angle)
            && self.transition_angle.same_recipe(other.transition_angle)
    }

    pub(crate) fn validate(self) -> Result<(), String> {
        self.flat_angle
            .validate_source()
            .map_err(|cause| format!("coil end flat angle: {cause}"))?;
        self.transition_angle
            .validate_source()
            .map_err(|cause| format!("coil end transition angle: {cause}"))?;
        Ok(())
    }
}

/// Axis source for a Coil: an origin/work axis or an explicit sketch line.
#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum CoilAxisSource {
    WorkAxis(WorkAxisReference),
    SketchLine(SketchEntityId),
}

/// Serialized Coil recipe stored inside the Part feature program.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchCoilRecipe {
    pub version: u32,
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub axis: CoilAxisSource,
    #[serde(default)]
    pub axis_reversed: bool,
    #[serde(default)]
    pub method: CoilMethod,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub pitch: Option<DatumLength>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub revolutions: Option<f64>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub height: Option<DatumLength>,
    pub taper: DatumAngle,
    #[serde(default)]
    pub handedness: CoilHandedness,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub start_end: Option<CoilEndTreatment>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub end_end: Option<CoilEndTreatment>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub boolean_target: Option<RevolveBooleanRecipe>,
}

/// Validated authored Coil definition with exact sizing, axis, rotation, ends, and Boolean target.
#[derive(Clone, Debug)]
pub struct SketchCoilDefinition {
    sketch: DocumentSketchId,
    selection: SketchPrismSelection,
    axis: CoilAxisSource,
    axis_reversed: bool,
    method: CoilMethod,
    pitch: Option<DatumLength>,
    revolutions: Option<f64>,
    height: Option<DatumLength>,
    taper: DatumAngle,
    handedness: CoilHandedness,
    start_end: Option<CoilEndTreatment>,
    end_end: Option<CoilEndTreatment>,
    boolean_target: Option<(RevolveBooleanRecipe, FeatureId)>,
}

impl SketchCoilDefinition {
    /// Construct and validate a Coil definition from its recipe.
    pub fn from_recipe(recipe: &SketchCoilRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "coil recipe version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.selection.outer.is_empty() {
            return Err("coil profile: expected a nonempty outer boundary".into());
        }
        if let Some(pitch) = recipe.pitch {
            pitch
                .validate_source()
                .map_err(|cause| format!("coil pitch: {cause}"))?;
        }
        if let Some(height) = recipe.height {
            height
                .validate_source()
                .map_err(|cause| format!("coil height: {cause}"))?;
        }
        if let Some(revs) = recipe.revolutions
            && (!revs.is_finite() || revs <= 0.0)
        {
            return Err(format!(
                "coil revolutions {revs}: expected a positive finite turn count"
            ));
        }
        recipe
            .taper
            .validate_source()
            .map_err(|cause| format!("coil taper: {cause}"))?;
        if let Some(end) = recipe.start_end {
            end.validate()?;
        }
        if let Some(end) = recipe.end_end {
            end.validate()?;
        }
        match recipe.method {
            CoilMethod::PitchAndRevolution | CoilMethod::Spiral => {
                if recipe.pitch.is_none() || recipe.revolutions.is_none() {
                    return Err("coil: expected pitch and revolutions".into());
                }
            }
            CoilMethod::RevolutionAndHeight => {
                if recipe.revolutions.is_none() || recipe.height.is_none() {
                    return Err("coil: expected revolutions and height".into());
                }
            }
            CoilMethod::PitchAndHeight => {
                if recipe.pitch.is_none() || recipe.height.is_none() {
                    return Err("coil: expected pitch and height".into());
                }
            }
        }
        let boolean_target = recipe
            .boolean_target
            .as_ref()
            .map(|target| {
                let (_, source) = reference_identity(&target.key, 4, "coil Boolean target")?;
                Ok::<_, String>((target.clone(), source))
            })
            .transpose()?;
        Ok(Self {
            sketch: recipe.sketch,
            selection: recipe.selection.clone(),
            axis: recipe.axis,
            axis_reversed: recipe.axis_reversed,
            method: recipe.method,
            pitch: recipe.pitch,
            revolutions: recipe.revolutions,
            height: recipe.height,
            taper: recipe.taper,
            handedness: recipe.handedness,
            start_end: recipe.start_end,
            end_end: recipe.end_end,
            boolean_target,
        })
    }

    /// Capture the exact versioned recipe.
    pub fn recipe(&self) -> SketchCoilRecipe {
        SketchCoilRecipe {
            version: 1,
            sketch: self.sketch,
            selection: self.selection.clone(),
            axis: self.axis,
            axis_reversed: self.axis_reversed,
            method: self.method,
            pitch: self.pitch,
            revolutions: self.revolutions,
            height: self.height,
            taper: self.taper,
            handedness: self.handedness,
            start_end: self.start_end,
            end_end: self.end_end,
            boolean_target: self.boolean_target.as_ref().map(|(r, _)| r.clone()),
        }
    }

    pub fn sketch(&self) -> DocumentSketchId {
        self.sketch
    }

    pub fn selection(&self) -> &SketchPrismSelection {
        &self.selection
    }

    pub fn axis(&self) -> CoilAxisSource {
        self.axis
    }

    pub fn axis_reversed(&self) -> bool {
        self.axis_reversed
    }

    pub fn method(&self) -> CoilMethod {
        self.method
    }

    pub fn pitch(&self) -> Option<DatumLength> {
        self.pitch
    }

    pub fn revolutions(&self) -> Option<f64> {
        self.revolutions
    }

    pub fn height(&self) -> Option<DatumLength> {
        self.height
    }

    pub fn taper(&self) -> DatumAngle {
        self.taper
    }

    pub fn handedness(&self) -> CoilHandedness {
        self.handedness
    }

    pub fn start_end(&self) -> Option<CoilEndTreatment> {
        self.start_end
    }

    pub fn end_end(&self) -> Option<CoilEndTreatment> {
        self.end_end
    }

    pub fn boolean_target(&self) -> Option<(PrismBoolean, &[u8], FeatureId)> {
        self.boolean_target
            .as_ref()
            .map(|(recipe, source)| (recipe.operation, recipe.key.as_slice(), *source))
    }

    /// Direct sketch, axis feature, target body, and scalar parameter dependencies.
    pub fn source_reads(&self) -> FeatureReadFootprint {
        let mut keys = vec![FeatureReadKey::Sketch(self.sketch.0)];
        if let CoilAxisSource::WorkAxis(WorkAxisReference::Feature(id)) = self.axis {
            keys.push(FeatureReadKey::Feature(id));
        }
        if let Some(p) = self.pitch.and_then(DatumLength::parameter) {
            keys.push(FeatureReadKey::Parameter(p));
        }
        if let Some(h) = self.height.and_then(DatumLength::parameter) {
            keys.push(FeatureReadKey::Parameter(h));
        }
        if let Some(p) = self.taper.parameter_id() {
            keys.push(FeatureReadKey::Parameter(p));
        }
        for end in [self.start_end, self.end_end].into_iter().flatten() {
            if let Some(p) = end.flat_angle.parameter_id() {
                keys.push(FeatureReadKey::Parameter(p));
            }
            if let Some(p) = end.transition_angle.parameter_id() {
                keys.push(FeatureReadKey::Parameter(p));
            }
        }
        if let Some((_, _, source)) = self.boolean_target() {
            keys.push(FeatureReadKey::Feature(source));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Compare exact authored inputs, including exact scalar representation.
    pub fn same_recipe(&self, other: &Self) -> bool {
        let same_len = |a: Option<DatumLength>, b: Option<DatumLength>| match (a, b) {
            (Some(x), Some(y)) => x.same_recipe(y),
            (None, None) => true,
            _ => false,
        };
        let same_end = |a: Option<CoilEndTreatment>, b: Option<CoilEndTreatment>| match (a, b) {
            (Some(x), Some(y)) => x.same_recipe(y),
            (None, None) => true,
            _ => false,
        };
        let same_revs = match (self.revolutions, other.revolutions) {
            (Some(a), Some(b)) => a.to_bits() == b.to_bits(),
            (None, None) => true,
            _ => false,
        };
        self.sketch == other.sketch
            && self.selection == other.selection
            && self.axis == other.axis
            && self.axis_reversed == other.axis_reversed
            && self.method == other.method
            && same_len(self.pitch, other.pitch)
            && same_revs
            && same_len(self.height, other.height)
            && self.taper.same_recipe(other.taper)
            && self.handedness == other.handedness
            && same_end(self.start_end, other.start_end)
            && same_end(self.end_end, other.end_end)
            && self.boolean_target == other.boolean_target
    }
}

impl PartialEq for SketchCoilDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
