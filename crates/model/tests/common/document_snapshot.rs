// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    history::DocumentSnapshot, parameter::ParameterCollection, units::UnitsOfMeasure,
};

/// Build independent owned state for history and lifecycle tests; example: `let document = empty_snapshot();`.
pub fn empty_snapshot() -> DocumentSnapshot {
    DocumentSnapshot {
        work_planes: Default::default(),
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        parameters: ParameterCollection::new(UnitsOfMeasure::default()),
        properties: Default::default(),
    }
}
