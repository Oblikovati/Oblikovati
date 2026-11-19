// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    parameter::ParameterCollection, property::DocumentProperties, units::UnitsOfMeasure,
};

fn document_with_depth() -> DocumentSnapshot {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "depth", USER_PARAM, "8 mm")
        .unwrap();
    parameters
        .insert_expression(2, "angle", USER_PARAM, "2 deg")
        .unwrap();
    DocumentSnapshot {
        parameters,
        properties: DocumentProperties::default(),
        assembly: None,
        drawing: None,
        model_states: None,
        body_names: Default::default(),
        design_views: Default::default(),
        part_design_views: None,
        work_geometry: Default::default(),
        assets: Default::default(),
        sketch_settings: None,
        sketches: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    }
}

#[test]
fn draft_validation_uses_the_feature_distance_rules_without_edits() {
    let snapshot = document_with_depth();
    for source in ["8", "8 mm", "1 mm + 2 mm", "depth"] {
        assert!(
            validate_extrude_distance(&snapshot, source).is_ok(),
            "{source:?}"
        );
    }
    for source in ["", "0", "-1 mm", "2 deg", "angle", "missing", "depth * 2"] {
        assert!(
            validate_extrude_distance(&snapshot, source).is_err(),
            "{source:?}"
        );
    }
    assert!(matches!(
        distance(&snapshot, "depth"),
        Ok(DatumLength::Parameter(1))
    ));
    assert_eq!(snapshot.parameters.ids(), &[1, 2]);
}
