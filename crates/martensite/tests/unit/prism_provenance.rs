// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbo::profile_extrusion::{ProfileCurveId, ProfileVertexId};
fn entry(source: HistorySource, role: HistoryRole, result: ResultEntity) -> HistoryRecord {
    HistoryRecord {
        source,
        role,
        result,
        relation: HistoryRelation::Generated,
    }
}
#[test]
fn incompatible_result_role_or_parent_is_refused() {
    let edge = ResultEntity::Edge(2);
    let invalid_parent = entry(
        HistorySource::Vertex(ProfileVertexId(4)),
        HistoryRole::StartEdge,
        edge,
    );
    assert!(semantic_role(&invalid_parent, edge).is_err());
    let invalid_role = entry(
        HistorySource::Curve(ProfileCurveId(7)),
        HistoryRole::EndVertex,
        edge,
    );
    assert!(semantic_role(&invalid_role, edge).is_err());
    let seam = entry(
        HistorySource::Curve(ProfileCurveId(7)),
        HistoryRole::LateralEdge,
        edge,
    );
    assert_eq!(
        semantic_role(&seam, edge).unwrap(),
        (
            PrismHistorySource::Curve(7),
            PrismSubentityRole::LateralEdge
        )
    );
}
#[test]
fn duplicate_source_role_pairs_are_refused_before_naming() {
    let entry = PrismSubentityProvenance {
        source: PrismHistorySource::Vertex(1),
        role: PrismSubentityRole::StartVertex,
        relation: PrismHistoryRelation::Generated,
        point: [0.; 3],
    };
    assert!(
        ordered(vec![entry.clone(), entry])
            .unwrap_err()
            .to_string()
            .contains("unique source-role")
    );
}
#[test]
fn circular_edge_representative_lies_on_the_curve() {
    use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Point3};
    use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeEdge;
    let axes = Axis2::new(
        Point3::new(0., 0., 0.),
        Direction3::new(0., 0., 1.).unwrap(),
    )
    .unwrap();
    let edge = BRepBuilderAPIMakeEdge::from_circ(Circle3::new(axes, 2.).unwrap())
        .unwrap()
        .edge()
        .unwrap();
    let point = edge_midpoint(&edge.into()).unwrap();
    assert!((point[0].hypot(point[1]) - 2.).abs() < 1e-10, "{point:?}");
    assert!(point[2].abs() < 1e-10);
}
