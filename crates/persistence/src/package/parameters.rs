// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use oblikovati_model::{
    identity::RecipeIdAllocator,
    parameter::{ParameterCollection, ParameterHealth},
    units::UnitsOfMeasure,
};

mod derived;
mod diagnostics;
mod export;
mod groups;
mod identities;
mod measured;
mod property;
mod record;
mod reservations;
mod settings;
mod tolerance;
use record::ParameterRecipe;

/// A reconstructed table and its document-owned identity allocator.
#[derive(Debug)]
pub struct RestoredParameters {
    pub parameters: ParameterCollection<u64>,
    pub ids: RecipeIdAllocator,
    pub derived_high_water: i64,
}

impl DocumentPackage {
    /// Restore parameters with the working scale and preferences saved in this document.
    /// Example: `let restored = package.restore_document_parameters()?;`.
    pub fn restore_document_parameters(&self) -> Result<RestoredParameters, PackageCodecError> {
        self.restore_parameters(self.document_units()?)
    }

    /// Build a new parameter table; errors never modify an existing live table.
    /// The caller supplies working units from the enclosing document recipe.
    /// Example: `let restored = package.restore_parameters(document_units)?;`.
    pub fn restore_parameters(
        &self,
        units: UnitsOfMeasure,
    ) -> Result<RestoredParameters, PackageCodecError> {
        let records = self.parameter_records()?;
        if self.manifest.schema_version < 3
            && records
                .iter()
                .any(|record| record.expression_context.is_some())
        {
            return Err(PackageCodecError(format!(
                "scalar expressionContext in document schema {}: expected document schema 3",
                self.manifest.schema_version
            )));
        }
        let (live_ids, allocator) = identities::restore_ids(self, &records)?;
        let mut restored = RestoredParameters {
            parameters: ParameterCollection::new(units),
            ids: allocator,
            derived_high_water: 0,
        };
        for (record, id) in records.iter().zip(live_ids) {
            record.restore(id, &mut restored.parameters)?;
        }

        groups::restore_groups(self.root.get("model"), &mut restored.parameters)?;
        settings::restore_settings(self.root.get("model"), &mut restored.parameters)?;
        derived::restore_tables(
            self.root.get("model"),
            self.manifest.schema_version,
            &mut restored.parameters,
        )?;
        validate_health(&restored.parameters, self.manifest.schema_version >= 3)?;
        tolerance::restore_tolerances(
            &records,
            &mut restored.parameters,
            self.manifest.schema_version,
        )?;
        restored.derived_high_water = reservations::restore_mark(self, &restored.parameters)?;
        restored.parameters.drain_changed();
        Ok(restored)
    }

    fn parameter_records(&self) -> Result<Vec<ParameterRecipe>, PackageCodecError> {
        let Some(model) = self.root.get("model").and_then(Value::as_mapping) else {
            return Ok(Vec::new());
        };
        match model.get("parameters") {
            None | Some(Value::Null) => Ok(Vec::new()),
            Some(records) => serde_yaml_ng::from_value(records.clone()).map_err(codec_error),
        }
    }
}

fn validate_health(
    parameters: &ParameterCollection<u64>,
    allow_failure: bool,
) -> Result<(), PackageCodecError> {
    for id in parameters.ids() {
        let state = parameters.get(*id).expect("owned parameter ID");
        let cycle = matches!(
            state.health,
            ParameterHealth::Failed(oblikovati_model::parameter::ParameterFailure::Cycle(_))
        );
        if cycle || (!allow_failure && !matches!(state.health, ParameterHealth::Healthy)) {
            return Err(PackageCodecError(format!(
                "restored parameter {:?} has {:?}: expected resolved, finite, acyclic values",
                state.name, state.health
            )));
        }
    }
    Ok(())
}

fn parameter_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!("parameter recipe: {error}"))
}
