// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    feature::{
        AuthoredFeature, FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_plane::{WorkPlaneDefinition, WorkPlaneDistance, WorkPlaneReference},
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use std::collections::BTreeSet;

fn append_plane(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    base: Option<FeatureId>,
) -> Result<FeatureId, String> {
    let id = ids.allocate_feature().map_err(|error| error.to_string())?;
    let definition = chain_definition(base);
    snapshot
        .work_planes
        .append(AuthoredFeature::work_plane(id, "Construction", definition))
        .map_err(|error| error.to_string())?;
    snapshot
        .work_planes
        .set_work_plane_construction(id, true)
        .map_err(|error| error.to_string())?;
    Ok(id)
}

fn construction_chain(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> (FeatureId, FeatureId) {
    files
        .commands(document)
        .unwrap()
        .run("Host sketch", |snapshot, ids| {
            let first = append_plane(snapshot, ids, None)?;
            let second = append_plane(snapshot, ids, Some(first))?;
            snapshot
                .sketches
                .attach_work_plane(sketch, second)
                .map_err(|error| error.to_string())?;
            Ok::<_, String>((first, second))
        })
        .unwrap()
}

fn plane_count(files: &DiscoveryWorkspace, document: u64) -> usize {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_planes
        .features()
        .len()
}

#[test]
fn typed_deletion_prunes_the_last_consumer_chain_and_undo_restores_its_recipe() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    construction_chain(&mut files, document, sketch);
    files.save(document, "test", &mut source).unwrap();
    assert!(delete_call(&mut files, &mut source, 0).unwrap().ok);
    assert_eq!(plane_count(&files, document), 0);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(plane_count(&files, document), 2);
    assert!(!files.workspace().needs_save(document).unwrap());
    files.commands(document).unwrap().redo().unwrap();
    let reopened = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(plane_count(&files, reopened), 0);
    assert!(owned_geometry(&files, reopened).sketches().is_empty());
    let next = insert_circle(&mut files, reopened);
    assert!(next.0 > sketch.0);
    let planes = construction_chain(&mut files, reopened, next);
    assert_eq!(planes, (FeatureId(3), FeatureId(4)));
}

fn set_attribution(
    files: &mut DiscoveryWorkspace,
    document: u64,
    plane: FeatureId,
    sketch: DocumentSketchId,
    unknown: bool,
) {
    files
        .commands(document)
        .unwrap()
        .run("Consumer", |snapshot, _| {
            let definition = *snapshot.work_planes.get(plane).unwrap().definition();
            let mut keys = BTreeSet::from([FeatureReadKey::Sketch(sketch.0)]);
            keys.extend(definition.reads().keys());
            let reads = FeatureReadFootprint::new(keys, unknown);
            snapshot.work_planes.replace(plane, definition, reads)
        })
        .unwrap();
}

#[test]
fn typed_consumed_and_unattributed_declines_do_not_change_the_document() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let (first, _) = construction_chain(&mut files, document, sketch);
    for (unknown, expected) in [(false, "consumed"), (true, "unattributed")] {
        set_attribution(&mut files, document, first, sketch, unknown);
        let before = owner_revision(&files, document);
        let geometry = owned_geometry(&files, document).clone();
        expect_api_decline(delete_call(&mut files, &mut source, 0), expected);
        assert_eq!(before, owner_revision(&files, document));
        assert_eq!(plane_count(&files, document), 2);
        assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
    }
}

fn chain_definition(base: Option<FeatureId>) -> WorkPlaneDefinition {
    match base {
        None => WorkPlaneDefinition::Fixed(oblikovati_model::sketch::SketchPlane::xy()),
        Some(base) => WorkPlaneDefinition::Offset {
            base: WorkPlaneReference::Feature(base),
            distance: WorkPlaneDistance::Constant(2.0),
        },
    }
}
