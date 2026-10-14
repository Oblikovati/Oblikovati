// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{DocumentEditHistory, SketchLineChainCreation, SketchLineChainCreationReport},
    identity::RecipeIdAllocator,
    sketch::SketchInferencePolicy,
};
#[test]
fn native_inferred_chain_keeps_visible_vertices_and_keys_after_reopen() {
    for closed in [false, true] {
        verify_native_inferred_chain(closed);
    }
}
fn verify_native_inferred_chain(closed: bool) {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("inferred-chain.opd");
    let (package, report) = inferred_chain_package(closed);
    std::fs::write(&path, package).unwrap();
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    verify_inferred_chain_keys(&mut owner, &report);
    let first = publish_native_scene(&mut owner, &mut window);
    assert!(window.sketch_display_declines().unwrap().is_empty());
    super::placement_recipe_tests::verify_reopened_placement(&mut owner, document, &path);
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        first.rgba
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn inferred_chain_package(closed: bool) -> (Vec<u8>, SketchLineChainCreationReport) {
    let snapshot = fixture::selected_prism_document();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    let mut history = DocumentEditHistory::new(snapshot, ids).unwrap();
    let token = history.begin("Visible inferred chain").unwrap();
    let request = SketchLineChainCreation {
        points: vec![[5.0, 0.0], [7.0, 0.0], [7.0, 2.0], [6.0, 2.0], [6.0, 1.0]],
        closed,
        construction: false,
        centerline: false,
    };
    let report = history
        .edit(&token, |snapshot, ids| {
            let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
            snapshot.create_inferred_sketch_line_chain(
                fixture::SKETCH,
                &request,
                policy,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&native_scene_package()).unwrap();
    let encoded = package
        .with_sketches(
            &history.snapshot().sketches,
            1,
            history.sketch_reservations().entity,
        )
        .unwrap()
        .encode()
        .unwrap();
    (encoded, report)
}
fn verify_inferred_chain_keys(
    owner: &mut NativeDesktopOwner,
    report: &SketchLineChainCreationReport,
) {
    let client = Client::new(owner.document_host());
    let rows = super::copy_tests::native_sketch_entities(&client);
    let members = report
        .points
        .iter()
        .copied()
        .chain(report.lines.iter().map(|line| line.primitive.entity));
    for member in members {
        super::copy_tests::verify_native_entity_reference(
            &client,
            rows.iter().find(|row| row.id == member.0).unwrap(),
        );
    }
}

#[test]
fn native_document_chain_preview_matches_commit_without_editing_its_source() {
    for closed in [false, true] {
        verify_document_chain_preview(closed);
    }
}
fn verify_document_chain_preview(closed: bool) {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("document-chain-preview.opd");
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &path);
    let original = publish_native_scene(&mut owner, &mut window);
    let request = SketchLineChainCreation {
        points: vec![[4.0001, 3.0001], [6.0, 3.02], [6.01, 5.0]],
        closed,
        construction: false,
        centerline: false,
    };
    let preview =
        verify_private_chain_preview(&mut owner, &mut window, document, &request, &original);
    let report = owner
        .commit_sketch_line_chain(document, fixture::SKETCH, &request)
        .unwrap();
    assert_eq!(report.lines[0].inference.points.len(), 1);
    verify_preview_geometry_adoption(&owner, document, &preview);
    window.set_sketch_preview(None).unwrap();
    verify_inferred_chain_keys(&mut owner, &report);
    let committed =
        super::copy_tests::verify_copy_readiness_and_undo(&mut owner, &mut window, &original, 2);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        committed.rgba
    );
    super::placement_recipe_tests::verify_reopened_placement(&mut owner, document, &path);
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        committed.rgba
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_private_chain_preview(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
    document: u64,
    request: &SketchLineChainCreation,
    original: &ViewportCapture,
) -> oblikovati_viewport::ViewportSketchPreview {
    let session = owner.documents().workspace().session(document).unwrap();
    let before = session.snapshot().sketches.clone();
    let marks = session.history().sketch_reservations();
    let generation = session.generation();
    let preview = owner
        .preview_sketch_line_chain(document, fixture::SKETCH, request)
        .unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    assert_eq!(session.snapshot().sketches, before);
    assert_eq!(session.history().sketch_reservations(), marks);
    assert_eq!(session.generation(), generation);
    assert_eq!(preview.member_count(), if request.closed { 6 } else { 5 });
    assert!(window.set_sketch_preview(Some(&preview)).unwrap());
    let visible = capture_native_scene(window);
    assert_ne!(visible.rgba, original.rgba);
    assert_eq!(visible.scene_generation, original.scene_generation);
    assert!(!window.set_sketch_preview(Some(&preview)).unwrap());
    verify_failed_preview_preserves_overlay(owner, window, document, request, &preview, &visible);
    window.set_sketch_preview(None).unwrap();
    assert_eq!(capture_native_scene(window).rgba, original.rgba);
    window.set_sketch_preview(Some(&preview)).unwrap();
    preview
}
fn verify_failed_preview_preserves_overlay(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
    document: u64,
    request: &SketchLineChainCreation,
    preview: &oblikovati_viewport::ViewportSketchPreview,
    visible: &ViewportCapture,
) {
    assert!(
        owner
            .preview_sketch_line_chain(999, fixture::SKETCH, request)
            .is_err()
    );
    assert!(
        owner
            .commit_sketch_line_chain(999, fixture::SKETCH, request)
            .is_err()
    );
    let mut invalid = request.clone();
    invalid.points[0][0] = f64::NAN;
    assert!(
        owner
            .preview_sketch_line_chain(document, fixture::SKETCH, &invalid)
            .is_err()
    );
    assert!(
        owner
            .commit_sketch_line_chain(document, fixture::SKETCH, &invalid)
            .is_err()
    );
    let mut malformed = preview.clone();
    let oblikovati_viewport::ViewportSketchPreview::Resolved { members, .. } = &mut malformed
    else {
        panic!("expected document result");
    };
    members.push(oblikovati_model::sketch::SketchEntityId(u64::MAX));
    assert!(window.set_sketch_preview(Some(&malformed)).is_err());
    assert_eq!(capture_native_scene(window).rgba, visible.rgba);
}
fn verify_preview_geometry_adoption(
    owner: &NativeDesktopOwner,
    document: u64,
    preview: &oblikovati_viewport::ViewportSketchPreview,
) {
    let oblikovati_viewport::ViewportSketchPreview::Resolved { geometry, .. } = preview else {
        panic!("expected document result");
    };
    let committed = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry();
    assert_eq!(committed, geometry.as_ref());
}
