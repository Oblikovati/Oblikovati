// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentPackage, PackageCodecError, Value};
use oblikovati_model::{
    feature::{
        FeatureSequence,
        work_plane::{WorkPlaneDefinition, WorkPlaneProgramRecipe},
    },
    identity::RecipeIdAllocator,
};
use serde::{Deserialize, Serialize};

/// Authored work planes and their permanent document-local allocation owner.
#[derive(Debug)]
pub struct RestoredWorkPlanes {
    pub program: FeatureSequence<WorkPlaneDefinition>,
    pub ids: RecipeIdAllocator,
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
struct StoredWorkPlanes {
    recipe: WorkPlaneProgramRecipe,
    feature_high_water: u64,
}
impl DocumentPackage {
    /// Restore validated recipes without cached frames; example: `package.restore_work_planes()?`.
    pub fn restore_work_planes(&self) -> Result<RestoredWorkPlanes, PackageCodecError> {
        require_legacy_features(self)?;
        let section = self
            .root
            .get("model")
            .and_then(|model| model.get("workPlanes"));
        let Some(section) = section else {
            return restore_program(FeatureSequence::default(), 0);
        };
        self.decode_work_planes(section)
    }
    fn decode_work_planes(&self, section: &Value) -> Result<RestoredWorkPlanes, PackageCodecError> {
        if self.manifest.schema_version != 3 {
            return Err(work_plane_codec_error(format!(
                "schema {}: expected schema 3",
                self.manifest.schema_version
            )));
        }
        let stored: StoredWorkPlanes =
            serde_yaml_ng::from_value(section.clone()).map_err(work_plane_codec_error)?;
        let program = FeatureSequence::from_work_plane_recipe(&stored.recipe)
            .map_err(work_plane_codec_error)?;
        restore_program(program, stored.feature_high_water)
    }
    /// Preserve complete authored programs and deleted ID reservations; example: `package.with_work_planes(&planes, mark)?`.
    pub fn with_work_planes(
        &self,
        program: &FeatureSequence<WorkPlaneDefinition>,
        mark: u64,
    ) -> Result<Self, PackageCodecError> {
        let previous = self.restore_work_planes()?;
        let retained = previous.ids.high_water();
        super::sketches::require_monotonic_mark("feature", mark, retained)?;
        restore_program(program.clone(), mark)?;
        if mark == retained && previous.program.same_work_plane_recipe(program) {
            return Ok(self.clone());
        }
        let recipe = program
            .work_plane_recipe()
            .map_err(work_plane_codec_error)?;
        self.capture_work_planes(StoredWorkPlanes {
            recipe,
            feature_high_water: mark,
        })
    }
    fn capture_work_planes(&self, stored: StoredWorkPlanes) -> Result<Self, PackageCodecError> {
        let encoded = serde_yaml_ng::to_value(stored).map_err(work_plane_codec_error)?;
        let candidate = self.with_model_section("workPlanes", encoded);
        candidate.restore_work_planes()?;
        Ok(candidate)
    }
    pub(super) fn with_model_section(&self, name: &str, contents: Value) -> Self {
        let mut candidate = self.clone();
        let mut model = self
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        model.insert(name.into(), contents);
        candidate.root.insert("model".into(), Value::Mapping(model));
        candidate.root.insert("schemaVersion".into(), 3u32.into());
        candidate.manifest.schema_version = 3;
        candidate
    }
}
fn restore_program(
    program: FeatureSequence<WorkPlaneDefinition>,
    mark: u64,
) -> Result<RestoredWorkPlanes, PackageCodecError> {
    let live: Vec<u64> = program
        .features()
        .iter()
        .map(|feature| feature.id().0)
        .collect();
    let ids = RecipeIdAllocator::restore(Some(mark), &live).map_err(work_plane_codec_error)?;
    Ok(RestoredWorkPlanes { program, ids })
}
fn require_legacy_features(package: &DocumentPackage) -> Result<(), PackageCodecError> {
    let Some(legacy) = package
        .root
        .get("model")
        .and_then(|model| model.get("features"))
    else {
        return Ok(());
    };
    if legacy.is_null() || legacy.as_sequence().is_some_and(Vec::is_empty) {
        return Ok(());
    }
    // #3729: legacy feature payloads include domains not yet represented by Rust recipes.
    Err(PackageCodecError(format!(
        "legacy model.features {legacy:?}: expected explicit legacy feature import before Rust work-plane restore or replacement"
    )))
}
fn work_plane_codec_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "model.workPlanes: expected a valid program and feature reservation mark: {error}"
    ))
}
