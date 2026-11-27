// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::SketchEntityId,
    spatial_sketch::{SpatialSketch, SpatialSketchLine, SpatialSketchPoint},
};

fn point(id: u64, position: [f64; 3]) -> SpatialSketchPoint {
    SpatialSketchPoint {
        id: SketchEntityId(id),
        position,
    }
}
fn line(id: u64, first: u64, second: u64) -> SpatialSketchLine {
    SpatialSketchLine {
        id: SketchEntityId(id),
        endpoints: [SketchEntityId(first), SketchEntityId(second)],
    }
}

#[test]
fn keeps_source_order_names_and_shared_endpoint_identity() {
    let points = vec![
        point(7, [0., 1., 2.]),
        point(2, [3., 4., 5.]),
        point(9, [1., 0., 0.]),
    ];
    let lines = vec![line(11, 7, 2), line(12, 2, 9)];
    let sketch = SpatialSketch::new(
        DocumentSketchId(4),
        Some("3D Sketch1".into()),
        points.clone(),
        lines.clone(),
    )
    .unwrap();
    assert_eq!(sketch.id(), DocumentSketchId(4));
    assert_eq!(sketch.name(), Some("3D Sketch1"));
    assert_eq!(sketch.points(), points);
    assert_eq!(sketch.lines(), lines);
    assert_eq!(
        sketch.lines()[0].endpoints[1],
        sketch.lines()[1].endpoints[0]
    );
}

#[test]
fn rejects_missing_duplicate_and_nonfinite_geometry() {
    let invalid = [
        SpatialSketch::new(DocumentSketchId(0), None, vec![], vec![]),
        SpatialSketch::new(DocumentSketchId(1), None, vec![point(0, [0.; 3])], vec![]),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [f64::NAN, 0., 0.])],
            vec![],
        ),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [0.; 3]), point(1, [1.; 3])],
            vec![],
        ),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [0.; 3])],
            vec![line(1, 1, 1)],
        ),
        SpatialSketch::new(
            DocumentSketchId(1),
            None,
            vec![point(1, [0.; 3])],
            vec![line(2, 1, 3)],
        ),
    ];
    for result in invalid {
        assert!(result.is_err());
    }
}

#[test]
fn document_owns_spatial_ids_and_rejects_planar_collisions() {
    use oblikovati_model::{
        document_sketch::DocumentSketches,
        sketch::{Sketch2D, SketchPlane},
    };
    let mut owner = DocumentSketches::default();
    let spatial = SpatialSketch::new(
        DocumentSketchId(4),
        Some("3D Sketch1".into()),
        vec![point(7, [0.; 3]), point(8, [1.; 3])],
        vec![line(9, 7, 8)],
    )
    .unwrap();
    owner.add_spatial(spatial.clone()).unwrap();
    assert_eq!(owner.sketch_ids(), vec![4]);
    assert_eq!(owner.entity_ids(), vec![7, 8, 9]);
    assert!(owner.add_spatial(spatial.clone()).is_err());
    assert!(
        owner
            .add(
                DocumentSketchId(4),
                "duplicate",
                SketchPlane::xy(),
                Sketch2D::default()
            )
            .is_err()
    );
    let mut planar = Sketch2D::default();
    planar.add_point(SketchEntityId(7), [0., 0.]).unwrap();
    assert!(
        owner
            .add(DocumentSketchId(5), "collision", SketchPlane::xy(), planar)
            .is_err()
    );
    let saved = owner.recipe();
    assert_eq!(saved.version, 14);
    assert!(owner.same_recipe(&DocumentSketches::from_recipe(&saved).unwrap()));
    let mut old = saved.clone();
    old.version = 13;
    assert!(DocumentSketches::from_recipe(&old).is_err());
    assert_eq!(owner.remove_spatial(DocumentSketchId(4)).unwrap(), spatial);
    assert!(!owner.same_recipe(&DocumentSketches::from_recipe(&saved).unwrap()));
}

#[test]
fn spatial_insertion_rejects_planar_id_and_entity_collisions() {
    use oblikovati_model::{
        document_sketch::DocumentSketches,
        sketch::{Sketch2D, SketchPlane},
    };
    let mut owner = DocumentSketches::default();
    let mut planar = Sketch2D::default();
    planar.add_point(SketchEntityId(7), [0., 0.]).unwrap();
    owner
        .add(DocumentSketchId(4), "planar", SketchPlane::xy(), planar)
        .unwrap();
    for sketch in [
        SpatialSketch::new(DocumentSketchId(4), None, vec![], vec![]).unwrap(),
        SpatialSketch::new(DocumentSketchId(5), None, vec![point(7, [0.; 3])], vec![]).unwrap(),
    ] {
        assert!(owner.add_spatial(sketch).is_err());
    }
    assert!(owner.spatial_sketches().is_empty());
}

#[test]
fn recipe_validation_rejects_bad_references_and_tracks_coordinate_bits() {
    use oblikovati_model::document_sketch::DocumentSketches;
    let mut owner = DocumentSketches::default();
    owner
        .add_spatial(
            SpatialSketch::new(
                DocumentSketchId(1),
                None,
                vec![point(2, [0., 0., 0.])],
                vec![],
            )
            .unwrap(),
        )
        .unwrap();
    let mut recipe = owner.recipe();
    recipe.spatial_sketches[0].points[0].position[0] = -0.0;
    let changed = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(!owner.same_recipe(&changed));
    recipe.spatial_sketches[0].lines.push(line(3, 2, 99));
    assert!(DocumentSketches::from_recipe(&recipe).is_err());
}
