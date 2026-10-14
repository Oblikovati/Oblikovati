// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{
    MODEL_VALUE_LOWER, MODEL_VALUE_MEDIAN, MODEL_VALUE_NOMINAL, MODEL_VALUE_UPPER, ModelValueType,
    TOLERANCE_DEVIATION, Tolerance,
};
use oblikovati_gui::GuiParameterToleranceEdit;
fn tolerance_action(
    owner: &TestOwner,
    row: usize,
    upper: f64,
    lower: f64,
    selection: ModelValueType,
) -> GuiParameterAction {
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    GuiParameterAction::Tolerance(GuiParameterToleranceEdit {
        target: GuiParameterTextTarget {
            document: view.document,
            parameter: view.rows[row].parameter,
            field: GuiParameterTextField::Equation,
        },
        request: 1,
        selection,
        tolerance: Tolerance {
            type_: TOLERANCE_DEVIATION,
            upper,
            lower,
            ..Default::default()
        },
    })
}
#[test]
fn gui_tolerance_edits_use_working_units_keep_nominal_drivers_and_support_undo() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    native_parameter_dependency(&mut owner);
    for (selection, name, expected) in [
        (MODEL_VALUE_NOMINAL, "nominal", 4.),
        (MODEL_VALUE_UPPER, "upper", 4. + 0.2),
        (MODEL_VALUE_LOWER, "lower", 4. - 0.1),
        (MODEL_VALUE_MEDIAN, "median", 4. + (0.2 - 0.1) * 0.5),
    ] {
        let before = owner.gui_parameter_snapshot().unwrap().unwrap();
        let action = tolerance_action(&owner, 0, 0.2, -0.1, selection);
        owner.apply_gui_parameter(&action).unwrap();
        let edited = owner.gui_parameter_snapshot().unwrap().unwrap();
        assert_eq!(edited.rows[0].parameter, before.rows[0].parameter);
        assert_eq!(edited.rows[0].detail.model_value, expected);
        assert_eq!(edited.rows[0].detail.model_value_type, name);
        assert_eq!(edited.rows[0].detail.parameter_info.value, "40 mm");
        assert_eq!(edited.rows[3].detail.model_value, 8.);
        assert_eq!(
            edited.rows[0].detail.tolerance.as_deref().unwrap().upper,
            0.2
        );
        owner.files.commands(document).unwrap().undo().unwrap();
        assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
        owner.files.commands(document).unwrap().redo().unwrap();
        assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), edited);
    }
}
#[test]
fn malformed_nonnumeric_and_stale_tolerance_edits_leave_history_unchanged() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for (row, upper, lower, selection) in [
        (0, f64::NAN, 0., MODEL_VALUE_NOMINAL),
        (0, -1., 1., MODEL_VALUE_NOMINAL),
        (0, 0.2, -0.1, ModelValueType(-1)),
        (1, 0.2, -0.1, MODEL_VALUE_NOMINAL),
        (2, 0.2, -0.1, MODEL_VALUE_NOMINAL),
    ] {
        assert!(
            owner
                .apply_gui_parameter(&tolerance_action(&owner, row, upper, lower, selection))
                .is_err()
        );
        assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), before);
        assert_eq!(
            owner
                .files
                .workspace()
                .session(document)
                .unwrap()
                .generation(),
            generation
        );
    }
    let stale = tolerance_action(&owner, 0, 0.2, -0.1, MODEL_VALUE_UPPER);
    create(&mut owner, &awake);
    let active = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(
        owner
            .apply_gui_parameter(&stale)
            .unwrap_err()
            .contains("expected current document")
    );
    assert_eq!(owner.gui_parameter_snapshot().unwrap().unwrap(), active);
}
#[test]
fn signed_zero_tolerance_bounds_round_trip_through_canonical_details() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "width".into(),
            expression: "4 cm".into(),
            ..Default::default()
        })
        .unwrap();
    owner
        .apply_gui_parameter(&tolerance_action(&owner, 0, 0., -0., MODEL_VALUE_NOMINAL))
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    let id = session.snapshot().parameters.ids()[0];
    assert_eq!(
        session
            .snapshot()
            .parameters
            .get(id)
            .unwrap()
            .tolerance
            .lower
            .to_bits(),
        (-0_f64).to_bits()
    );
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    let band = view.rows[0].detail.tolerance.as_deref().unwrap();
    assert_eq!(band.upper.to_bits(), 0_f64.to_bits());
    assert_eq!(band.lower.to_bits(), (-0_f64).to_bits());
}

#[test]
fn canonical_wire_tolerance_json_omits_both_signs_of_zero() {
    for lower in [0., -0.] {
        let band = oblikovati_api::wire::ToleranceInfo {
            type_: "deviation".into(),
            lower,
            ..Default::default()
        };
        let encoded = serde_json::to_value(&band).unwrap();
        assert!(encoded.get("lower").is_none());
        assert_eq!(encoded.get("type").unwrap(), "deviation");
    }
}
#[test]
fn tolerance_gui_edits_support_angle_and_unitless_values_without_length_conversion() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    for (name, expression) in [("angle", "1 rad"), ("count", "4")] {
        Client::new(owner.document_host())
            .parameters()
            .add(&ParameterSetArgs {
                name: name.into(),
                expression: expression.into(),
                ..Default::default()
            })
            .unwrap();
    }
    for (row, nominal) in [(0, 1.), (1, 4.)] {
        owner
            .apply_gui_parameter(&tolerance_action(&owner, row, 0.2, -0.1, MODEL_VALUE_UPPER))
            .unwrap();
        let view = owner.gui_parameter_snapshot().unwrap().unwrap();
        assert_eq!(view.rows[row].detail.model_value, nominal + 0.2);
        assert_eq!(view.rows[row].detail.tolerance.as_ref().unwrap().upper, 0.2);
    }
}
#[test]
fn tolerance_rows_use_document_display_units_without_changing_stored_deviations() {
    let (mut owner, awake) = owner();
    create(&mut owner, &awake);
    add_view_parameters(&mut owner);
    let before = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert!(before.rows.iter().all(|row| row.tolerance_text.is_empty()));
    owner
        .apply_gui_parameter(&tolerance_action(&owner, 0, 0.2, -0.1, MODEL_VALUE_UPPER))
        .unwrap();
    let millimetres = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(millimetres.rows[0].tolerance_text, "2 mm / -1 mm");
    Client::new(owner.document_host())
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_unit: Some(Box::new("cm".into())),
            ..Default::default()
        })
        .unwrap();
    let centimetres = owner.gui_parameter_snapshot().unwrap().unwrap();
    assert_eq!(centimetres.rows[0].tolerance_text, "0.2 cm / -0.1 cm");
    assert_eq!(
        centimetres.rows[0].detail.tolerance,
        millimetres.rows[0].detail.tolerance
    );
    assert_eq!(
        centimetres.rows[0].detail.model_value,
        millimetres.rows[0].detail.model_value
    );
    assert_eq!(centimetres.rows[0].parameter, millimetres.rows[0].parameter);
}
