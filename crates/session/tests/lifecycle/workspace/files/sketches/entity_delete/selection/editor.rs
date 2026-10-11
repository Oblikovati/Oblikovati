// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::selection::{SelectionSet, SketchPick, SketchPickTarget};
fn scene_picks(host: &mut Host, document: u64, scene: &DeletionScene) -> SelectionSet<SketchPick> {
    let intent = selected_scene(host, document, scene);
    let entities = intent.entities.into_iter().map(SketchPickTarget::Entity);
    let dimensions = intent
        .dimensions
        .into_iter()
        .map(SketchPickTarget::Dimension);
    let constraints = intent
        .geometric_constraints
        .into_iter()
        .map(SketchPickTarget::GeometricConstraint);
    let targets = entities.chain(dimensions).chain(constraints);
    let commands = host.commands(document).unwrap();
    let mut picks = SelectionSet::default();
    for target in targets {
        picks.add(commands.sketch_pick(scene.sketch, target).unwrap());
    }
    picks
}
#[test]
fn borrowed_editor_selection_deletes_and_saves_through_part_and_assembly_owners() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_borrowed_editor(kind);
    }
}
fn verify_borrowed_editor(kind: DocumentType) {
    let (mut host, mut source, document, scene) = deletion_owner(kind);
    let mut picks = scene_picks(&mut host, document, &scene);
    let report = host
        .commands(document)
        .unwrap()
        .delete_selected_sketch_picks(Some(scene.sketch), false, &mut picks)
        .unwrap();
    assert_eq!(report.entities, [scene.circles[0].entity]);
    assert_eq!(report.constraints.len(), 4);
    assert!(picks.items().is_empty());
    assert_unconstrained_survivors(&host, document, &scene);
    let reopened = save_reopen_owner(&mut host, document, &mut source);
    assert_unconstrained_survivors(&host, reopened, &scene);
}
#[test]
fn borrowed_editor_refuses_stale_picks_after_undo_without_dirtying_the_restored_recipe() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_borrowed_stale(kind);
    }
}
fn verify_borrowed_stale(kind: DocumentType) {
    let (mut host, _, document, scene) = deletion_owner(kind);
    let before = owner_snapshot(&host, document).sketches.clone();
    let mut stale = scene_picks(&mut host, document, &scene);
    delete_cloned_picks(&mut host, document, scene.sketch, &stale);
    host.commands(document).unwrap().undo().unwrap();
    let items = stale.items().to_vec();
    assert!(
        host.commands(document)
            .unwrap()
            .delete_selected_sketch_picks(Some(scene.sketch), false, &mut stale)
            .is_err()
    );
    assert_eq!(stale.items(), items);
    assert!(before.same_recipe(&owner_snapshot(&host, document).sketches));
    assert!(!host.workspace().needs_save(document).unwrap());
}

fn delete_cloned_picks(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
    picks: &SelectionSet<SketchPick>,
) {
    let mut selected = picks.clone();
    host.commands(document)
        .unwrap()
        .delete_selected_sketch_picks(Some(sketch), false, &mut selected)
        .unwrap();
    assert!(selected.items().is_empty());
}
