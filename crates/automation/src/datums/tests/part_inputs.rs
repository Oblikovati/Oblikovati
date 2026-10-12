// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[allow(dead_code)]
#[path = "../../../../../test-utilities/rust/selected_sketch_prism.rs"]
mod fixture;
fn mixed_part_snapshot() -> DocumentSnapshot {
    let mut snapshot = fixture::selected_sketch_document();
    add_point(&mut snapshot, 1);
    append_body_inputs(&mut snapshot);
    add_point(&mut snapshot, 3);
    snapshot
}
#[test]
fn body_inputs_do_not_change_datum_rows_or_collection_indices() {
    let snapshot = mixed_part_snapshot();
    let ids: Vec<_> = rows::entries(&snapshot, DatumCollection::Points)
        .map(|entry| entry.id())
        .collect();
    assert_eq!(ids, [FeatureId(1), FeatureId(3)]);
    let bytes = rows::points(&snapshot, ListWorkPointsArgs::default()).unwrap();
    let reply: ListWorkPointsResult = serde_json::from_slice(&bytes).unwrap();
    let points = reply.points.unwrap();
    assert_eq!(points.len(), 3);
    assert_eq!(points[1].index, 1);
    assert_eq!(points[2].index, 2);
    assert!(points.iter().all(|point| point.healthy));
}

fn append_body_inputs(snapshot: &mut DocumentSnapshot) {
    use oblikovati_model::feature::sketch_prism::SketchPrismDefinition;
    let definition = SketchPrismDefinition::new(
        fixture::SKETCH,
        fixture::selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Constant(2.),
        ],
    )
    .unwrap();
    snapshot
        .work_geometry
        .append(AuthoredFeature::sketch_prism(FeatureId(2), "Prism", definition, snapshot).unwrap())
        .unwrap();
}
