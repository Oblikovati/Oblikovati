// SPDX-License-Identifier: GPL-2.0-only

//! Typed parameter ownership, evaluation health, and deterministic dependency edits.
mod annotations;
mod authored;
mod conversion;
mod copy;
mod presentation;
mod property;
pub mod vocabulary;
pub use annotations::ParameterAnnotations;
pub use presentation::{dimension_display_name, kind_name, parse_dimension_display};
pub use property::CustomPropertyFormat;
mod choices;
mod comparison;
mod derived;
pub use derived::{DerivedTable, DerivedTableHealth};
mod edit;
mod error;
mod evaluate;
mod fits;
mod groups;
mod membership;
pub use groups::ParameterGroup;
mod literal;
mod reads;
mod record;
mod remove;
mod rename;
mod service;
mod settings;
mod state;
pub use settings::CollectionSettings;
mod sweep;
mod tolerance;
pub use reads::ParameterReads;

use crate::{
    dependency::{DependencyError, DependencyGraph},
    expression::{BoundExpression, Expression},
    units::UnitsOfMeasure,
};
pub use error::ParameterError;
use oblikovati_api::types::{
    DERIVED_PARAM, MODEL_PARAM, ParameterKind, REFERENCE_PARAM, TABLE_PARAM, USER_PARAM,
};
pub use state::{ParameterFailure, ParameterHealth, ParameterState, ParameterValue};
use std::{
    collections::{BTreeMap, BTreeSet},
    fmt::Debug,
};

use record::{ParameterFormula, ParameterRecord};

/// Own typed parameters under IDs supplied by the document's identity allocator.
#[derive(Clone, Debug)]
pub struct ParameterCollection<Id> {
    records: BTreeMap<Id, ParameterRecord<Id>>,
    names: BTreeMap<String, Id>,
    order: Vec<Id>,
    graph: DependencyGraph<Id>,
    changed: BTreeSet<Id>,
    units: UnitsOfMeasure,
    settings: CollectionSettings,
    groups: Vec<ParameterGroup<Id>>,
    derived_tables: Vec<DerivedTable<Id>>,
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Inspect the unit context owned by this recipe; example: `parameters.units().clone()`.
    pub fn units(&self) -> &UnitsOfMeasure {
        &self.units
    }

    /// Replace display preferences without rescaling existing quantities.
    /// Example: `parameters.set_unit_preferences(preferences)?` after editing a clone of `units()`.
    pub fn set_unit_preferences(
        &mut self,
        units: UnitsOfMeasure,
    ) -> Result<(), ParameterError<Id>> {
        if units.working_scale() != self.units.working_scale() {
            return Err(ParameterError::Settings {
                found: format!("working scale {}", units.working_scale()),
                expected: "the unchanged document working scale",
            });
        }
        self.units = units;
        Ok(())
    }

    /// Create a collection with fixed working units; for example `ParameterCollection::<u64>::new(units)`.
    pub fn new(units: UnitsOfMeasure) -> Self {
        Self {
            records: BTreeMap::new(),
            names: BTreeMap::new(),
            order: Vec::new(),
            graph: DependencyGraph::default(),
            changed: BTreeSet::new(),
            units,
            settings: CollectionSettings::default(),
            groups: Vec::new(),
            derived_tables: Vec::new(),
        }
    }

    /// Inspect a parameter without bypassing edit rules; for example `parameters.get(id)`.
    pub fn get(&self, id: Id) -> Option<&ParameterState<Id>> {
        self.records.get(&id).map(|record| &record.state)
    }

    /// Resolve a display name to its stable ID; for example `parameters.find("width")`.
    pub fn find(&self, name: &str) -> Option<Id> {
        self.names.get(name).copied()
    }

    /// Enumerate IDs in insertion order; for example `parameters.ids()`.
    pub fn ids(&self) -> &[Id] {
        &self.order
    }

    /// Query current dependency edges; for example `parameters.drivers(height)`.
    pub fn drivers(&self, id: Id) -> Result<Vec<Id>, ParameterError<Id>> {
        self.graph.drivers(id).map_err(Into::into)
    }

    /// Query current readers in insertion order; for example `parameters.readers(width)`.
    pub fn readers(&self, id: Id) -> Result<Vec<Id>, ParameterError<Id>> {
        self.graph.readers(id).map_err(Into::into)
    }

    /// Drain edited/recomputed IDs in insertion order; for example `let dirty = parameters.drain_changed();`.
    pub fn drain_changed(&mut self) -> Vec<Id> {
        let changed = self
            .order
            .iter()
            .copied()
            .filter(|id| self.changed.contains(id))
            .collect();
        self.changed.clear();
        changed
    }

    fn require_record(&self, id: Id) -> Result<&ParameterRecord<Id>, ParameterError<Id>> {
        self.records
            .get(&id)
            .ok_or(DependencyError::Missing(id).into())
    }

    fn require_editable(&self, id: Id) -> Result<(), ParameterError<Id>> {
        let record = self.require_record(id)?;
        if !editable(record.state.kind) {
            return Err(ParameterError::ReadOnly {
                id,
                name: record.state.name.clone(),
            });
        }
        Ok(())
    }
}

fn editable(kind: ParameterKind) -> bool {
    matches!(kind, MODEL_PARAM | USER_PARAM | TABLE_PARAM)
}

fn require_kind<Id>(kind: ParameterKind) -> Result<(), ParameterError<Id>> {
    if !matches!(
        kind,
        MODEL_PARAM | USER_PARAM | TABLE_PARAM | REFERENCE_PARAM | DERIVED_PARAM
    ) {
        return Err(ParameterError::Kind {
            kind,
            expected: "Model, User, Table, Reference, or Derived",
        });
    }
    Ok(())
}
