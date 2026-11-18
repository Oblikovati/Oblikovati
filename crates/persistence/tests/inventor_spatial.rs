// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_persistence::inventor_spatial::read_inventor_spatial_lines_and_points;

const FIXTURE: &[u8] = include_bytes!("fixtures/viewport-selection-solid.ipt");

#[test]
fn reads_spatial_points_lines_names_and_endpoint_identity_from_inventor_fixture() {
    let sketches = read_inventor_spatial_lines_and_points(FIXTURE).unwrap();
    assert_eq!(sketches.len(), 2);
    assert_eq!(sketches[0].name.as_deref(), Some("3D Sketch1"));
    assert_eq!(sketches[1].name.as_deref(), Some("3D Sketch6"));
    assert_eq!(
        sketches
            .iter()
            .map(|sketch| sketch.points.len())
            .collect::<Vec<_>>(),
        [2, 3]
    );
    assert_eq!(
        sketches
            .iter()
            .map(|sketch| sketch.lines.len())
            .collect::<Vec<_>>(),
        [1, 1]
    );
    assert_eq!(sketches[0].points[0].position_cm, [0., 0., 2.]);
    assert_eq!(sketches[0].points[1].position_cm, [1.5, 0., 2.]);
    assert_eq!(sketches[1].points[0].position_cm, [-1., -0.5, 2.5]);
    for sketch in &sketches {
        assert_eq!(
            sketch.lines[0].endpoints,
            [
                sketch.points[sketch.points.len() - 2].node,
                sketch.points[sketch.points.len() - 1].node
            ]
        );
    }
    assert_eq!(
        sketches,
        read_inventor_spatial_lines_and_points(FIXTURE).unwrap()
    );
}

#[test]
fn rejects_non_inventor_bytes_without_returning_partial_geometry() {
    let error =
        read_inventor_spatial_lines_and_points(b"not an Inventor compound file").unwrap_err();
    assert!(error.contains("Inventor document"), "{error}");
}

#[test]
fn projects_fixture_sketches_with_caller_owned_ids_and_preserves_references() {
    use oblikovati_model::document_sketch::DocumentSketchId;
    use oblikovati_persistence::inventor_spatial::project_inventor_point_and_line_sketch;
    let source = read_inventor_spatial_lines_and_points(FIXTURE).unwrap();
    let units = oblikovati_model::units::UnitsOfMeasure::default();
    let mut ids = oblikovati_model::identity::RecipeIdAllocator::restore(Some(1000), &[]).unwrap();
    let first =
        project_inventor_point_and_line_sketch(DocumentSketchId(101), &source[0], &units, &mut ids)
            .unwrap();
    let second =
        project_inventor_point_and_line_sketch(DocumentSketchId(102), &source[1], &units, &mut ids)
            .unwrap();
    let scaled = units.clone().with_working_scale(0.1).unwrap();
    let millimetre_working = project_inventor_point_and_line_sketch(
        DocumentSketchId(103),
        &source[0],
        &scaled,
        &mut ids,
    )
    .unwrap();
    assert_eq!(millimetre_working.points()[0].position, [0., 0., 20.]);
    let mut document = oblikovati_model::document_sketch::DocumentSketches::default();
    document.add_spatial(first.clone()).unwrap();
    document.add_spatial(second.clone()).unwrap();
    assert!(
        document.same_recipe(
            &oblikovati_model::document_sketch::DocumentSketches::from_recipe(&document.recipe())
                .unwrap()
        )
    );
    assert_eq!(first.id(), DocumentSketchId(101));
    assert_eq!(first.points()[0].id.0, 1001);
    assert!(second.points()[0].id.0 > first.lines()[0].id.0);
    assert_eq!(second.name(), Some("3D Sketch6"));
    assert_eq!(first.points()[0].position, [0., 0., 2.]);
    assert_eq!(
        first.lines()[0].endpoints,
        [first.points()[0].id, first.points()[1].id]
    );
    assert_eq!(
        second.lines()[0].endpoints,
        [second.points()[1].id, second.points()[2].id]
    );
    assert!(
        project_inventor_point_and_line_sketch(DocumentSketchId(0), &source[0], &units, &mut ids)
            .is_err()
    );
}

