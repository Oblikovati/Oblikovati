// SPDX-License-Identifier: GPL-2.0-only

use super::*;

#[test]
fn syntax_queries_preserve_source_and_return_distinct_unresolved_names() {
    let workspace = Workspace::new(QueryObserver::default());
    let client = Client::new(SessionQueryHost::new(&workspace));
    let args = ExpressionArgs {
        expression: "width + depth * width".into(),
    };
    assert_eq!(
        client.units().get_driving_parameters(&args).unwrap().names,
        Some(vec!["width".into(), "depth".into()])
    );
    assert_eq!(
        client
            .units()
            .get_locale_corrected_expression(&args)
            .unwrap()
            .value,
        args.expression
    );
    let constant = ExpressionArgs {
        expression: "1 / 0".into(),
    };
    assert_eq!(
        client
            .units()
            .get_driving_parameters(&constant)
            .unwrap()
            .names,
        Some(vec![])
    );
    assert!(
        client
            .units()
            .get_locale_corrected_expression(&constant)
            .is_ok()
    );
    let invalid = ExpressionArgs {
        expression: "1 +".into(),
    };
    assert!(client.units().get_driving_parameters(&invalid).is_err());
    assert!(
        client
            .units()
            .get_locale_corrected_expression(&invalid)
            .is_err()
    );
}

#[test]
fn evaluation_uses_working_scale_and_bare_number_display_preferences() {
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut units = UnitsOfMeasure::default().with_working_scale(2.54).unwrap();
    units.set_preferred(Dimension::Length, "in").unwrap();
    let id = register(&mut workspace, units, &mut QueryGuids(0));
    let notices = workspace.observer().notifications;
    let client = Client::new(SessionQueryHost::new(&workspace));
    for (source, category, expected) in [
        ("2", "length", 2.0),
        ("2", "unitless", 2.0),
        ("5.08 cm", "length", 2.0),
        ("5.08 cm", "angle", 2.0),
    ] {
        let args = ExpressionWithTypeArgs {
            expression: source.into(),
            units_type: category.into(),
        };
        assert_eq!(
            client
                .units()
                .get_value_from_expression(&args)
                .unwrap()
                .value,
            expected
        );
    }
    let result = client
        .units()
        .get_database_units_from_expression(&ExpressionArgs {
            expression: "5.08 cm".into(),
        })
        .unwrap();
    assert_eq!(result.value, 2.0);
    assert_eq!(result.units_type, "length");
    assert_eq!(workspace.session(id).unwrap().generation(), 0);
    assert!(!workspace.needs_save(id).unwrap());
    assert_eq!(workspace.observer().notifications, notices);
}

#[test]
fn validity_reports_expression_failures_and_rejects_invalid_categories() {
    let mut workspace = Workspace::new(QueryObserver::default());
    register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    let client = Client::new(SessionQueryHost::new(&workspace));
    for (source, valid) in [
        ("2", true),
        ("2 cm", true),
        ("2 rad", false),
        ("missing", false),
        ("1 +", false),
        ("1 / 0", false),
        ("1e308 * 2", false),
    ] {
        let args = ExpressionWithTypeArgs {
            expression: source.into(),
            units_type: "length".into(),
        };
        let result = client.units().is_expression_valid(&args).unwrap();
        assert_eq!(result.valid, valid, "{source}");
        assert_eq!(result.error.is_empty(), valid, "{source}");
        assert_eq!(
            client.units().compatible_units(&args).unwrap().compatible,
            valid
        );
    }
    let invalid = ExpressionWithTypeArgs {
        expression: "2".into(),
        units_type: "invalid".into(),
    };
    assert!(client.units().is_expression_valid(&invalid).is_err());
    assert!(client.units().compatible_units(&invalid).is_err());
    assert!(client.units().get_value_from_expression(&invalid).is_err());
    let broken = ExpressionArgs {
        expression: "missing".into(),
    };
    assert!(
        client
            .units()
            .get_database_units_from_expression(&broken)
            .is_err()
    );
    assert!(
        client
            .units()
            .get_value_from_expression(&ExpressionWithTypeArgs {
                expression: broken.expression,
                units_type: "length".into(),
            })
            .is_err()
    );
}

#[test]
fn contextual_expression_queries_require_an_active_document() {
    let workspace = Workspace::new(QueryObserver::default());
    let host = SessionQueryHost::new(&workspace);
    for method in [
        METHOD_UNITS_GET_VALUE_FROM_EXPRESSION,
        METHOD_UNITS_GET_DATABASE_UNITS_FROM_EXPRESSION,
        METHOD_UNITS_IS_EXPRESSION_VALID,
        METHOD_UNITS_COMPATIBLE_UNITS,
    ] {
        assert!(
            host.call(method, b"{}")
                .unwrap_err()
                .to_string()
                .contains("active document")
        );
    }
}

#[test]
fn expression_queries_resolve_live_parameter_nominals_without_model_offsets() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters.set_deviation(1, 1.0, -1.0).unwrap();
    parameters
        .set_model_value_type(1, MODEL_VALUE_UPPER)
        .unwrap();
    let snapshot = DocumentSnapshot {
        attributes: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
        parameters,
        properties: DocumentProperties::default(),
    };
    let history =
        DocumentEditHistory::new(snapshot, RecipeIdAllocator::restore(Some(1), &[1]).unwrap())
            .unwrap();
    let mut workspace = Workspace::new(QueryObserver::default());
    let mut guids = QueryGuids(0);
    let id = workspace
        .register(
            history,
            FileIdentity::new("test", &mut guids).unwrap(),
            QueryObserver::default(),
            &mut guids,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id;
    let client = Client::new(SessionQueryHost::new(&workspace));
    let result = client
        .units()
        .get_database_units_from_expression(&ExpressionArgs {
            expression: "width * 2".into(),
        })
        .unwrap();
    assert_eq!(result.value, 8.0);
    assert_eq!(result.units_type, "length");
    assert_eq!(workspace.session(id).unwrap().generation(), 0);
    assert!(!workspace.needs_save(id).unwrap());
}
