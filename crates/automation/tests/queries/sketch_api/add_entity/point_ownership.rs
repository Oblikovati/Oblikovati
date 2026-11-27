// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn canonical_factories_create_one_entity_per_primitive_and_keep_defining_point_names() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_factory_ownership(kind);
    }
}
fn assert_factory_ownership(kind: DocumentType) {
    let (mut files, document, mut source) = configured_files(kind, Default::default());
    create_empty_sketch(&mut files, &mut source);
    let replies = create_factory_primitives(&mut files, &mut source);
    let entities = rows(&files);
    assert_visible_factory_rows(&entities, kind);
    verify_keys(&files, &entities);
    assert_hidden_factory_points(&files, document, &replies, kind);
    assert_factory_copy(&mut files, document, &mut source, &entities);
}
fn assert_visible_factory_rows(entities: &[SketchEntityInfo], kind: DocumentType) {
    let mut expected = vec!["point", "line", "circle", "ellipse", "arc"];
    if kind == DOCUMENT_PART {
        expected.insert(0, "point");
    }
    assert_eq!(
        entities
            .iter()
            .map(|row| row.kind.as_str())
            .collect::<Vec<_>>(),
        expected
    );
    assert_eq!(
        entities.iter().map(|row| row.index).collect::<Vec<_>>(),
        (0..expected.len() as i64).collect::<Vec<_>>()
    );
}
fn create_factory_primitives(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
) -> Vec<AddSketchEntityResult> {
    let point = literal_input("point", &[[30.0, 40.0]]);
    let line = literal_input("line", &[[3.0, 5.0], [7.0, 8.0]]);
    let mut ellipse = literal_input("ellipse", &[[8.0, 9.0]]);
    ellipse.axis = Some(vec![3.0, 4.0]);
    ellipse.major_radius = "3 cm".into();
    ellipse.minor_radius = "1 cm".into();
    [point, line, circle_input(), ellipse, arc_input()]
        .iter()
        .map(|input| add_call(files, source, input).unwrap())
        .collect()
}
fn assert_hidden_factory_points(
    files: &DiscoveryWorkspace,
    document: u64,
    replies: &[AddSketchEntityResult],
    kind: DocumentType,
) {
    let visible: Vec<u64> = rows(files).iter().map(|row| row.id).collect();
    let expected = if kind == DOCUMENT_PART {
        vec![1, 2, 5, 7, 9, 13]
    } else {
        vec![1, 4, 6, 8, 12]
    };
    assert_eq!(visible, expected);
    for reply in replies.iter().skip(1) {
        assert_hidden_reply_points(files, reply, &visible);
    }
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    assert_eq!(
        geometry.owned_entity_ids().len(),
        12 + usize::from(kind == DOCUMENT_PART)
    );
    assert_eq!(geometry.recipe().version, 12);
    assert_eq!(geometry.recipe().defining_points.unwrap().len(), 7);
}
fn assert_hidden_reply_points(
    files: &DiscoveryWorkspace,
    reply: &AddSketchEntityResult,
    visible: &[u64],
) {
    let client = Client::new(FileQueryHost::new(files));
    for &point in reply.point_i_ds.as_ref().unwrap() {
        assert!(!visible.contains(&point));
        let key = files
            .workspace()
            .identity(files.workspace().active_document().unwrap())
            .unwrap()
            .internal_name
            .sketch_entity_key(point)
            .to_string();
        let found = client
            .sketch()
            .resolve_reference(&ResolveSketchReferenceArgs { reference_key: key })
            .unwrap();
        assert!(found.found);
        assert_eq!(found.entity_id, point);
    }
}
fn assert_factory_copy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    original: &[SketchEntityInfo],
) {
    let document = reopen_with_observer(files, document, source, Default::default());
    assert_eq!(rows(files), original);
    activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/factory-ownership-copy.obk"),
    );
    let copied = rows(files);
    assert_eq!(copied.len(), original.len());
    for (first, second) in original.iter().zip(&copied) {
        assert_eq!(
            (first.id, first.index, &first.points),
            (second.id, second.index, &second.points)
        );
        assert_ne!(first.reference_key, second.reference_key);
    }
    verify_keys(files, &copied);
}
