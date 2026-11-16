// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::{DocumentSketches, SketchPointProjection},
    feature::{work_geometry::WorkPointReference, work_plane::WorkPlaneOrigin},
    sketch::{
        FixedCoordinateAssumption, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchRelationId,
    },
};

#[test]
fn profile_mapping_borrows_a_current_canonical_root() {
    let mut document = selected_sketch_document();
    let mut shape = Sketch2D::default();
    let anchor = SketchEntityId(1);
    let endpoint = SketchEntityId(2);
    shape.add_point(anchor, [0., 0.]).unwrap();
    shape.add_point(endpoint, [1., 0.]).unwrap();
    shape
        .add_line(SketchEntityId(10), [anchor, endpoint])
        .unwrap();
    shape
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: endpoint,
                position: [1., 0.],
            },
        )
        .unwrap();
    let mut sketches = DocumentSketches::default();
    sketches
        .add_on_origin_plane(SKETCH, "Profile", WorkPlaneOrigin::XY, shape)
        .unwrap();
    sketches
        .set_point_projections(
            SKETCH,
            vec![SketchPointProjection {
                anchor,
                reference: WorkPointReference::Origin,
            }],
        )
        .unwrap();
    document.sketches = sketches;
    let batch = document
        .solve_one_document_sketch(SKETCH, Default::default(), || false)
        .unwrap();
    let outcome = batch.report.reports[0].result.as_ref().unwrap().clone();
    document.sketches = batch.candidate;
    let projection = document
        .canonical_origin_projection(SKETCH, anchor, &outcome)
        .unwrap();
    let bounds = [[-0.01, 0.01], [-0.01, 0.01], [0.99, 1.01], [-0.01, 0.01]];
    let fixed = [
        FixedCoordinateAssumption {
            relation: SketchRelationId::Authored(SketchConstraintId(1)),
            equation: 0,
            value: 1.,
        },
        FixedCoordinateAssumption {
            relation: SketchRelationId::Authored(SketchConstraintId(1)),
            equation: 1,
            value: 0.,
        },
    ];
    let root = projection.conditional_root_auto(&bounds, &fixed).unwrap();
    let mapping = oblikovati_martensite::conditional_profile_mapping(&root).unwrap();
    assert_eq!(mapping.lines()[0].curve, SketchEntityId(10));
    assert_eq!(mapping.lines()[0].endpoints, [anchor, endpoint]);
    assert_eq!(
        mapping
            .vertices()
            .iter()
            .map(|vertex| vertex.vertex)
            .collect::<Vec<_>>(),
        [anchor, endpoint]
    );
    assert!(
        mapping
            .vertices()
            .iter()
            .all(|vertex| vertex.displacement_upper >= 0.)
    );
}
