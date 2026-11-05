// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::types::*;
use oblikovati_model::{
    parameter::{CollectionSettings, ParameterCollection},
    units::UnitsOfMeasure,
};

#[test]
fn settings_defaults_match_the_go_document() {
    let parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let settings = parameters.settings();
    assert_eq!(settings.linear_dimension_precision, 3);
    assert_eq!(settings.angular_dimension_precision, 2);
    assert_eq!(settings.dimension_display_type, DIMENSION_DISPLAY_VALUE);
    assert!(settings.linear_standard_tolerance.is_empty());
    assert!(settings.angular_standard_tolerance.is_empty());
    assert!(!settings.use_standard_tolerances && !settings.export_standard_tolerances);
    assert!(!settings.display_parameter_as_expression);
}

#[test]
fn all_modes_and_precision_boundaries_preserve_metadata_without_recomputing_values() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "depth", USER_PARAM, "10 mm")
        .unwrap();
    parameters.set_symmetric(1, 0.02).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    parameters.drain_changed();
    let before = parameters.get(1).unwrap().clone();
    for mode in [
        DIMENSION_DISPLAY_VALUE,
        DIMENSION_DISPLAY_NAME,
        DIMENSION_DISPLAY_EXPRESSION,
        DIMENSION_DISPLAY_TOLERANCE,
        DIMENSION_DISPLAY_PRECISE_VALUE,
    ] {
        check_settings_edit(&mut parameters, mode);
        assert_eq!(parameters.get(1), Some(&before));
        assert_eq!(parameters.model_value(1).unwrap().value, 1.02);
        assert!(parameters.drain_changed().is_empty());
    }
}

fn check_settings_edit(parameters: &mut ParameterCollection<u64>, mode: DimensionDisplayType) {
    for precision in [0, 8] {
        let settings = CollectionSettings {
            linear_standard_tolerance: "0.1 mm".into(),
            angular_standard_tolerance: "0.5 deg".into(),
            use_standard_tolerances: true,
            export_standard_tolerances: true,
            linear_dimension_precision: precision,
            angular_dimension_precision: precision,
            dimension_display_type: mode,
            display_parameter_as_expression: true,
        };
        parameters.set_settings(settings.clone()).unwrap();
        assert_eq!(parameters.settings(), &settings);
    }
}

#[test]
fn invalid_settings_cannot_partially_change_flags_or_tolerance_strings() {
    let mut parameters = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let before = parameters.settings().clone();
    let changed = CollectionSettings {
        use_standard_tolerances: true,
        linear_standard_tolerance: "changed".into(),
        ..before.clone()
    };
    for invalid in [
        CollectionSettings {
            linear_dimension_precision: -1,
            ..changed.clone()
        },
        CollectionSettings {
            angular_dimension_precision: 9,
            ..changed.clone()
        },
        CollectionSettings {
            dimension_display_type: DimensionDisplayType(0),
            ..changed
        },
    ] {
        let error = parameters.set_settings(invalid).unwrap_err().to_string();
        assert!(error.contains("found") && error.contains("expected"));
        assert_eq!(parameters.settings(), &before);
        assert!(parameters.drain_changed().is_empty());
    }
}

#[test]
fn collection_clones_own_their_settings_independently() {
    let original = ParameterCollection::<u64>::new(UnitsOfMeasure::default());
    let mut clone = original.clone();
    let mut settings = clone.settings().clone();
    settings.linear_standard_tolerance = "authored expression retained as metadata".into();
    settings.use_standard_tolerances = true;
    clone.set_settings(settings).unwrap();
    assert_eq!(original.settings(), &CollectionSettings::default());
    assert!(clone.settings().use_standard_tolerances);
    clone.set_settings(CollectionSettings::default()).unwrap();
    assert_eq!(clone.settings(), original.settings());
}
