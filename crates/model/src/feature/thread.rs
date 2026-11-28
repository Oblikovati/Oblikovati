// SPDX-License-Identifier: GPL-2.0-only
//! Authored Part Modify Thread feature recipe, validation, and parameter resolution (#3918).
use super::{
    FeatureId, FeatureReadFootprint, FeatureReadKey, fillet::reference_identity,
    work_geometry::DatumLength,
};
use crate::parameter::ParameterCollection;
use std::collections::BTreeSet;

/// Helical direction of a standard thread.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ThreadHandedness {
    #[default]
    RightHand,
    LeftHand,
}

/// Authored thread standard, nominal size, designation, class, and hand.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ThreadSpecification {
    pub thread_type: String,
    pub size: String,
    pub designation: String,
    pub class: String,
    #[serde(default)]
    pub handedness: ThreadHandedness,
}

fn default_display_in_model() -> bool {
    true
}

/// Persisted Part Modify Thread recipe (Part program version 58).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ThreadRecipe {
    pub version: u32,
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub full_depth: bool,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub depth: Option<DatumLength>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub offset: Option<DatumLength>,
    #[serde(default = "default_display_in_model")]
    pub display_in_model: bool,
    pub spec: ThreadSpecification,
}

/// Validated authored Thread definition.
#[derive(Clone, Debug, PartialEq)]
pub struct ThreadDefinition {
    recipe: ThreadRecipe,
    body_source: FeatureId,
    face_source: FeatureId,
}

/// Resolved numeric Thread inputs ready for Martensite evaluation (model units: cm).
#[derive(Clone, Debug, PartialEq)]
pub struct ResolvedThread {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub full_depth: bool,
    pub depth_cm: Option<f64>,
    pub offset_cm: f64,
    pub display_in_model: bool,
    pub spec: ThreadSpecification,
}

impl ThreadDefinition {
    /// Validate and construct a Thread definition from its serialized recipe.
    /// Example: `ThreadDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: ThreadRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "thread recipe version {}: expected 1",
                recipe.version
            ));
        }
        let (document, body_source) = reference_identity(&recipe.body, 4, "thread body")?;
        let (face_document, face_source) = reference_identity(&recipe.face, 1, "thread face")?;
        if document != face_document {
            return Err(format!(
                "thread face document {face_document:?}: expected body document {document:?}"
            ));
        }
        validate_spec(&recipe.spec)?;
        if !recipe.full_depth {
            let depth = recipe
                .depth
                .ok_or_else(|| "thread depth None: expected a positive depth".to_owned())?;
            validate_positive_source(depth, "thread depth")?;
        } else if let Some(depth) = recipe.depth {
            validate_positive_source(depth, "thread depth")?;
        }
        if let Some(offset) = recipe.offset {
            validate_nonnegative_source(offset, "thread offset")?;
        }
        Ok(Self {
            recipe,
            body_source,
            face_source,
        })
    }

    /// Copy authored inputs for persistence; example: `thread.recipe()`.
    pub fn recipe(&self) -> ThreadRecipe {
        self.recipe.clone()
    }

    /// Inspect the target body reference key; example: `thread.body()`.
    pub fn body(&self) -> &[u8] {
        &self.recipe.body
    }

    /// Inspect the target cylindrical or conical face key; example: `thread.face()`.
    pub fn face(&self) -> &[u8] {
        &self.recipe.face
    }

    /// Inspect the source feature that produced the target body; example: `thread.body_source()`.
    pub fn body_source(&self) -> FeatureId {
        self.body_source
    }

    /// Collect the feature and parameter dependencies; example: `thread.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = BTreeSet::new();
        keys.insert(FeatureReadKey::Feature(self.body_source));
        keys.insert(FeatureReadKey::Feature(self.face_source));
        if let Some(DatumLength::Parameter(id)) = self.recipe.depth {
            keys.insert(FeatureReadKey::Parameter(id));
        }
        if let Some(DatumLength::Parameter(id)) = self.recipe.offset {
            keys.insert(FeatureReadKey::Parameter(id));
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Resolve current dimensions in working units (cm); example: `thread.resolve(&parameters)?`.
    pub fn resolve(&self, parameters: &ParameterCollection<u64>) -> Result<ResolvedThread, String> {
        let depth_cm = if self.recipe.full_depth {
            None
        } else {
            let depth = self
                .recipe
                .depth
                .ok_or_else(|| "thread depth None: expected a positive depth".to_owned())?;
            Some(resolve_positive(depth, parameters, "depth")?)
        };
        let offset_cm = if self.recipe.full_depth {
            0.0
        } else if let Some(offset) = self.recipe.offset {
            resolve_nonnegative(offset, parameters, "offset")?
        } else {
            0.0
        };
        Ok(ResolvedThread {
            body: self.recipe.body.clone(),
            face: self.recipe.face.clone(),
            full_depth: self.recipe.full_depth,
            depth_cm,
            offset_cm,
            display_in_model: self.recipe.display_in_model,
            spec: self.recipe.spec.clone(),
        })
    }
}

