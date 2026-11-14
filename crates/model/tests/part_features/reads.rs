// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn rehosting_refreshes_body_dependency_edges_before_candidate_adoption() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let token = owner.begin("Rehost sketch").unwrap();
    owner
        .edit(&token, |document, ids| {
            let id = ids.allocate_feature().unwrap();
            append_host(document, id, "Replacement");
            document.work_geometry.reorder(id, 1).unwrap();
            document.sketches.attach_work_plane(SKETCH, id).unwrap();
            Ok::<_, String>(())
        })
        .unwrap();
    owner.commit(&token, false).unwrap();
    let mut current = owner.snapshot().clone();
    current.validate_prism_reads().unwrap();
    current.work_geometry.remove(FeatureId(1)).unwrap();
    assert!(current.work_geometry.remove(FeatureId(3)).is_err());
}
#[test]
fn stale_saved_body_reads_are_rejected_on_initial_adoption() {
    let mut owner = part_owner();
    append_prism(&mut owner);
    let mut altered = owner.snapshot().clone();
    let id = FeatureId(3);
    append_host(&mut altered, id, "Later");
    altered.sketches.attach_work_plane(SKETCH, id).unwrap();
    let ids = RecipeIdAllocator::restore(None, altered.parameters.ids()).unwrap();
    assert!(
        DocumentEditHistory::new(altered, ids)
            .unwrap_err()
            .to_string()
            .contains("FeatureId(3)")
    );
}

fn append_host(document: &mut DocumentSnapshot, id: FeatureId, name: &str) {
    document
        .work_geometry
        .append(AuthoredFeature::work_plane(
            id,
            name,
            WorkPlaneDefinition::Fixed(SketchPlane::yz()),
        ))
        .unwrap();
}
