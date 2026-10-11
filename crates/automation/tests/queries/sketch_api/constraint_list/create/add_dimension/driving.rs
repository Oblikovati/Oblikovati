// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_forms_accept_driving_targets_through_the_client_in_both_owners() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for input in measured_forms().into_iter().chain(seeded_forms()) {
            assert_driving_form(owner, input);
        }
    }
}
fn assert_driving_form(owner: DocumentType, mut input: AddDimensionArgs) {
    let (mut files, document, mut source) = fresh(owner);
    let measured = create_dimension(&mut files, &mut source, &input).unwrap();
    files.commands(document).unwrap().undo().unwrap();
    let angular = input.kind == "angle" || input.kind == "threePointAngle";
    let unit = if angular { "rad" } else { "cm" };
    input.expression = format!("{} {unit}", measured.value);
    input.driven = false;
    let result = create_dimension(&mut files, &mut source, &input).unwrap();
    assert_eq!(result.parameter, "d1");
    assert_eq!(result.index, 0);
    assert!((result.value - measured.value).abs() <= 1e-8);
    let row = &dimensions::dimensions(&files)[0];
    assert!(!row.driven);
    assert_eq!(row.expression, input.expression);
    assert_eq!(row.text_point, input.text_point);
    assert_eq!(result.dof, status_call(&files, 0).unwrap().dof);
}
#[test]
fn malformed_sources_consume_identities_but_publish_no_owned_pair() {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    for expression in ["", "2 *", "d2"] {
        let mut input = dimension_request("radius", &[9]);
        input.expression = expression.into();
        assert!(create_dimension(&mut files, &mut source, &input).is_err());
        let snapshot = files.workspace().session(document).unwrap().snapshot();
        assert_eq!(snapshot.parameters.ids(), vec![1]);
        assert!(dimensions::dimensions(&files).is_empty());
    }
    let result =
        create_dimension(&mut files, &mut source, &dimension_request("radius", &[9])).unwrap();
    assert_eq!(result.parameter, "d3");
}
#[test]
fn deleting_a_created_dimension_then_recreating_never_reuses_its_name() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, _, mut source) = fresh(owner);
        let input = dimension_request("radius", &[9]);
        create_dimension(&mut files, &mut source, &input).unwrap();
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .dimension()
            .delete(&Default::default())
            .unwrap();
        assert!(dimensions::dimensions(&files).is_empty());
        let result = create_dimension(&mut files, &mut source, &input).unwrap();
        assert_eq!(result.index, 0);
        assert_eq!(result.parameter, "d1");
    }
}
