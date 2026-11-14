// SPDX-License-Identifier: GPL-2.0-only
//! Preparation only: feature adoption, reference-key migration, and package migration remain separate.
mod source;
use super::{DocumentPackage, LegacySketchCandidate, PackageCodecError};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        sketch_prism::{SketchPrismDefinition, SketchPrismSelection},
        work_geometry::DatumLength,
    },
    sketch::OwnedSketchGeometry,
};
use source::{LegacyExtrude, LegacyFeature};

/// Resolve a Go profile index to its exact authored boundaries, refusing ambiguity.
pub trait LegacyProfileResolver {
    /// Resolve the saved index using verified Go-compatible profile semantics.
    /// Example: `resolver.resolve(&candidate, sketch, 0)?`.
    fn resolve(
        &self,
        candidate: &LegacySketchCandidate,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String>;
}
/// One source feature and its validated native prism definition, without a new feature identity.
#[derive(Debug)]
pub struct LegacyExtrusionCandidate {
    pub name: String,
    pub creation_sequence: u64,
    pub suppressed: bool,
    pub definition: SketchPrismDefinition,
}
/// An isolated candidate; it is not a migrated or open document.
#[derive(Debug)]
pub struct LegacyPartCandidate {
    pub sketches: LegacySketchCandidate,
    pub extrusions: Vec<LegacyExtrusionCandidate>,
}
impl DocumentPackage {
    /// Prepare supported sketches and extrusions without adopting or modifying the package.
    /// Example: `let candidate = package.prepare_legacy_part_candidate(&resolver)?;`.
    pub fn prepare_legacy_part_candidate(
        &self,
        resolver: &dyn LegacyProfileResolver,
    ) -> Result<LegacyPartCandidate, PackageCodecError> {
        require_supported_model(self.root.get("model"))?;
        let sketches = self.prepare_legacy_sketch_candidate()?;
        let source = self
            .root
            .get("model")
            .and_then(|model| model.get("features"));
        let rows: Vec<LegacyFeature> = match source {
            None | Some(serde_yaml_ng::Value::Null) => Vec::new(),
            Some(value) => serde_yaml_ng::from_value(value.clone()).map_err(failure)?,
        };
        let extrusions = rows
            .into_iter()
            .map(|row| prepare_extrusion(&sketches, row, resolver))
            .collect::<Result<_, _>>()?;
        Ok(LegacyPartCandidate {
            sketches,
            extrusions,
        })
    }
}
fn require_supported_model(model: Option<&serde_yaml_ng::Value>) -> Result<(), PackageCodecError> {
    let Some(model) = model else {
        return Ok(());
    };
    if model.is_null() {
        return Ok(());
    }
    let fields = model
        .as_mapping()
        .ok_or_else(|| failure(format!("model {model:?}: expected a mapping")))?;
    for key in fields.keys() {
        if !matches!(
            key.as_str(),
            Some("units" | "parameters" | "sketches" | "features")
        ) {
            return Err(failure(format!(
                "model section {key:?}: expected a section supported by this candidate"
            )));
        }
    }
    Ok(())
}
fn failure(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!(
        "legacy model.features: expected a supported, lossless extrusion recipe: {error}"
    ))
}
fn prepare_extrusion(
    sketches: &LegacySketchCandidate,
    row: LegacyFeature,
    resolver: &dyn LegacyProfileResolver,
) -> Result<LegacyExtrusionCandidate, PackageCodecError> {
    if row.kind != "extrude" {
        return Err(failure(format!(
            "feature kind {:?}: expected extrude",
            row.kind
        )));
    }
    let index = profile_index(&row.extrude)?;
    let displacement = distance(&row.extrude)?;
    let sketch = sketches
        .sketches
        .sketches()
        .get(row.extrude.sketch)
        .ok_or_else(|| {
            failure(format!(
                "sketch index {}: expected an existing sketch",
                row.extrude.sketch
            ))
        })?
        .id();
    let selection = resolver.resolve(sketches, sketch, index).map_err(failure)?;
    validate_boundary_ownership(sketches, sketch, &selection)?;
    let definition = SketchPrismDefinition::new(
        sketch,
        selection,
        [
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Constant(displacement),
        ],
    )
    .map_err(failure)?;
    Ok(LegacyExtrusionCandidate {
        name: row.name,
        creation_sequence: row.seq,
        suppressed: row.suppressed,
        definition,
    })
}
fn validate_boundary_ownership(
    sketches: &LegacySketchCandidate,
    sketch: DocumentSketchId,
    selection: &SketchPrismSelection,
) -> Result<(), PackageCodecError> {
    let geometry = sketches.sketches.get(sketch).map_err(failure)?.geometry();
    let index = geometry.read_index();
    for line in selection
        .outer
        .iter()
        .chain(selection.holes.iter().flatten())
    {
        let entity = index.entity(line.curve).map_err(failure)?;
        if !matches!(entity, OwnedSketchGeometry::Line(_)) {
            return Err(failure(format!(
                "profile entity {:?}: expected a line in sketch {sketch:?}",
                line.curve
            )));
        }
    }
    Ok(())
}
fn profile_index(extrude: &LegacyExtrude) -> Result<usize, PackageCodecError> {
    match (extrude.profile, extrude.profiles.as_slice()) {
        (None, []) => Ok(0),
        (Some(index), []) => Ok(index),
        (None, [index]) => Ok(*index),
        (old, current) => Err(failure(format!(
            "profile selection {old:?} / {current:?}: expected one unambiguous profile index"
        ))),
    }
}
fn distance(extrude: &LegacyExtrude) -> Result<f64, PackageCodecError> {
    if extrude.operation != "newBody"
        || !matches!(extrude.extent.as_str(), "" | "distance")
        || !matches!(extrude.direction.as_str(), "" | "positive")
    {
        return Err(failure(format!(
            "operation {:?}, extent {:?}, direction {:?}: expected newBody with positive distance",
            extrude.operation, extrude.extent, extrude.direction
        )));
    }
    if !extrude.distance.is_finite() || extrude.distance <= 0.0 {
        return Err(failure(format!(
            "distance {:?}: expected a finite positive distance",
            extrude.distance
        )));
    }
    Ok(extrude.distance)
}
