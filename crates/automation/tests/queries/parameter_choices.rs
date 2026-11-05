// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;

fn toleranced_parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "height", USER_PARAM, "2 cm")
        .unwrap();
    parameters
        .insert_expression(3, "plain", USER_PARAM, "1 cm")
        .unwrap();
    parameters.set_deviation(1, 0.2, -0.1).unwrap();
    parameters.set_deviation(2, 0.4, -0.3).unwrap();
    parameters
}

#[test]
fn value_lists_preserve_custom_order_sort_when_requested_and_clear_with_null() {
    let (mut workspace, id) = parameter_workspace(toleranced_parameters());
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        let mut request = ParameterExpressionListArgs {
            name: "width".into(),
            expressions: Some(vec!["8 cm".into(), "2 cm".into()]),
            allow_custom_values: true,
            custom_order: true,
        };
        let detail = client.parameters().set_expression_list(&request).unwrap();
        assert_eq!(detail.parameter_info.value, "40 mm");
        let choices = detail.expression_list.unwrap();
        assert_eq!(choices.expressions, request.expressions);
        assert!(choices.custom_order && choices.allow_custom_values);
        request.custom_order = false;
        let detail = client.parameters().set_expression_list(&request).unwrap();
        assert_eq!(
            detail.expression_list.unwrap().expressions,
            Some(vec!["2 cm".into(), "8 cm".into()])
        );
        request.expressions = None;
        assert_eq!(
            client
                .parameters()
                .set_expression_list(&request)
                .unwrap()
                .expression_list,
            None
        );
    }
    assert_eq!(
        workspace.session(id).unwrap().undo_labels(),
        ["Edit Value List", "Edit Value List", "Edit Value List"]
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
            .choices
            .expressions,
        Some(vec!["2 cm".into(), "8 cm".into()])
    );
}

#[test]
fn sweeps_report_eligible_count_and_preserve_nominal_values() {
    let (mut workspace, id) = parameter_workspace(toleranced_parameters());
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for (selection, expected) in [
            ("upper", 4.2),
            ("lower", 3.9),
            ("median", 4.05),
            ("nominal", 4.0),
        ] {
            assert_eq!(
                client
                    .parameters()
                    .set_all_model_value_type(&ParameterSweepArgs {
                        model_value_type: selection.into()
                    })
                    .unwrap()
                    .affected,
                2
            );
            let detail = client
                .parameters()
                .get_detail(&ParameterNameArgs {
                    name: "width".into(),
                })
                .unwrap();
            assert_eq!(detail.model_value, expected);
            assert_eq!(detail.parameter_info.value, "40 mm");
            assert_eq!(
                client
                    .parameters()
                    .get_detail(&ParameterNameArgs {
                        name: "plain".into()
                    })
                    .unwrap()
                    .model_value_type,
                "nominal"
            );
        }
        assert_eq!(
            client
                .parameters()
                .set_all_model_value_type(&ParameterSweepArgs {
                    model_value_type: "nominal".into()
                })
                .unwrap()
                .affected,
            2
        );
    }
    assert_eq!(workspace.session(id).unwrap().undo_labels().len(), 4);
    workspace.commands(id).unwrap().undo().unwrap();
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .model_value_type,
        MODEL_VALUE_MEDIAN
    );
}

#[test]
fn failed_sweep_is_atomic_and_invalid_selection_is_refused() {
    let mut parameters = toleranced_parameters();
    parameters.set_expression(2, "missing").unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for selection in ["upper", "bad", "Upper", ""] {
            assert!(
                client
                    .parameters()
                    .set_all_model_value_type(&ParameterSweepArgs {
                        model_value_type: selection.into()
                    })
                    .is_err()
            );
        }
    }
    for parameter in [1, 2] {
        assert_eq!(
            workspace
                .session(id)
                .unwrap()
                .snapshot()
                .parameters
                .get(parameter)
                .unwrap()
                .model_value_type,
            MODEL_VALUE_NOMINAL
        );
    }
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    assert!(!workspace.needs_save(id).unwrap());
}

#[test]
fn value_list_errors_noops_and_sweep_veto_do_not_change_history() {
    let mut parameters = toleranced_parameters();
    parameters.convert_kind(2, REFERENCE_PARAM).unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    {
        let client = Client::new(SessionHost::new(&mut workspace));
        for name in ["missing", "height"] {
            assert!(
                client
                    .parameters()
                    .set_expression_list(&ParameterExpressionListArgs {
                        name: name.into(),
                        expressions: Some(vec!["2 cm".into()]),
                        ..Default::default()
                    })
                    .is_err()
            );
        }
        client
            .parameters()
            .set_expression_list(&ParameterExpressionListArgs {
                name: "width".into(),
                ..Default::default()
            })
            .unwrap();
    }
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
    let (mut workspace, id) = super::parameters::parameter_workspace_observer(
        toleranced_parameters(),
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    assert!(
        Client::new(SessionHost::new(&mut workspace))
            .parameters()
            .set_all_model_value_type(&ParameterSweepArgs {
                model_value_type: "upper".into()
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
            .model_value_type,
        MODEL_VALUE_NOMINAL
    );
    assert!(!workspace.needs_save(id).unwrap());
}
