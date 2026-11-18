// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn spatial_sketches_and_shared_reservations_survive_save_and_reopen() {
    use oblikovati_model::spatial_sketch::{SpatialSketch, SpatialSketchLine, SpatialSketchPoint};
    let mut owner = sketches();
    let spatial = SpatialSketch::new(
        DocumentSketchId(10),
        Some("3D Sketch6".into()),
        vec![
            SpatialSketchPoint {
                id: SketchEntityId(20),
                position: [1., 2., 3.],
            },
            SpatialSketchPoint {
                id: SketchEntityId(21),
                position: [4., 5., 6.],
            },
        ],
        vec![SpatialSketchLine {
            id: SketchEntityId(22),
            endpoints: [SketchEntityId(20), SketchEntityId(21)],
        }],
    )
    .unwrap();
    owner.add_spatial(spatial.clone()).unwrap();
    let saved = package().with_sketches(&owner, 10, 22).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    let mut restored = reopened.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&owner));
    assert_eq!(restored.sketches.spatial_sketches(), &[spatial]);
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 11);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 23);
}
