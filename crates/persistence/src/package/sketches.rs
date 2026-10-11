// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentPackage, PackageCodecError};
use oblikovati_model::history::{SketchConstraintReservations, SketchReservationMarks};
use oblikovati_model::{
    document_sketch::{DocumentSketchRecipe, DocumentSketches},
    identity::RecipeIdAllocator,
};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
mod reservations;
/// Validated sketch contents and permanent allocation owners for the document.
#[derive(Debug)]
pub struct RestoredDocumentSketches {
    pub sketches: DocumentSketches,
    pub sketch_ids: RecipeIdAllocator,
    pub entity_ids: RecipeIdAllocator,
    pub constraint_ids: SketchConstraintReservations,
}
#[derive(Deserialize, Serialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
struct StoredSketchCollection {
    recipe: DocumentSketchRecipe,
    sketch_high_water: u64,
    entity_high_water: u64,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        deserialize_with = "restore_optional_constraint_marks"
    )]
    constraint_high_water: Option<BTreeMap<u64, u64>>,
}
impl DocumentPackage {
    /// Restore schema-3 placed sketches and ID reservations; example: `let restored = package.restore_sketches()?;`.
    pub fn restore_sketches(&self) -> Result<RestoredDocumentSketches, PackageCodecError> {
        require_legacy_import(self)?;
        let Some(section) = self
            .root
            .get("model")
            .and_then(|model| model.get("planarSketches"))
        else {
            return restore_collection(DocumentSketches::default(), 0, 0, None);
        };
        if self.manifest.schema_version != 3 {
            return Err(PackageCodecError(format!(
                "model.planarSketches in schema {}: expected document schema 3",
                self.manifest.schema_version
            )));
        }
        let stored: StoredSketchCollection =
            serde_yaml_ng::from_value(section.clone()).map_err(sketch_codec_error)?;
        let sketches = DocumentSketches::from_recipe(&stored.recipe).map_err(sketch_codec_error)?;
        restore_collection(
            sketches,
            stored.sketch_high_water,
            stored.entity_high_water,
            stored.constraint_high_water.as_ref(),
        )
    }
    /// Capture sketches without dropping other model sections; example: `package.with_sketches(&sketches, sketch_mark, entity_mark)?`.
    pub fn with_sketches(
        &self,
        sketches: &DocumentSketches,
        sketch_high_water: u64,
        entity_high_water: u64,
    ) -> Result<Self, PackageCodecError> {
        let marks = self.initial_sketch_marks(sketches, sketch_high_water, entity_high_water)?;
        self.with_sketch_reservations(sketches, marks)
    }
    fn capture_sketches(&self, stored: StoredSketchCollection) -> Result<Self, PackageCodecError> {
        let encoded = serde_yaml_ng::to_value(stored).map_err(sketch_codec_error)?;
        let candidate = self.with_model_section("planarSketches", encoded);
        candidate.restore_sketches()?;
        Ok(candidate)
    }
}
fn restore_collection(
    sketches: DocumentSketches,
    sketch_mark: u64,
    entity_mark: u64,
    constraint_marks: Option<&BTreeMap<u64, u64>>,
) -> Result<RestoredDocumentSketches, PackageCodecError> {
    let sketch_ids = RecipeIdAllocator::restore(Some(sketch_mark), &sketches.sketch_ids())
        .map_err(sketch_codec_error)?;
    let entity_ids = RecipeIdAllocator::restore(Some(entity_mark), &sketches.entity_ids())
        .map_err(sketch_codec_error)?;
    let constraint_ids =
        SketchConstraintReservations::restore(&sketches, sketch_mark, constraint_marks)
            .map_err(sketch_codec_error)?;
    Ok(RestoredDocumentSketches {
        sketches,
        sketch_ids,
        entity_ids,
        constraint_ids,
    })
}
pub(super) fn require_monotonic_mark(
    kind: &str,
    supplied: u64,
    retained: u64,
) -> Result<(), PackageCodecError> {
    if supplied < retained {
        return Err(PackageCodecError(format!(
            "{kind} reservation mark {supplied}: expected at least retained mark {retained}"
        )));
    }
    Ok(())
}
fn require_legacy_import(package: &DocumentPackage) -> Result<(), PackageCodecError> {
    let Some(legacy) = package
        .root
        .get("model")
        .and_then(|model| model.get("sketches"))
    else {
        return Ok(());
    };
    if legacy.is_null() || legacy.as_sequence().is_some_and(Vec::is_empty) {
        return Ok(());
    }
    // #3719: retain raw legacy recipes until an explicit importer covers their full domains.
    Err(PackageCodecError(format!(
        "legacy model.sketches {legacy:?}: expected explicit legacy sketch import before Rust sketch restore or replacement"
    )))
}
fn sketch_codec_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "model.planarSketches: expected a valid placed recipe and reservation marks: {error}"
    ))
}
fn restore_optional_constraint_marks<'de, D: serde::Deserializer<'de>>(
    source: D,
) -> Result<Option<BTreeMap<u64, u64>>, D::Error> {
    let value = serde_yaml_ng::Value::deserialize(source)?;
    if !value.is_mapping() {
        return Err(serde::de::Error::custom(format!(
            "constraintHighWater {value:?}: expected a mapping of sketch IDs to constraint marks"
        )));
    }
    serde_yaml_ng::from_value(value)
        .map(Some)
        .map_err(serde::de::Error::custom)
}
