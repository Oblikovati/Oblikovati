// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::units::UnitsOfMeasure;
use oblikovati_persistence::package::DocumentPackage;

fn package(groups: &str) -> DocumentPackage {
    let source = format!(
        "schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters:\n    - {{name: width, kind: user, expression: '2'}}\n    - {{name: height, kind: user, expression: 'width * 2'}}\n  parameterGroups: {groups}\n"
    );
    DocumentPackage::decode(source.as_bytes()).unwrap()
}

#[test]
fn groups_restore_order_labels_clients_and_multi_membership() {
    let package = package(
        "[{internalName: second, displayName: Sizes, clientId: addon, members: [height, width]}, {internalName: first, members: [width]}]",
    );
    let mut restored = package
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    let parameters = &mut restored.parameters;
    assert_eq!(parameters.groups()[0].key.0, "second");
    assert_eq!(parameters.groups()[1].key.0, "first");
    assert_eq!(parameters.group("second").unwrap().display_name, "Sizes");
    assert_eq!(parameters.group("second").unwrap().client_id, "addon");
    assert_eq!(parameters.group("first").unwrap().display_name, "first");
    assert_eq!(parameters.group_members("second").unwrap(), [1, 2]);
    assert_eq!(parameters.group_members("first").unwrap(), [1]);
    assert!(parameters.drain_changed().is_empty());
    parameters.set_expression(1, "3").unwrap();
    assert_eq!(
        parameters.get(2).unwrap().current_value().unwrap().value,
        6.0
    );
}

#[test]
fn repeated_member_names_are_idempotent_like_the_model_group_set() {
    let package = package("[{internalName: sizes, members: [width, width]}]");
    let restored = package
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    assert_eq!(restored.parameters.group_members("sizes").unwrap(), [1]);
}

#[test]
fn missing_members_and_invalid_group_records_refuse_reconstruction() {
    for groups in [
        "[{internalName: sizes, members: [absent]}]",
        "[{internalName: ''}]",
        "[{internalName: sizes}, {internalName: sizes}]",
        "[{internalName: sizes, future: true}]",
        "{}",
    ] {
        assert!(
            package(groups)
                .restore_parameters(UnitsOfMeasure::default())
                .is_err(),
            "{groups}"
        );
    }
    let error = package("[{internalName: sizes, members: [absent]}]")
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap_err();
    assert!(error.to_string().contains("sizes"));
    assert!(error.to_string().contains("absent"));
}

#[test]
fn empty_null_and_memberless_groups_are_valid() {
    for groups in ["[]", "null"] {
        let restored = package(groups)
            .restore_parameters(UnitsOfMeasure::default())
            .unwrap();
        assert!(restored.parameters.groups().is_empty());
    }
    let restored = package("[{internalName: empty}]")
        .restore_parameters(UnitsOfMeasure::default())
        .unwrap();
    assert!(
        restored
            .parameters
            .group_members("empty")
            .unwrap()
            .is_empty()
    );
}
