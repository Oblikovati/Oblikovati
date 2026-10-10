// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;

fn tolerance_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "band", USER_PARAM, "1 mm")
        .unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    parameters
}

#[test]
fn tolerance_bands_resolve_formulas_literals_and_absolute_limits_with_undo() {
    let (mut workspace, id) = parameter_workspace(tolerance_parameters());
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for (mode, upper, lower, kind, expected_upper, expected_lower) in [
            ("symmetric", "band * 2", "", "symmetric", 0.2, -0.2),
            ("deviation", "3", "-1 mm", "deviation", 0.3, -0.1),
            (
                "limits",
                "width + band",
                "3.8 cm",
                "limitsStacked",
                0.1,
                -0.2,
            ),
        ] {
            let detail = client
                .parameters()
                .set_tolerance(&ParameterToleranceArgs {
                    name: "width".into(),
                    mode: mode.into(),
                    upper: upper.into(),
                    lower: lower.into(),
                    ..Default::default()
                })
                .unwrap();
            let tolerance = detail.tolerance.unwrap();
            assert_eq!(tolerance.type_, kind);
            assert!((tolerance.upper - expected_upper).abs() < 1e-14);
            assert!((tolerance.lower - expected_lower).abs() < 1e-14);
            assert!((detail.model_value - (4.0 + expected_upper)).abs() < 1e-14);
            assert_eq!(detail.parameter_info.expression, "4 cm");
        }
    }
    assert_eq!(
        workspace.session(id).unwrap().undo_labels(),
        ["Edit Tolerance"; 3]
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .tolerance
            .type_,
        TOLERANCE_DEVIATION
    );
    workspace.commands(id).unwrap().redo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .tolerance
            .type_,
        TOLERANCE_LIMITS_STACKED
    );
}

#[test]
fn fits_and_bandless_modes_replace_the_complete_tolerance() {
    let (mut workspace, _) = parameter_workspace(tolerance_parameters());
    let client = Client::new(SessionHost::new(&mut workspace));
    let detail = client
        .parameters()
        .set_tolerance(&ParameterToleranceArgs {
            name: "width".into(),
            mode: "fits".into(),
            hole: "H7".into(),
            shaft: "g6".into(),
            ..Default::default()
        })
        .unwrap();
    let tolerance = detail.tolerance.unwrap();
    assert_eq!(tolerance.type_, "limitsFitsStacked");
    assert_eq!(tolerance.hole_tolerance, "H7");
    assert_eq!(tolerance.shaft_tolerance, "g6");
    assert!((tolerance.upper - 0.0025).abs() < 1e-14);
    assert_eq!(tolerance.lower, 0.0);
    for mode in ["min", "max", "basic", "reference", "default"] {
        let detail = client
            .parameters()
            .set_tolerance(&ParameterToleranceArgs {
                name: "width".into(),
                mode: mode.into(),
                ..Default::default()
            })
            .unwrap();
        let tolerance = detail.tolerance.unwrap();
        assert_eq!(tolerance.type_, mode);
        assert_eq!((tolerance.upper, tolerance.lower), (0.0, 0.0));
        assert!(tolerance.hole_tolerance.is_empty() && tolerance.shaft_tolerance.is_empty());
        assert_eq!(detail.model_value, 4.0);
    }
}

#[test]
fn invalid_tolerances_leave_document_and_history_unchanged() {
    let (mut workspace, id) = parameter_workspace(tolerance_parameters());
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for (mode, upper, lower, hole, shaft) in [
            ("unknown", "", "", "", ""),
            ("symmetric", "", "", "", ""),
            ("symmetric", "-1 mm", "", "", ""),
            ("symmetric", "1 deg", "", "", ""),
            ("symmetric", "missing", "", "", ""),
            ("deviation", "1 mm", "", "", ""),
            ("deviation", "-1 mm", "1 mm", "", ""),
            ("fits", "", "", "", ""),
            ("fits", "", "", "H7", "Z9"),
        ] {
            assert!(
                client
                    .parameters()
                    .set_tolerance(&ParameterToleranceArgs {
                        name: "width".into(),
                        mode: mode.into(),
                        upper: upper.into(),
                        lower: lower.into(),
                        hole: hole.into(),
                        shaft: shaft.into(),
                    })
                    .is_err(),
                "mode={mode}, upper={upper}, lower={lower}, hole={hole}, shaft={shaft}"
            );
        }
    }
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .tolerance,
        Tolerance::default()
    );
}

#[test]
fn tolerance_commit_veto_restores_the_previous_band() {
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        tolerance_parameters(),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    assert!(
        Client::new(SessionHost::new(&mut workspace))
            .parameters()
            .set_tolerance(&ParameterToleranceArgs {
                name: "width".into(),
                mode: "symmetric".into(),
                upper: "1 mm".into(),
                ..Default::default()
            })
            .is_err()
    );
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .tolerance,
        Tolerance::default()
    );
    assert!(!workspace.needs_save(id).unwrap());
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
}
