// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Thicken/Offset feature recipe, validation, and resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    sketch_prism::PrismBoolean,
};
use std::{collections::BTreeSet, fmt};

/// Output mode selected for a Part Modify Thicken/Offset feature.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ThickenOutputMode {
    #[default]
    Solid,
    Surface,
}

impl ThickenOutputMode {
    /// Whether this feature creates an offset surface shell rather than a thickened solid.
    pub const fn is_surface(self) -> bool {
        matches!(self, Self::Surface)
    }
}

/// Offset/thicken direction relative to the selected face normal.
///
/// Order matches the Properties panel Direction buttons:
/// 1. `Negative` (first direction, inward `-normal`)
/// 2. `Positive` (second/middle direction, outward `+normal`, default)
/// 3. `Symmetric` (third direction, both sides `[-d/2, +d/2]`)
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ThickenDirection {
    Negative,
    #[default]
    Positive,
    Symmetric,
}

/// Solid Boolean output option for Thicken/Offset in Solid mode.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ThickenBoolean {
    #[default]
    Join,
    Cut,
    Intersect,
    NewSolid,
}

impl ThickenBoolean {
    /// Convert to a Martensite solid Boolean when modifying the target body.
    pub const fn prism_boolean(self) -> Option<PrismBoolean> {
        match self {
            Self::Join => Some(PrismBoolean::Join),
            Self::Cut => Some(PrismBoolean::Cut),
            Self::Intersect => Some(PrismBoolean::Intersect),
            Self::NewSolid => None,
        }
    }
}

impl fmt::Display for ThickenBoolean {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Join => f.write_str("Join"),
            Self::Cut => f.write_str("Cut"),
            Self::Intersect => f.write_str("Intersect"),
            Self::NewSolid => f.write_str("New Solid"),
        }
    }
}

/// Approximation bias used when `allow_approximation` is enabled.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ThickenApproximationType {
    #[default]
    Mean,
    NeverTooThin,
    NeverTooThick,
}

impl ThickenApproximationType {
    /// Display caption in the Thicken/Offset Advanced Properties dropdown.
    pub const fn label(self) -> &'static str {
        match self {
            Self::NeverTooThin => "Never too thin",
            Self::NeverTooThick => "Never too thick",
            Self::Mean => "Mean",
        }
    }
}

/// Tolerance policy used when `allow_approximation` is enabled.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ThickenApproximationTolerance {
    #[default]
    Optimized,
    Specified,
}

/// Persisted Part Modify Thicken/Offset recipe (Part program version 60, source version 50).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ThickenOffsetRecipe {
    pub version: u32,
    pub target_body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    #[serde(default)]
    pub output: ThickenOutputMode,
    /// Offset or thicken distance in model units (cm).
    pub distance: f64,
    #[serde(default)]
    pub direction: ThickenDirection,
    #[serde(default)]
    pub boolean: ThickenBoolean,
    #[serde(default)]
    pub automatic_face_chain: bool,
    #[serde(default)]
    pub automatic_blending: bool,
    #[serde(default)]
    pub allow_approximation: bool,
    #[serde(default)]
    pub approximation_type: ThickenApproximationType,
    #[serde(default)]
    pub approximation_tolerance: ThickenApproximationTolerance,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub approximation_percent: Option<f64>,
}

/// Validated authored Thicken/Offset definition.
#[derive(Clone, Debug, PartialEq)]
pub struct ThickenOffsetDefinition {
    recipe: ThickenOffsetRecipe,
    body_source: FeatureId,
}

/// Resolved Thicken/Offset inputs ready for Martensite evaluation.
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedThickenOffset {
    pub faces: Vec<Vec<u8>>,
    pub output: ThickenOutputMode,
    pub distance: f64,
    pub direction: ThickenDirection,
    pub boolean: ThickenBoolean,
    pub automatic_face_chain: bool,
    pub automatic_blending: bool,
    pub allow_approximation: bool,
    pub approximation_type: ThickenApproximationType,
    pub approximation_tolerance: ThickenApproximationTolerance,
    pub approximation_percent: Option<f64>,
}

