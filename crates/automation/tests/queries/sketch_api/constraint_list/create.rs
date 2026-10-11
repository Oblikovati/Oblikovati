// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{AddConstraintArgs, AddConstraintResult};
fn fresh(kind: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) = configured_files(kind, Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Owned constructor operands", install_operands)
        .unwrap();
    (files, document, source)
}
fn install_operands(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let sketch = ids.allocate_sketch().unwrap();
    for expected in 1..=11 {
        assert_eq!(ids.allocate_sketch_entity().unwrap(), entity(expected));
    }
    snapshot
        .sketches
        .add(sketch, "Author", SketchPlane::xy(), constructor_geometry())
        .map_err(|cause| cause.to_string())
}
fn constructor_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    let points = [
        [0.0, 0.0],
        [1.0, 0.0],
        [0.0, 1.0],
        [3.0, 2.0],
        [4.0, 2.0],
        [3.0, 3.0],
    ];
    for (index, position) in points.into_iter().enumerate() {
        geometry
            .add_point(entity(index as u64 + 1), position)
            .unwrap();
    }
    append_curves(&mut geometry);
    geometry
}
fn append_curves(geometry: &mut Sketch2D) {
    geometry
        .add_line(entity(7), [entity(1), entity(2)])
        .unwrap();
    geometry
        .add_line(entity(8), [entity(4), entity(5)])
        .unwrap();
    geometry.add_circle(entity(9), entity(1), 1.0).unwrap();
    geometry.add_circle(entity(10), entity(4), 1.0).unwrap();
    geometry
        .add_arc(entity(11), entity(1), [entity(2), entity(3)], true)
        .unwrap();
}
fn request(kind: &str, refs: &[u64]) -> AddConstraintArgs {
    AddConstraintArgs {
        kind: kind.into(),
        entities: Some(refs.to_vec()),
        ..Default::default()
    }
}
fn add(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &AddConstraintArgs,
) -> Result<AddConstraintResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .add(input)
}
#[test]
fn canonical_constructor_vocabulary_returns_actual_kind_owned_rows_and_resulting_dof() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for (kind, refs, actual) in constructors() {
            assert_constructor(owner, kind, &refs, actual);
        }
    }
}
fn constructors() -> Vec<(&'static str, Vec<u64>, &'static str)> {
    point_constructors()
        .into_iter()
        .chain(line_constructors())
        .chain(circular_constructors())
        .collect()
}
fn point_constructors() -> Vec<(&'static str, Vec<u64>, &'static str)> {
    vec![
        ("coincident", vec![2, 4], "coincident"),
        ("horizontal", vec![2, 4], "horizontalAlign"),
        ("vertical", vec![2, 4], "verticalAlign"),
        ("horizontalAlign", vec![2, 4], "horizontalAlign"),
        ("verticalAlign", vec![2, 4], "verticalAlign"),
        ("fix", vec![2], "fix"),
        ("midpoint", vec![6, 7], "midpoint"),
        ("midpoint", vec![6, 11], "midpoint"),
        ("pointOnLine", vec![6, 7], "pointOnLine"),
        ("pointOnCircle", vec![3, 9], "pointOnCircle"),
        ("pointOnCircle", vec![6, 11], "pointOnCircle"),
    ]
}
fn line_constructors() -> Vec<(&'static str, Vec<u64>, &'static str)> {
    vec![
        ("horizontal", vec![7], "horizontal"),
        ("vertical", vec![7], "vertical"),
        ("parallel", vec![7, 8], "parallel"),
        ("perpendicular", vec![7, 8], "perpendicular"),
        ("collinear", vec![7, 8], "collinear"),
        ("equalLength", vec![7, 8], "equalLength"),
        ("tangent", vec![7, 10], "tangent"),
        ("tangent", vec![8, 11], "tangent"),
    ]
}
fn circular_constructors() -> Vec<(&'static str, Vec<u64>, &'static str)> {
    let mut cases = Vec::new();
    for refs in [vec![9, 10], vec![10, 9], vec![10, 11], vec![11, 10]] {
        for kind in ["concentric", "equalRadius", "tangent"] {
            cases.push((kind, refs.clone(), kind));
        }
    }
    cases
}
fn assert_constructor(owner: DocumentType, kind: &str, refs: &[u64], actual: &str) {
    let (mut files, document, mut source) = fresh(owner);
    let focus = files.workspace().active_document();
    let result = add(&mut files, &mut source, &request(kind, refs)).unwrap();
    assert_eq!(result.index, 0);
    assert_eq!(result.kind, actual);
    let row = &list(&files)[0];
    assert_eq!(row.kind, result.kind);
    assert_eq!(row.entities.as_deref(), Some(refs));
    assert_eq!(row.index, result.index);
    assert_eq!(result.dof, status_call(&files, 0).unwrap().dof);
    assert_eq!(files.workspace().active_document(), focus);
    assert_eq!(
        owner_revision(&files, document).1.last().unwrap(),
        "Create sketch constraint"
    );
}
#[test]
fn unsupported_kinds_counts_orders_and_foreign_refs_cannot_mutate_or_allocate() {
    let inputs = invalid_constructors();
    for (kind, refs) in inputs {
        assert_refused(kind, &refs);
    }
}
fn invalid_constructors() -> Vec<(&'static str, Vec<u64>)> {
    vec![
        ("horizontal", vec![]),
        ("horizontal", vec![1, 2, 3]),
        ("horizontal", vec![9]),
        ("horizontalAlign", vec![7]),
        ("fix", vec![7]),
        ("midpoint", vec![2, 9]),
        ("midpoint", vec![7, 2]),
        ("pointOnCircle", vec![9, 2]),
        ("pointOnLine", vec![7, 2]),
        ("coincident", vec![7, 8]),
        ("parallel", vec![2, 7]),
        ("concentric", vec![7, 9]),
    ]
    .into_iter()
    .chain(invalid_curve_constructors())
    .collect()
}
fn assert_refused(kind: &str, refs: &[u64]) {
    let (mut files, document, mut source) = fresh(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let input = request(kind, refs);
    assert!(add(&mut files, &mut source, &input).is_err());
    assert_eq!(owner_revision(&files, document), before);
    assert!(list(&files).is_empty());
    let created = add(&mut files, &mut source, &request("horizontal", &[7])).unwrap();
    assert_eq!(created.index, 0);
    assert_eq!(
        owned_geometry(&files, document).sketches()[0]
            .geometry()
            .constraints()[0]
            .id,
        SketchConstraintId(1)
    );
}
#[test]
fn optional_ellipse_flags_and_custom_metadata_do_not_change_supported_point_line_requests() {
    let (mut files, _, mut source) = fresh(DOCUMENT_PART);
    let mut input = request("horizontal", &[7]);
    input.use_ellipse_major_axis = Some(std::boxed::Box::new(false));
    input.use_ellipse_one_major_axis = Some(std::boxed::Box::new(false));
    input.use_ellipse_two_major_axis = Some(std::boxed::Box::new(false));
    input.name = "ignored by legacy built-in constructor".into();
    input.client_id = "ignored by legacy built-in constructor".into();
    let result = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .constrain()
        .add_with(&input)
        .unwrap();
    assert_eq!(result.kind, "horizontal");
    assert_eq!(list(&files)[0].name, "");
    assert_eq!(list(&files)[0].client_id, "");
}

fn invalid_curve_constructors() -> Vec<(&'static str, Vec<u64>)> {
    vec![
        ("tangent", vec![9, 8]),
        ("tangent", vec![7, 8]),
        ("tangent", vec![2, 9]),
        ("equalRadius", vec![7, 8]),
        ("symmetry", vec![1, 2, 7]),
        ("smooth", vec![7, 8]),
        ("custom", vec![1]),
        ("offset", vec![7, 8]),
        ("future", vec![2]),
        ("fix", vec![99]),
    ]
}

#[path = "create/lifecycle.rs"]
mod lifecycle;

#[path = "create/dimensions.rs"]
mod dimensions;