fn validate_spec(spec: &ThreadSpecification) -> Result<(), String> {
    for (label, value) in [
        ("thread type", spec.thread_type.as_str()),
        ("thread size", spec.size.as_str()),
        ("thread designation", spec.designation.as_str()),
        ("thread class", spec.class.as_str()),
    ] {
        if value.trim().is_empty() || value.contains('\0') {
            return Err(format!(
                "{label} {value:?}: expected non-empty text without NUL"
            ));
        }
    }
    Ok(())
}

fn validate_positive_source(length: DatumLength, label: &str) -> Result<(), String> {
    length
        .validate_source()
        .map_err(|cause| format!("{label} {length:?}: expected finite length source: {cause}"))?;
    if length.parameter() == Some(0) {
        return Err(format!("{label}: expected nonzero parameter id"));
    }
    if let DatumLength::Constant(v) = length
        && (!v.is_finite() || v <= 0.0)
    {
        return Err(format!("{label} {v}: expected finite positive length"));
    }
    Ok(())
}

fn validate_nonnegative_source(length: DatumLength, label: &str) -> Result<(), String> {
    length
        .validate_source()
        .map_err(|cause| format!("{label} {length:?}: expected finite length source: {cause}"))?;
    if length.parameter() == Some(0) {
        return Err(format!("{label}: expected nonzero parameter id"));
    }
    if let DatumLength::Constant(v) = length
        && (!v.is_finite() || v < 0.0)
    {
        return Err(format!("{label} {v}: expected finite nonnegative length"));
    }
    Ok(())
}

fn resolve_positive(
    length: DatumLength,
    parameters: &ParameterCollection<u64>,
    label: &str,
) -> Result<f64, String> {
    let value = length
        .resolve(parameters)
        .map_err(|cause| format!("thread {label} {length:?}: {cause}"))?;
    if !value.is_finite() || value <= 0.0 {
        return Err(format!(
            "thread {label} {value}: expected finite positive length"
        ));
    }
    Ok(value)
}

fn resolve_nonnegative(
    length: DatumLength,
    parameters: &ParameterCollection<u64>,
    label: &str,
) -> Result<f64, String> {
    let value = length
        .resolve(parameters)
        .map_err(|cause| format!("thread {label} {length:?}: {cause}"))?;
    if !value.is_finite() || value < 0.0 {
        return Err(format!(
            "thread {label} {value}: expected finite nonnegative length"
        ));
    }
    Ok(value)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::identity::{DocumentGuid, SemanticReferenceName};

    fn sample_keys() -> (Vec<u8>, Vec<u8>) {
        let doc: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let body = SemanticReferenceName::new(doc, 1, 4, "revolve", "body", &[1])
            .unwrap()
            .encode()
            .unwrap();
        let face = SemanticReferenceName::new(doc, 1, 1, "revolve", "side", &[1])
            .unwrap()
            .encode()
            .unwrap();
        (body, face)
    }

    fn sample_spec() -> ThreadSpecification {
        ThreadSpecification {
            thread_type: "ANSI Metric M Profile".into(),
            size: "9".into(),
            designation: "M9x1.25".into(),
            class: "6H".into(),
            handedness: ThreadHandedness::RightHand,
        }
    }

    #[test]
    fn validates_partial_depth_and_offset_resolution() {
        let (body, face) = sample_keys();
        let recipe = ThreadRecipe {
            version: 1,
            body: body.clone(),
            face: face.clone(),
            full_depth: false,
            depth: Some(DatumLength::Constant(2.0)),
            offset: Some(DatumLength::Constant(9.9)),
            display_in_model: true,
            spec: sample_spec(),
        };
        let def = ThreadDefinition::from_recipe(recipe).unwrap();
        let params = ParameterCollection::new(crate::units::UnitsOfMeasure::default());
        let resolved = def.resolve(&params).unwrap();
        assert!(!resolved.full_depth);
        assert_eq!(resolved.depth_cm, Some(2.0));
        assert_eq!(resolved.offset_cm, 9.9);
        assert!(resolved.display_in_model);
    }

    #[test]
    fn rejects_zero_partial_depth() {
        let (body, face) = sample_keys();
        let recipe = ThreadRecipe {
            version: 1,
            body,
            face,
            full_depth: false,
            depth: Some(DatumLength::Constant(0.0)),
            offset: Some(DatumLength::Constant(0.0)),
            display_in_model: true,
            spec: sample_spec(),
        };
        assert!(ThreadDefinition::from_recipe(recipe).is_err());
    }
}
