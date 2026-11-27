// SPDX-License-Identifier: GPL-2.0-only
//! Authored Derived Part feature inputs, separate from kernel results.
use crate::feature::FeatureReadFootprint;

/// Derive style options offered in the Derived Part Objects tab.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DeriveStyle {
    #[default]
    SingleSolid,
    SolidWithSeams,
    MaintainSolids,
    SingleComposite,
}

/// Origin mirror plane offered when Mirror part is enabled in Options.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum DeriveMirrorPlane {
    #[default]
    XY,
    YZ,
    XZ,
}

/// Serialized Derived Part recipe stored inside the Part feature program (v47).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct DerivedPartRecipe {
    pub version: u32,
    pub source_path: String,
    pub source_name: String,
    pub style: DeriveStyle,
    pub include_bodies: bool,
    pub include_sketches: bool,
    pub include_work_geometry: bool,
    pub include_parameters: bool,
    pub scale: f64,
    pub mirror: bool,
    pub mirror_plane: DeriveMirrorPlane,
    pub link_face_color: bool,
    pub link_sketch_format: bool,
    pub show_all_objects: bool,
    pub linked: bool,
    pub model_state: String,
    pub design_view: String,
}

/// Validated authored Derived Part definition.
#[derive(Clone, Debug)]
pub struct DerivedPartDefinition {
    recipe: DerivedPartRecipe,
}

/// Resolved Derived Part inputs ready for certified Martensite evaluation.
#[derive(Clone, Debug)]
pub struct ResolvedDerivedPart {
    pub source_path: String,
    pub source_name: String,
    pub style: DeriveStyle,
    pub include_bodies: bool,
    pub scale: f64,
    pub mirror: Option<DeriveMirrorPlane>,
}

impl DerivedPartDefinition {
    /// Validate and construct a Derived Part definition from its versioned recipe.
    pub fn from_recipe(recipe: &DerivedPartRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "derived part version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.source_path.trim().is_empty() || recipe.source_path.contains('\0') {
            return Err("derived part source_path: expected a non-empty path without NUL".into());
        }
        if recipe.source_name.trim().is_empty() || recipe.source_name.contains('\0') {
            return Err("derived part source_name: expected a non-empty name without NUL".into());
        }
        if !recipe.scale.is_finite() || recipe.scale <= 0.0 {
            return Err(format!(
                "derived part scale {:?}: expected a positive finite scale factor",
                recipe.scale
            ));
        }
        if recipe.model_state.contains('\0') || recipe.design_view.contains('\0') {
            return Err("derived part representation: expected text without NUL".into());
        }
        Ok(Self {
            recipe: recipe.clone(),
        })
    }

    pub fn recipe(&self) -> DerivedPartRecipe {
        self.recipe.clone()
    }

    pub fn source_path(&self) -> &str {
        &self.recipe.source_path
    }

    pub fn source_name(&self) -> &str {
        &self.recipe.source_name
    }

    pub fn style(&self) -> DeriveStyle {
        self.recipe.style
    }

    pub fn include_bodies(&self) -> bool {
        self.recipe.include_bodies
    }

    pub fn include_sketches(&self) -> bool {
        self.recipe.include_sketches
    }

    pub fn include_work_geometry(&self) -> bool {
        self.recipe.include_work_geometry
    }

    pub fn include_parameters(&self) -> bool {
        self.recipe.include_parameters
    }

    pub fn scale(&self) -> f64 {
        self.recipe.scale
    }

    pub fn mirror(&self) -> bool {
        self.recipe.mirror
    }

    pub fn mirror_plane(&self) -> DeriveMirrorPlane {
        self.recipe.mirror_plane
    }

    pub fn link_face_color(&self) -> bool {
        self.recipe.link_face_color
    }

    pub fn link_sketch_format(&self) -> bool {
        self.recipe.link_sketch_format
    }

    pub fn show_all_objects(&self) -> bool {
        self.recipe.show_all_objects
    }

    pub fn linked(&self) -> bool {
        self.recipe.linked
    }

    pub fn model_state(&self) -> &str {
        &self.recipe.model_state
    }

    pub fn design_view(&self) -> &str {
        &self.recipe.design_view
    }

    pub fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::new(std::iter::empty(), false)
    }

    pub fn resolve(&self) -> ResolvedDerivedPart {
        ResolvedDerivedPart {
            source_path: self.recipe.source_path.clone(),
            source_name: self.recipe.source_name.clone(),
            style: self.recipe.style,
            include_bodies: self.recipe.include_bodies,
            scale: self.recipe.scale,
            mirror: self.recipe.mirror.then_some(self.recipe.mirror_plane),
        }
    }

    pub fn same_recipe(&self, other: &Self) -> bool {
        self.recipe.version == other.recipe.version
            && self.recipe.source_path == other.recipe.source_path
            && self.recipe.source_name == other.recipe.source_name
            && self.recipe.style == other.recipe.style
            && self.recipe.include_bodies == other.recipe.include_bodies
            && self.recipe.include_sketches == other.recipe.include_sketches
            && self.recipe.include_work_geometry == other.recipe.include_work_geometry
            && self.recipe.include_parameters == other.recipe.include_parameters
            && self.recipe.scale.to_bits() == other.recipe.scale.to_bits()
            && self.recipe.mirror == other.recipe.mirror
            && self.recipe.mirror_plane == other.recipe.mirror_plane
            && self.recipe.link_face_color == other.recipe.link_face_color
            && self.recipe.link_sketch_format == other.recipe.link_sketch_format
            && self.recipe.show_all_objects == other.recipe.show_all_objects
            && self.recipe.linked == other.recipe.linked
            && self.recipe.model_state == other.recipe.model_state
            && self.recipe.design_view == other.recipe.design_view
    }
}

impl PartialEq for DerivedPartDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
