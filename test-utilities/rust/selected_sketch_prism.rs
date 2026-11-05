// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::feature::sketch_prism::{SketchDirectedLine, SketchPrismSelection};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    history::DocumentSnapshot,
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane,
    },
    units::UnitsOfMeasure,
};
pub const SKETCH: DocumentSketchId = DocumentSketchId(1);
/// Create a parameter-driven, fully constrained rectangle; example: `selected_sketch_document()`.
pub fn selected_sketch_document() -> DocumentSnapshot {
    let mut sketches = DocumentSketches::default();
    sketches
        .add(SKETCH, "Profile", SketchPlane::xy(), rectangle())
        .unwrap();
    let mut document = document_from_sketches(sketches);
    solve_selected_sketches(&mut document);
    document
}
fn rectangle() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    add_ring(
        &mut geometry,
        1,
        101,
        [[0., 0.], [4., 0.], [4., 3.], [0., 3.]],
    );
    add_rectangle_constraints(&mut geometry);
    geometry
}
fn add_ring(geometry: &mut Sketch2D, first_point: u64, first_line: u64, positions: [[f64; 2]; 4]) {
    for (index, position) in positions.into_iter().enumerate() {
        geometry
            .add_point(SketchEntityId(first_point + index as u64), position)
            .unwrap();
    }
    for index in 0..4 {
        let endpoints = [first_point + index, first_point + (index + 1) % 4].map(SketchEntityId);
        geometry
            .add_line(SketchEntityId(first_line + index), endpoints)
            .unwrap();
    }
}
/// Select four authored lines in traversal order; example: `selected_sketch_line_ring(101)`.
pub fn selected_sketch_line_ring(first_line: u64) -> Vec<SketchDirectedLine> {
    (first_line..first_line + 4)
        .map(|id| SketchDirectedLine {
            line: SketchEntityId(id),
            reversed: false,
        })
        .collect()
}
/// Select the rectangle boundary without inferred geometry; example: `selected_sketch_boundary()`.
pub fn selected_sketch_boundary() -> SketchPrismSelection {
    SketchPrismSelection {
        outer: selected_sketch_line_ring(101),
        holes: Vec::new(),
    }
}
/// Refresh numerical evidence after an authored edit; example: `solve_selected_sketches(&mut document)`.
pub fn solve_selected_sketches(document: &mut DocumentSnapshot) {
    document.sketches = document
        .sketches
        .solve_pending(&document.parameters, Default::default(), || false)
        .candidate;
}
/// Add a fully constrained inner boundary; example: `add_fixed_sketch_hole(&mut document)` before solving.
pub fn add_fixed_sketch_hole(document: &mut DocumentSnapshot) {
    let mut geometry = document.sketches.get(SKETCH).unwrap().geometry().clone();
    let positions = [[1., 1.], [1., 2.], [3., 2.], [3., 1.]];
    add_ring(&mut geometry, 11, 111, positions);
    for (index, position) in positions.into_iter().enumerate() {
        geometry
            .add_constraint(
                SketchConstraintId(11 + index as u64),
                PlanarConstraint::Fix {
                    point: SketchEntityId(11 + index as u64),
                    position,
                },
            )
            .unwrap();
    }
    document
        .sketches
        .replace_geometry(SKETCH, geometry)
        .unwrap();
}
/// Exercise traversal independent of source direction; example: `reverse_profile_source_line(&mut document, 102)`.
pub fn reverse_profile_source_line(document: &mut DocumentSnapshot, line: u64) {
    let mut recipe = document.sketches.get(SKETCH).unwrap().geometry().recipe();
    recipe
        .lines
        .iter_mut()
        .find(|curve| curve.id == SketchEntityId(line))
        .unwrap()
        .endpoints
        .reverse();
    document
        .sketches
        .replace_geometry(SKETCH, Sketch2D::from_recipe(&recipe).unwrap())
        .unwrap();
}

fn parameters() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", oblikovati_api::types::USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "height", oblikovati_api::types::USER_PARAM, "3 cm")
        .unwrap();
    parameters
}

fn document_from_sketches(sketches: DocumentSketches) -> DocumentSnapshot {
    DocumentSnapshot {
        sketches,
        parameters: parameters(),
        work_geometry: Default::default(),
        assets: Default::default(),
        body_names: Default::default(),
        sketch_settings: None,
        properties: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    }
}

fn add_rectangle_constraints(geometry: &mut Sketch2D) {
    let fixed = PlanarConstraint::Fix {
        point: SketchEntityId(1),
        position: [0.; 2],
    };
    let constraints = std::iter::once(fixed)
        .chain(axis_constraints())
        .chain(dimension_constraints());

    for (index, definition) in constraints.enumerate() {
        geometry
            .add_constraint(SketchConstraintId(index as u64 + 1), definition)
            .unwrap();
    }
}

fn axis_constraints() -> [PlanarConstraint; 4] {
    let ids = [1, 2, 3, 4].map(SketchEntityId);
    [
        PlanarConstraint::Horizontal {
            points: [ids[0], ids[1]],
        },
        PlanarConstraint::Vertical {
            points: [ids[1], ids[2]],
        },
        PlanarConstraint::Vertical {
            points: [ids[0], ids[3]],
        },
        PlanarConstraint::Horizontal {
            points: [ids[2], ids[3]],
        },
    ]
}

fn dimension_constraints() -> [PlanarConstraint; 2] {
    let ids = [1, 2, 3].map(SketchEntityId);
    [
        PlanarConstraint::Distance {
            points: [ids[0], ids[1]],
            parameter: 1,
            measure: DistanceMeasure::Horizontal,
        },
        PlanarConstraint::Distance {
            points: [ids[1], ids[2]],
            parameter: 2,
            measure: DistanceMeasure::Vertical,
        },
    ]
}

/// Capture a fully constrained stored prism recipe; example: `selected_prism_document()` for session tests.
pub fn selected_prism_document() -> DocumentSnapshot {
    use oblikovati_model::feature::{AuthoredFeature, FeatureId};
    let mut document = selected_sketch_document();
    document
        .parameters
        .insert_expression(3, "depth", oblikovati_api::types::USER_PARAM, "2 cm")
        .unwrap();
    let feature = AuthoredFeature::sketch_prism(
        FeatureId(1),
        "Prism",
        selected_prism_definition(),
        &document,
    )
    .unwrap();
    document.work_geometry.append(feature).unwrap();
    document
}
/// Share the parameter-backed stored source without evaluated coordinates; example: `selected_prism_definition().recipe()`.
pub fn selected_prism_definition() -> oblikovati_model::feature::sketch_prism::SketchPrismDefinition
{
    use oblikovati_model::feature::{
        sketch_prism::SketchPrismDefinition, work_geometry::DatumLength,
    };
    SketchPrismDefinition::new(
        SKETCH,
        selected_sketch_boundary(),
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(3),
        ],
    )
    .unwrap()
}
