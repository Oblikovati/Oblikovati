// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{AddDimensionArgs, AddDimensionResult};

fn dimension_request(kind: &str, refs: &[u64]) -> AddDimensionArgs {
    AddDimensionArgs {
        kind: kind.into(),
        entities: Some(refs.to_vec()),
        expression: "1 cm".into(),
        driven: true,
        text_point: Some(vec![2.5, -7.25]),
        ..Default::default()
    }
}
fn create_dimension(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &AddDimensionArgs,
) -> Result<AddDimensionResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .add(input)
}
fn measured_forms() -> Vec<AddDimensionArgs> {
    let mut forms = Vec::new();
    for orientation in ["", "aligned", "horizontal", "vertical"] {
        let mut input = dimension_request("distance", &[1, 5]);
        input.orientation = orientation.into();
        forms.push(input);
    }
    for curve in [9, 11] {
        for kind in ["radius", "diameter"] {
            forms.push(dimension_request(kind, &[curve]));
        }
    }
    forms.push(dimension_request("arcLength", &[11]));
    forms
}
fn seeded_forms() -> Vec<AddDimensionArgs> {
    let mut forms = vec![
        dimension_request("angle", &[7, 8]),
        dimension_request("threePointAngle", &[1, 2, 3]),
    ];
    for diameter in [false, true] {
        let mut offset = dimension_request("offsetDim", &[3, 7]);
        offset.linear_diameter = diameter;
        forms.push(offset);
        for far in [false, true] {
            for curve in [9, 11] {
                let mut tangent = dimension_request("tangentDistance", &[8, curve]);
                tangent.linear_diameter = diameter;
                tangent.far_side = far;
                forms.push(tangent);
            }
        }
    }
    forms
}
#[test]
fn canonical_dimension_creation_covers_native_forms_and_both_document_owners() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for mut input in measured_forms().into_iter().chain(seeded_forms()) {
            if input.kind == "angle" || input.kind == "threePointAngle" {
                input.expression = "90 deg".into();
            }
            assert_dimension_creation(owner, &input);
        }
    }
}
fn assert_dimension_creation(owner: DocumentType, input: &AddDimensionArgs) {
    let (mut files, document, mut source) = fresh(owner);
    let result = create_dimension(&mut files, &mut source, input).unwrap();
    let rows = dimensions::dimensions(&files);
    assert_eq!(rows.len(), 1);
    let row = &rows[0];
    assert_eq!(result.index, row.index);
    assert_eq!(result.kind, row.kind);
    assert_eq!(result.parameter, row.name);
    assert_eq!(result.value, row.value);
    assert_eq!(result.dof, status_call(&files, 0).unwrap().dof);
    assert_eq!(row.expression, input.expression);
    assert_eq!(row.text_point, input.text_point);
    assert!(row.driven);
    assert_eq!(row.name, "d0");
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Create sketch dimension"
    );
}
#[test]
fn invalid_dimension_operands_and_options_leave_owner_and_reservations_unchanged() {
    for (kind, refs) in invalid_dimensions() {
        let input = dimension_request(kind, &refs);
        assert_dimension_refused(&input);
    }
    let mut input = dimension_request("radius", &[9]);
    input.orientation = "diagonal".into();
    assert_dimension_refused(&input);
    input.orientation.clear();
    input.text_point = Some(vec![1.0]);
    assert_dimension_refused(&input);
}
fn invalid_dimensions() -> Vec<(&'static str, Vec<u64>)> {
    vec![
        ("distance", vec![]),
        ("distance", vec![1]),
        ("distance", vec![7, 8]),
        ("radius", vec![7]),
        ("diameter", vec![99]),
        ("arcLength", vec![9]),
        ("angle", vec![1, 7]),
        ("threePointAngle", vec![1, 2, 7]),
        ("offsetDim", vec![7, 3]),
        ("tangentDistance", vec![9, 8]),
        ("tangentDistance", vec![7, 8]),
        ("future", vec![1]),
        ("ellipseRadius", vec![9]),
        ("offsetSplineDim", vec![7]),
    ]
}
fn assert_dimension_refused(input: &AddDimensionArgs) {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let cause = create_dimension(&mut files, &mut source, input).unwrap_err();
    assert!(cause.to_string().contains("expected"), "{cause}");
    assert_eq!(owner_revision(&files, document), before);
    assert!(dimensions::dimensions(&files).is_empty());
    let result =
        create_dimension(&mut files, &mut source, &dimension_request("radius", &[9])).unwrap();
    assert_eq!(result.parameter, "d0");
}

#[test]
fn driving_source_stays_live_and_creation_history_restores_the_owned_pair() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_live_creation(owner);
    }
}
fn assert_live_creation(owner: DocumentType) {
    let (mut files, document, mut source) = fresh(owner);
    let mut input = dimension_request("radius", &[9]);
    input.expression = "width * 2".into();
    input.driven = false;
    let result = create_dimension(&mut files, &mut source, &input).unwrap();
    assert!((result.value - 2.0).abs() < 1e-8);
    let rows = dimensions::dimensions(&files);
    assert_eq!(rows[0].expression, "width * 2");
    assert!(!rows[0].driven);
    files.commands(document).unwrap().undo().unwrap();
    assert!(dimensions::dimensions(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimensions::dimensions(&files), rows);
    change_source(&mut files, document);
    assert!((dimensions::dimensions(&files)[0].value - 3.0).abs() < 1e-8);
}
fn change_source(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Change source", |snapshot, _| {
            snapshot
                .parameters
                .set_expression(1, "1.5 cm")
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}
#[test]
fn empty_annotation_is_optional_and_ignored_flags_do_not_change_radial_creation() {
    for annotation in [None, Some(vec![])] {
        let (mut files, _, mut source) = fresh(DOCUMENT_PART);
        let mut input = dimension_request("radius", &[9]);
        input.text_point = annotation;
        input.far_side = true;
        input.linear_diameter = true;
        input.orientation = "vertical".into();
        create_dimension(&mut files, &mut source, &input).unwrap();
        let row = &dimensions::dimensions(&files)[0];
        assert!(row.text_point.is_none());
        assert!(!row.linear_diameter);
        assert_eq!(row.orientation, "");
        assert_eq!(row.value, 1.0);
    }
}

#[path = "add_dimension/acceptance.rs"]
mod acceptance;

#[path = "add_dimension/driving.rs"]
mod driving;

#[path = "add_dimension/offset_spline.rs"]
mod offset_spline;

#[path = "add_dimension/ellipse.rs"]
mod ellipse;
