// SPDX-License-Identifier: GPL-2.0-only
//! Prepare a Loft from source identities shared by preview and commit.
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        sketch_loft::{SketchLoftDefinition, SketchLoftSection},
    },
    history::{DocumentSnapshot, RecipeReservations},
};

/// A named Loft with ordered, structurally valid source identities.
pub struct PreparedLoft {
    name: String,
    definition: SketchLoftDefinition,
}

/// Prepare a new solid from two sketch circles. Example: `prepare_loft(snapshot, sections)?`.
pub fn prepare_loft(
    snapshot: &DocumentSnapshot,
    sections: [SketchLoftSection; 2],
) -> Result<PreparedLoft, Error> {
    Ok(PreparedLoft {
        name: crate::fillet::next_name(snapshot, "Loft")?,
        definition: SketchLoftDefinition::new(sections).map_err(Error)?,
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
