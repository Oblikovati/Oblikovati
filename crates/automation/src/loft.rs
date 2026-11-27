// SPDX-License-Identifier: GPL-2.0-only
//! Prepare a Loft from source identities shared by preview and commit.
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        sketch_loft::{
            SketchLoftCenterLine, SketchLoftDefinition, SketchLoftOutput, SketchLoftPlacedSection,
            SketchLoftSection,
        },
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// A named Loft with ordered, structurally valid source identities.
pub struct PreparedLoft {
    name: String,
    definition: SketchLoftDefinition,
}

/// Prepare a Loft from two sketch circles. Example: `prepare_loft(snapshot, sections, output, None)?`.
pub fn prepare_loft(
    snapshot: &DocumentSnapshot,
    sections: [SketchLoftSection; 2],
    output: SketchLoftOutput,
    center_line: Option<SketchLoftCenterLine>,
) -> Result<PreparedLoft, Error> {
    let prefix = match output {
        SketchLoftOutput::Solid => "Loft",
        SketchLoftOutput::Surface => "LoftSrf",
    };
    let definition = SketchLoftDefinition::new(sections)
        .map_err(Error)?
        .with_output(output);
    let definition = match center_line {
        Some(path) => definition.with_center_line(path).map_err(Error)?,
        None => definition,
    };
    Ok(PreparedLoft {
        name: crate::fillet::next_name(snapshot, prefix)?,
        definition,
    })
}

impl PreparedLoft {
    /// Append the Loft to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_sketch_loft(&self.name, self.definition, ids)
            .map_err(|cause| Error(cause.to_string()))
    }
}

/// Resolve the current Loft definition by identity. Example: `loft_definition(snapshot, feature)?`.
pub fn loft_definition(
    snapshot: &DocumentSnapshot,
    feature: FeatureId,
) -> Result<&SketchLoftDefinition, Error> {
    let authored = snapshot
        .work_geometry
        .get(feature)
        .map_err(|cause| Error(cause.to_string()))?;
    let oblikovati_model::feature::part::PartFeatureDefinition::SketchLoft(definition) =
        authored.definition()
    else {
        return Err(Error(format!("feature {feature:?}: expected a Loft")));
    };
    Ok(definition)
}

/// Replace the intermediate section in a private candidate. Example: `replace_loft_section(snapshot, feature, section)?`.
pub fn replace_loft_section(
    snapshot: &mut DocumentSnapshot,
    feature: FeatureId,
    placed: SketchLoftPlacedSection,
) -> Result<FeatureId, Error> {
    let definition = loft_definition(snapshot, feature)?
        .clone()
        .with_placed_section(Some(placed))
        .map_err(Error)?;
    let reads = definition.reads(snapshot).map_err(Error)?;
    snapshot
        .work_geometry
        .replace(
            feature,
            oblikovati_model::feature::part::PartFeatureDefinition::SketchLoft(Box::new(
                definition,
            )),
            reads,
        )
        .map_err(|cause| Error(cause.to_string()))?;
    Ok(feature)
}
