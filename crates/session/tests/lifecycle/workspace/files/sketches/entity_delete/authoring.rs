// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn deletion_owner(kind: DocumentType) -> (Host, SequenceGuids, u64, DeletionScene) {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1700);
    let document = create_deletion_document(&mut host, &mut source, kind);
    let scene = author_deletion_scene(&mut host, document);
    author_dimension_metadata(&mut host, document, &scene);
    ground_surviving_points(&mut host, document, &scene);
    host.save(document, "alpha", &mut source).unwrap();
    (host, source, document, scene)
}
fn create_deletion_document(
    host: &mut Host,
    source: &mut SequenceGuids,
    kind: DocumentType,
) -> u64 {
    let parameters = recipe().parameters;
    let package = DocumentPackage::empty(kind, "Deletion")
        .with_parameters(&parameters, parameters.units().clone())
        .unwrap();
    host.create(
        &path("entity-delete.obk"),
        package,
        RecordingObserver::default(),
        "alpha",
        source,
    )
    .unwrap()
    .session_id
}
fn author_deletion_scene(host: &mut Host, document: u64) -> DeletionScene {
    let sketch = create_deletion_sketch(host, document);
    let circles = create_shared_circles(host, document, sketch);
    let marker = create_marker(host, document, sketch);
    let dimensions = [
        create_radius(host, document, sketch, circles[0].entity, true),
        create_radius(host, document, sketch, circles[1].entity, false),
    ];
    DeletionScene {
        sketch,
        circles,
        dimensions,
        marker,
    }
}
fn create_shared_circles(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
) -> [SketchPrimitiveCreationReport; 2] {
    let first = create_circle(host, document, sketch, SketchPointInput::New([0.0, -0.0]));
    let second = create_circle(
        host,
        document,
        sketch,
        SketchPointInput::Owned(first.points[0]),
    );
    [first, second]
}
fn create_deletion_sketch(host: &mut Host, document: u64) -> DocumentSketchId {
    let request = DocumentSketchCreation {
        name: "Shared center".into(),
        placement: SketchPlacement::Fixed(SketchPlane::xy()),
        orientation: None,
        project_origin: false,
    };
    host.commands(document)
        .unwrap()
        .create_sketch(&request)
        .unwrap()
}
fn create_marker(host: &mut Host, document: u64, sketch: DocumentSketchId) -> SketchEntityId {
    let input = SketchPrimitiveCreation::Point {
        position: [8.0, -4.0],
    };
    host.commands(document)
        .unwrap()
        .create_sketch_primitive(sketch, &input)
        .unwrap()
        .entity
}
pub(super) fn create_circle(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
    center: SketchPointInput,
) -> SketchPrimitiveCreationReport {
    let input = SketchPrimitiveCreation::Circle {
        center,
        radius: 2.0,
        construction: false,
    };
    host.commands(document)
        .unwrap()
        .create_sketch_primitive(sketch, &input)
        .unwrap()
}
pub(super) fn create_radius(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
    circle: SketchEntityId,
    reference: bool,
) -> SketchDimensionCreationReport {
    let input = SketchDimensionCreation {
        operands: SketchDimensionalConstraintCreation::Measured(
            MeasuredDimensionOperands::Radius { curve: circle },
        ),
        expression: "width * 2".into(),
        reference,
        text_point: Some([-2.0, 3.0]),
    };
    host.commands(document)
        .unwrap()
        .create_sketch_dimension(sketch, &input)
        .unwrap()
}
fn author_dimension_metadata(host: &mut Host, document: u64, scene: &DeletionScene) {
    let edit = SketchDimensionEdit {
        drive_limits: Some(dimension_bounds()),
        ..Default::default()
    };
    for dimension in &scene.dimensions {
        host.commands(document)
            .unwrap()
            .edit_sketch_dimension(scene.sketch, dimension.constraint, &edit)
            .unwrap();
        expose_dimension(host, document, dimension.parameter);
    }
}
fn expose_dimension(host: &mut Host, document: u64, parameter: u64) {
    let annotations = ParameterAnnotations {
        comment: "authored dimension".into(),
        exposed_as_property: true,
        ..Default::default()
    };
    host.commands(document)
        .unwrap()
        .run("Expose dimension", |snapshot, _| {
            snapshot.parameters.set_annotations(parameter, annotations)
        })
        .unwrap();
}
pub(super) fn dimension_bounds() -> SketchDimensionLimits {
    SketchDimensionLimits {
        lower: 1.0,
        upper: 4.0,
    }
}

fn ground_surviving_points(host: &mut Host, document: u64, scene: &DeletionScene) {
    use oblikovati_model::sketch::SketchGeometricConstraintCreation;
    for entity in [scene.circles[1].points[0], scene.marker] {
        let input = SketchGeometricConstraintCreation::Ground { entity };
        host.commands(document)
            .unwrap()
            .create_sketch_geometric_constraint(scene.sketch, &input)
            .unwrap();
    }
}
