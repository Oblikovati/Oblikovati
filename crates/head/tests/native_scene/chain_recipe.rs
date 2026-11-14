// SPDX-License-Identifier: GPL-2.0-only
use super::copy_tests::{native_sketch_entities, verify_native_entity_reference};
use super::*;
use oblikovati_model::{
    history::SketchPlacementFieldLock,
    sketch::{SketchPlacementRecipe, SketchPlane},
};
use oblikovati_viewport::ViewportSketchPreview;
#[test]
fn native_chain_recipe_preview_commits_last_length_and_reopens_shared_vertices() {
    for closed in [false, true] {
        run_native_chain_recipe(closed);
    }
}
fn run_native_chain_recipe(closed: bool) {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("chain-recipe.opd");
    let mut window = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(window.wake.clone()).unwrap();
    let document = open_native_part(&mut owner, &path);
    let original = publish_native_scene(&mut owner, &mut window);
    let recipe = SketchPlacementRecipe::line_chain(
        &[[5., 0.], [7., 0.], [7., 2.], [6., 2.], [6., 1.]],
        closed,
    )
    .unwrap();
    verify_chain_preview(&mut window, &recipe, &original);
    verify_chain_commit(&mut owner, document, &recipe);
    super::copy_tests::verify_copy_readiness_and_undo(&mut owner, &mut window, &original, 9);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    let committed = publish_native_scene(&mut owner, &mut window);
    super::placement_recipe_tests::verify_reopened_placement(&mut owner, document, &path);
    assert_eq!(
        publish_native_scene(&mut owner, &mut window).rgba,
        committed.rgba
    );
    assert!(window.sketch_display_declines().unwrap().is_empty());
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_chain_preview(
    window: &mut NativeHeadWindow,
    recipe: &SketchPlacementRecipe,
    original: &ViewportCapture,
) {
    let preview = ViewportSketchPreview::Recipe {
        recipe: recipe.clone(),
        plane: SketchPlane::xy(),
        solve_options: Default::default(),
    };
    assert!(window.set_sketch_preview(Some(&preview)).unwrap());
    let outlined = capture_native_scene(window);
    assert_ne!(outlined.rgba, original.rgba);
    assert_eq!(outlined.scene_generation, original.scene_generation);
    window.set_sketch_preview(None).unwrap();
    assert_eq!(capture_native_scene(window).rgba, original.rgba);
}
fn verify_chain_commit(
    owner: &mut NativeDesktopOwner,
    document: u64,
    recipe: &SketchPlacementRecipe,
) {
    let locks = [
        Some(SketchPlacementFieldLock {
            expression: format!("{} cm", recipe.fields[0].value),
            text_point: None,
        }),
        None,
    ];
    let report = owner
        .commit_sketch_placement(document, fixture::SKETCH, recipe, &locks)
        .unwrap();
    assert_eq!(report.points.len(), recipe.points.len());
    assert_eq!(report.dimensions.len(), 1);
    let client = Client::new(owner.document_host());
    let rows = native_sketch_entities(&client);
    for member in &report.members {
        verify_native_entity_reference(
            &client,
            rows.iter().find(|row| row.id == member.0).unwrap(),
        );
    }
    let geometry = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(fixture::SKETCH)
        .unwrap()
        .geometry();
    crate::placement_test_samples::assert_recipe_tail(geometry, recipe);
}
