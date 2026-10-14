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