impl ThickenOffsetDefinition {
    /// Validate and construct a Thicken/Offset definition from its serialized recipe.
    /// Example: `ThickenOffsetDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: ThickenOffsetRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "thicken/offset recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) =
            reference_identity(&recipe.target_body, 4, "thicken/offset body")?;
        if recipe.faces.is_empty() {
            return Err("thicken/offset faces []: expected at least one selected face".into());
        }
        let mut seen = BTreeSet::new();
        for face in &recipe.faces {
            let (face_doc, _) = reference_identity(face, 1, "thicken/offset face")?;
            if face_doc != document {
                return Err(format!(
                    "thicken/offset face document {face_doc:?}: expected body document {document:?}"
                ));
            }
            if !seen.insert(face.as_slice()) {
                return Err(format!(
                    "thicken/offset face {face:?}: expected distinct selected faces"
                ));
            }
        }
        if !recipe.distance.is_finite() {
            return Err(format!(
                "thicken/offset distance {}: expected a finite distance",
                recipe.distance
            ));
        }
        match recipe.output {
            ThickenOutputMode::Solid if recipe.distance <= 0.0 => {
                return Err(format!(
                    "thicken distance {}: expected a strictly positive solid thicken distance",
                    recipe.distance
                ));
            }
            ThickenOutputMode::Surface if recipe.distance < 0.0 => {
                return Err(format!(
                    "offset surface distance {}: expected a non-negative offset distance",
                    recipe.distance
                ));
            }
            _ => {}
        }
        if let Some(percent) = recipe.approximation_percent
            && (!percent.is_finite() || percent <= 0.0 || percent > 100.0)
        {
            return Err(format!(
                "thicken/offset approximation percent {percent}: expected a finite percentage in (0, 100]"
            ));
        }
        if recipe.allow_approximation
            && recipe.approximation_tolerance == ThickenApproximationTolerance::Specified
            && recipe.approximation_percent.is_none()
        {
            return Err(
                "thicken/offset specified tolerance: expected a positive approximation percentage"
                    .into(),
            );
        }
        Ok(Self {
            recipe,
            body_source,
        })
    }

    /// Copy the serializable recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> ThickenOffsetRecipe {
        self.recipe.clone()
    }

    /// Borrow the authored target body key; example: `definition.target_body()`.
    pub fn target_body(&self) -> &[u8] {
        &self.recipe.target_body
    }

    /// Producer feature of the target body; example: `definition.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Selected faces in authored order; example: `definition.faces()`.
    pub fn faces(&self) -> &[Vec<u8>] {
        &self.recipe.faces
    }

    /// Output mode (Solid Thicken or Surface OffsetSrf).
    pub fn output(&self) -> ThickenOutputMode {
        self.recipe.output
    }

    /// Distance in model units (cm).
    pub fn distance(&self) -> f64 {
        self.recipe.distance
    }

    /// Selected thicken/offset direction.
    pub fn direction(&self) -> ThickenDirection {
        self.recipe.direction
    }

    /// Selected solid Boolean mode.
    pub fn boolean(&self) -> ThickenBoolean {
        self.recipe.boolean
    }

    /// Whether Automatic Face Chain was enabled.
    pub fn automatic_face_chain(&self) -> bool {
        self.recipe.automatic_face_chain
    }

    /// Whether Automatic Blending was enabled.
    pub fn automatic_blending(&self) -> bool {
        self.recipe.automatic_blending
    }

    /// Whether this feature replaces the target solid body in-place.
    pub fn replaces_target_body(&self) -> bool {
        self.recipe.output == ThickenOutputMode::Solid
            && (self.recipe.automatic_blending || self.recipe.boolean != ThickenBoolean::NewSolid)
    }

    /// Attribute all read dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve authored fields for kernel evaluation; example: `definition.resolve()`.
    pub fn resolve(&self) -> ResolvedThickenOffset {
        ResolvedThickenOffset {
            faces: self.recipe.faces.clone(),
            output: self.recipe.output,
            distance: self.recipe.distance,
            direction: self.recipe.direction,
            boolean: self.recipe.boolean,
            automatic_face_chain: self.recipe.automatic_face_chain,
            automatic_blending: self.recipe.automatic_blending,
            allow_approximation: self.recipe.allow_approximation,
            approximation_type: self.recipe.approximation_type,
            approximation_tolerance: self.recipe.approximation_tolerance,
            approximation_percent: self.recipe.approximation_percent,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn key(doc: DocumentGuid, kind: u32, idx: u64) -> Vec<u8> {
        SemanticReferenceName::new(doc, 1, kind, "prism", "e", &[idx])
            .unwrap()
            .encode()
            .unwrap()
    }

    #[test]
    fn thicken_offset_validates_distance_rules_per_output_mode() {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = key(doc, 4, 1);
        let face = key(doc, 1, 2);
        let base = ThickenOffsetRecipe {
            version: 1,
            target_body: body,
            faces: vec![face],
            output: ThickenOutputMode::Solid,
            distance: 0.1,
            direction: ThickenDirection::Positive,
            boolean: ThickenBoolean::NewSolid,
            automatic_face_chain: false,
            automatic_blending: false,
            allow_approximation: false,
            approximation_type: ThickenApproximationType::Mean,
            approximation_tolerance: ThickenApproximationTolerance::Optimized,
            approximation_percent: None,
        };
        let solid = ThickenOffsetDefinition::from_recipe(base.clone()).unwrap();
        assert!(!solid.replaces_target_body());

        // Solid rejects zero and negative distances.
        assert!(
            ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
                distance: 0.0,
                ..base.clone()
            })
            .is_err()
        );
        assert!(
            ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
                distance: -0.1,
                ..base.clone()
            })
            .is_err()
        );

        // Surface accepts zero distance and rejects negative distance.
        let zero_srf = ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
            output: ThickenOutputMode::Surface,
            distance: 0.0,
            ..base.clone()
        })
        .unwrap();
        assert!(!zero_srf.replaces_target_body());
        assert!(
            ThickenOffsetDefinition::from_recipe(ThickenOffsetRecipe {
                output: ThickenOutputMode::Surface,
                distance: -0.1,
                ..base
            })
            .is_err()
        );
    }
}
