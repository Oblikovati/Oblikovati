// SPDX-License-Identifier: GPL-2.0-only
//! Authored Imported Model feature inputs (STL mesh and STEP BRep, Part program v50).
use crate::feature::FeatureReadFootprint;

/// Exchange file format handled by Import.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ImportedFileKind {
    #[default]
    Stl,
    Step,
}

/// STEP import mode selected in the Import dialog Options tab.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ImportMode {
    #[default]
    ConvertModel,
    ReferenceModel,
}

/// Unit interpretation choices offered in STL Import Options.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum StlImportUnits {
    #[default]
    TemplateUnits,
    Inch,
    Foot,
    Centimeter,
    Millimeter,
    Meter,
    Micron,
}

impl StlImportUnits {
    /// Ordered list of unit choices displayed by the STL Import Options combo.
    pub const ALL: [Self; 7] = [
        Self::TemplateUnits,
        Self::Inch,
        Self::Foot,
        Self::Centimeter,
        Self::Millimeter,
        Self::Meter,
        Self::Micron,
    ];

    /// Display label matching the Import specification.
    pub const fn label(self) -> &'static str {
        match self {
            Self::TemplateUnits => "Template Units",
            Self::Inch => "inch",
            Self::Foot => "foot",
            Self::Centimeter => "centimeter",
            Self::Millimeter => "millimeter",
            Self::Meter => "meter",
            Self::Micron => "micron",
        }
    }

    /// Conversion factor from one source STL unit into internal centimetres.
    pub fn scale_to_cm(self, template_working_scale_cm: f64) -> f64 {
        match self {
            Self::TemplateUnits => template_working_scale_cm,
            Self::Inch => 2.54,
            Self::Foot => 30.48,
            Self::Centimeter => 1.0,
            Self::Millimeter => 0.1,
            Self::Meter => 100.0,
            Self::Micron => 0.0001,
        }
    }
}

/// Authored STL import options persisted with an imported mesh feature.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StlImportOptions {
    pub units: StlImportUnits,
    pub template_units: String,
    pub bounding_box: [f64; 3],
    pub translation_report: bool,
    pub color_enabled: bool,
    pub color_rgb: [u8; 3],
}

impl Default for StlImportOptions {
    fn default() -> Self {
        Self {
            units: StlImportUnits::TemplateUnits,
            template_units: "millimeter".into(),
            bounding_box: [0.0; 3],
            translation_report: false,
            color_enabled: false,
            color_rgb: [192, 192, 192],
        }
    }
}

/// Authored STEP object filters and Part Options persisted with an imported STEP feature.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct StepImportOptions {
    pub import_solids: bool,
    pub import_surfaces: bool,
    pub import_wires: bool,
    pub import_points: bool,
    pub surfaces_mode: String,
}

impl Default for StepImportOptions {
    fn default() -> Self {
        Self {
            import_solids: true,
            import_surfaces: true,
            import_wires: false,
            import_points: false,
            surfaces_mode: "Composite".into(),
        }
    }
}

/// Serialized Imported Model recipe stored inside the Part feature program (v50).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ImportedModelRecipe {
    pub version: u32,
    pub source_path: String,
    pub source_name: String,
    pub file_kind: ImportedFileKind,
    pub import_mode: ImportMode,
    #[serde(default)]
    pub body_names: Vec<String>,
    #[serde(default)]
    pub link_suppressed: bool,
    #[serde(default)]
    pub link_broken: bool,
    pub unit_scale: f64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub stl_options: Option<StlImportOptions>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub step_options: Option<StepImportOptions>,
}

/// Validated authored Imported Model definition.
#[derive(Clone, Debug)]
pub struct ImportedModelDefinition {
    recipe: ImportedModelRecipe,
}

/// Resolved Imported Model inputs ready for certified Martensite evaluation.
#[derive(Clone, Debug)]
pub struct ResolvedImportedModel {
    pub source_path: String,
    pub source_name: String,
    pub file_kind: ImportedFileKind,
    pub import_mode: ImportMode,
    pub unit_scale: f64,
}

impl ImportedModelDefinition {
    /// Validate and construct an Imported Model definition from its versioned recipe.
    pub fn from_recipe(recipe: &ImportedModelRecipe) -> Result<Self, String> {
        if recipe.version != 1 {
            return Err(format!(
                "imported model version {}: expected 1",
                recipe.version
            ));
        }
        if recipe.source_path.trim().is_empty() || recipe.source_path.contains('\0') {
            return Err("imported model source_path: expected a non-empty path without NUL".into());
        }
        if recipe.source_name.trim().is_empty() || recipe.source_name.contains('\0') {
            return Err("imported model source_name: expected a non-empty name without NUL".into());
        }
        if !recipe.unit_scale.is_finite() || recipe.unit_scale <= 0.0 {
            return Err(format!(
                "imported model unit_scale {:?}: expected a positive finite scale",
                recipe.unit_scale
            ));
        }
        Ok(Self {
            recipe: recipe.clone(),
        })
    }

    pub fn recipe(&self) -> ImportedModelRecipe {
        self.recipe.clone()
    }

    pub fn source_path(&self) -> &str {
        &self.recipe.source_path
    }

    pub fn source_name(&self) -> &str {
        &self.recipe.source_name
    }

    pub fn file_kind(&self) -> ImportedFileKind {
        self.recipe.file_kind
    }

    pub fn import_mode(&self) -> ImportMode {
        self.recipe.import_mode
    }

    pub fn body_names(&self) -> &[String] {
        &self.recipe.body_names
    }

    pub fn link_suppressed(&self) -> bool {
        self.recipe.link_suppressed
    }

    pub fn link_broken(&self) -> bool {
        self.recipe.link_broken
    }

    pub fn unit_scale(&self) -> f64 {
        self.recipe.unit_scale
    }

    pub fn stl_options(&self) -> Option<&StlImportOptions> {
        self.recipe.stl_options.as_ref()
    }

    pub fn step_options(&self) -> Option<&StepImportOptions> {
        self.recipe.step_options.as_ref()
    }

    pub fn with_link_suppressed(&self, suppressed: bool) -> Self {
        let mut recipe = self.recipe.clone();
        recipe.link_suppressed = suppressed;
        Self { recipe }
    }

    pub fn with_link_broken(&self) -> Self {
        let mut recipe = self.recipe.clone();
        recipe.link_broken = true;
        recipe.link_suppressed = false;
        Self { recipe }
    }

    pub fn same_recipe(&self, other: &Self) -> bool {
        self.recipe == other.recipe
    }

    pub fn source_reads(&self) -> FeatureReadFootprint {
        FeatureReadFootprint::default()
    }

    pub fn resolve(&self) -> ResolvedImportedModel {
        ResolvedImportedModel {
            source_path: self.recipe.source_path.clone(),
            source_name: self.recipe.source_name.clone(),
            file_kind: self.recipe.file_kind,
            import_mode: self.recipe.import_mode,
            unit_scale: self.recipe.unit_scale,
        }
    }
}

impl PartialEq for ImportedModelDefinition {
    fn eq(&self, other: &Self) -> bool {
        self.same_recipe(other)
    }
}