#[test]
fn refuses_spatial_coordinate_overflow_in_working_units() {
    use oblikovati_model::{document_sketch::DocumentSketchId, units::UnitsOfMeasure};
    use oblikovati_persistence::inventor_spatial::{
        InventorSpatialPoint, InventorSpatialSketch, project_inventor_point_and_line_sketch,
    };
    let source = InventorSpatialSketch {
        node: 1,
        name: None,
        points: vec![InventorSpatialPoint {
            node: 2,
            position_cm: [f64::MAX, 0., 0.],
        }],
        lines: vec![],
    };
    let units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    let mut ids = oblikovati_model::identity::RecipeIdAllocator::default();
    let error =
        project_inventor_point_and_line_sketch(DocumentSketchId(1), &source, &units, &mut ids)
            .unwrap_err();
    assert!(error.contains("finite coordinates"), "{error}");
}

#[test]
fn projection_rejects_invalid_references_and_exhausted_document_ids() {
    use oblikovati_model::{
        document_sketch::DocumentSketchId, identity::RecipeIdAllocator, units::UnitsOfMeasure,
    };
    use oblikovati_persistence::inventor_spatial::{
        InventorSpatialLine, InventorSpatialPoint, InventorSpatialSketch,
        project_inventor_point_and_line_sketch,
    };
    let mut source = InventorSpatialSketch {
        node: 1,
        name: None,
        points: vec![InventorSpatialPoint {
            node: 2,
            position_cm: [0.; 3],
        }],
        lines: vec![InventorSpatialLine {
            node: 3,
            endpoints: [2, 99],
        }],
    };
    let mut ids = RecipeIdAllocator::default();
    assert!(
        project_inventor_point_and_line_sketch(
            DocumentSketchId(1),
            &source,
            &UnitsOfMeasure::default(),
            &mut ids
        )
        .is_err()
    );
    source.lines.clear();
    let mut exhausted = RecipeIdAllocator::restore(Some(u64::MAX), &[]).unwrap();
    assert!(
        project_inventor_point_and_line_sketch(
            DocumentSketchId(1),
            &source,
            &UnitsOfMeasure::default(),
            &mut exhausted
        )
        .is_err()
    );
    source.points.push(source.points[0].clone());
    assert!(
        project_inventor_point_and_line_sketch(
            DocumentSketchId(1),
            &source,
            &UnitsOfMeasure::default(),
            &mut ids
        )
        .is_err()
    );
}

#[test]
fn fixture_spatial_records_survive_document_save_and_reopen() {
    use oblikovati_model::{
        document_sketch::{DocumentSketchId, DocumentSketches},
        identity::RecipeIdAllocator,
        units::UnitsOfMeasure,
    };
    use oblikovati_persistence::{
        inventor_spatial::project_inventor_point_and_line_sketch, package::DocumentPackage,
    };
    let sources = read_inventor_spatial_lines_and_points(FIXTURE).unwrap();
    let mut owner = DocumentSketches::default();
    let mut sketch_ids = RecipeIdAllocator::restore(Some(10), &[]).unwrap();
    let mut entity_ids = RecipeIdAllocator::restore(Some(100), &[]).unwrap();
    let units = UnitsOfMeasure::default().with_working_scale(0.1).unwrap();
    for source in &sources {
        let id = DocumentSketchId(sketch_ids.allocate().unwrap());
        let sketch =
            project_inventor_point_and_line_sketch(id, source, &units, &mut entity_ids).unwrap();
        owner.add_spatial(sketch).unwrap();
    }
    let base =
        DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nmodel:\n  parameters: []\n")
            .unwrap();
    let package = base
        .with_sketches(&owner, sketch_ids.high_water(), entity_ids.high_water())
        .unwrap();
    let mut restored = DocumentPackage::decode(&package.encode().unwrap())
        .unwrap()
        .restore_sketches()
        .unwrap();
    assert!(owner.same_recipe(&restored.sketches));
    let sketches = restored.sketches.spatial_sketches();
    assert_eq!(sketches[0].name(), Some("3D Sketch1"));
    assert_eq!(sketches[1].name(), Some("3D Sketch6"));
    assert_eq!(sketches[0].points()[1].position, [15., 0., 20.]);
    assert_eq!(
        sketches[1].lines()[0].endpoints,
        [sketches[1].points()[1].id, sketches[1].points()[2].id]
    );
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 13);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 108);
}
