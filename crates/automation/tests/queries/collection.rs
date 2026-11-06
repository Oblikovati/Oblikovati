// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_model::parameter::CollectionSettings;

#[test]
fn collection_queries_read_defaults_and_require_document_context() {
    let (workspace, id) = parameter_workspace(ParameterCollection::new(UnitsOfMeasure::default()));
    let client = Client::new(SessionQueryHost::new(&workspace));
    assert_eq!(
        client.parameters().get_settings().unwrap(),
        ParameterSettingsInfo {
            linear_dimension_precision: 3,
            angular_dimension_precision: 2,
            dimension_display_type: "value".into(),
            ..Default::default()
        }
    );
    assert_eq!(client.parameters().list_groups().unwrap().groups, None);
    assert!(!workspace.needs_save(id).unwrap());
    let empty = Workspace::new(QueryObserver::default());
    let client = Client::new(SessionQueryHost::new(&empty));
    assert!(client.parameters().get_settings().is_err());
    assert!(client.parameters().list_groups().is_err());
}

#[test]
fn settings_queries_preserve_every_field_and_display_mode() {
    for (mode, name) in [
        (DIMENSION_DISPLAY_VALUE, "value"),
        (DIMENSION_DISPLAY_NAME, "name"),
        (DIMENSION_DISPLAY_EXPRESSION, "expression"),
        (DIMENSION_DISPLAY_TOLERANCE, "tolerance"),
        (DIMENSION_DISPLAY_PRECISE_VALUE, "preciseValue"),
    ] {
        let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
        parameters
            .set_settings(CollectionSettings {
                linear_standard_tolerance: "0.02 mm".into(),
                angular_standard_tolerance: "0.1 deg".into(),
                use_standard_tolerances: true,
                export_standard_tolerances: true,
                linear_dimension_precision: 8,
                angular_dimension_precision: 0,
                dimension_display_type: mode,
                display_parameter_as_expression: true,
            })
            .unwrap();
        let (workspace, id) = parameter_workspace(parameters);
        let client = Client::new(SessionQueryHost::new(&workspace));
        assert_eq!(
            client.parameters().get_settings().unwrap(),
            ParameterSettingsInfo {
                linear_standard_tolerance: "0.02 mm".into(),
                angular_standard_tolerance: "0.1 deg".into(),
                use_standard_tolerances: true,
                export_standard_tolerances: true,
                linear_dimension_precision: 8,
                angular_dimension_precision: 0,
                dimension_display_type: name.into(),
                display_parameter_as_expression: true,
            }
        );
        assert_eq!(workspace.session(id).unwrap().generation(), 0);
        assert!(!workspace.needs_save(id).unwrap());
    }
}

#[test]
fn groups_keep_creation_order_and_members_follow_parameter_order() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(9, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "height", USER_PARAM, "2 cm")
        .unwrap();
    parameters.add_group("dims", "Dimensions", "addin").unwrap();
    parameters.add_group("empty", "Empty", "").unwrap();
    parameters.add_to_group(2, "dims").unwrap();
    parameters.add_to_group(9, "dims").unwrap();
    let (workspace, id) = parameter_workspace(parameters);
    let notices = workspace.observer().notifications;
    let client = Client::new(SessionQueryHost::new(&workspace));
    assert_eq!(
        client.parameters().list_groups().unwrap().groups,
        Some(vec![
            ParameterGroupInfo {
                internal_name: "dims".into(),
                display_name: "Dimensions".into(),
                client_id: "addin".into(),
                members: Some(vec!["width".into(), "height".into()])
            },
            ParameterGroupInfo {
                internal_name: "empty".into(),
                display_name: "Empty".into(),
                client_id: String::new(),
                members: None
            },
        ])
    );
    assert_eq!(workspace.observer().notifications, notices);
    assert_eq!(workspace.session(id).unwrap().generation(), 0);
    assert!(!workspace.needs_save(id).unwrap());
}
