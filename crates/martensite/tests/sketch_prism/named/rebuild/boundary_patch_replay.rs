// SPDX-License-Identifier: GPL-2.0-only
//! Martensite unit and ordered replay tests for Surface Boundary Patch (#3918).
use super::*;
use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, chain_sketch_boundary_loop, is_interior_diagonal_curve,
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureEvaluationStatus,
        boundary_patch::{
            BoundaryCondition, BoundaryCurveRef, BoundaryEdgeSet, BoundaryPatchDefinition,
            BoundaryPatchRecipe,
        },
        work_plane::WorkPlaneOrigin,
    },
    sketch::{Sketch2D, SketchEntityId},
};

fn rect_sketch(with_diagonal: bool) -> Sketch2D {
    let mut g = Sketch2D::default();
    let corners = [(-0.6, 0.0), (0.6, 0.0), (0.6, 0.5), (-0.6, 0.5)];
    for (i, (x, y)) in corners.iter().enumerate() {
        g.add_point(SketchEntityId((2 + i) as u64), [*x, *y])
            .unwrap();
    }
    for i in 0..4 {
        let a = SketchEntityId((2 + i) as u64);
        let b = SketchEntityId((2 + (i + 1) % 4) as u64);
        g.add_line(SketchEntityId((6 + i) as u64), [a, b]).unwrap();
    }
    if with_diagonal {
        g.add_line(SketchEntityId(10), [SketchEntityId(2), SketchEntityId(4)])
            .unwrap();
    }
    g
}

fn five_loops_document() -> DocumentSnapshot {
    let mut snapshot = selected_prism_document();
    snapshot.work_geometry = Default::default();
    snapshot.sketches = Default::default();
    let mut g = Sketch2D::default();
    let origins = [
        (-2.8, 0.8),
        (-0.6, 0.8),
        (1.6, 0.8),
        (-1.7, -0.8),
        (0.5, -0.8),
    ];
    let mut next_id = 10u64;
    for (ox, oy) in origins {
        let pts = [
            (ox, oy),
            (ox + 1.2, oy),
            (ox + 1.2, oy + 0.5),
            (ox, oy + 0.5),
        ];
        let p0 = next_id;
        for (k, (x, y)) in pts.iter().enumerate() {
            g.add_point(SketchEntityId(p0 + k as u64), [*x, *y])
                .unwrap();
        }
        next_id += 4;
        for k in 0..4 {
            let a = SketchEntityId(p0 + k as u64);
            let b = SketchEntityId(p0 + ((k + 1) % 4) as u64);
            g.add_line(SketchEntityId(next_id), [a, b]).unwrap();
            next_id += 1;
        }
    }
    snapshot
        .sketches
        .add_on_origin_plane(DocumentSketchId(9), "Sketch9", WorkPlaneOrigin::XY, g)
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    snapshot
}

#[test]
fn boundary_patch_chains_four_edges_and_rejects_interior_diagonal() {
    let g = rect_sketch(true);
    assert!(is_interior_diagonal_curve(&g, SketchEntityId(10)));
    for id in 6..=9 {
        assert!(!is_interior_diagonal_curve(&g, SketchEntityId(id)));
    }
    let chained = chain_sketch_boundary_loop(&g, SketchEntityId(6));
    assert_eq!(chained.len(), 4);
    assert!(!chained.contains(&SketchEntityId(10)));
    assert!(chain_sketch_boundary_loop(&g, SketchEntityId(10)).is_empty());
}

#[test]
fn boundary_patch_builds_five_surface_bodies_and_persists_condition_weight() {
    let mut history = owner(five_loops_document());
    let sketch = DocumentSketchId(9);
    // Each loop has 4 lines after its 4 points: starting at 14, 22, 30, 38, 46.
    for (idx, line_start) in [14u64, 22, 30, 38, 46].into_iter().enumerate() {
        let edges: Vec<BoundaryCurveRef> = (0..4)
            .map(|k| BoundaryCurveRef::SketchCurve {
                sketch,
                entity: SketchEntityId(line_start + k),
            })
            .collect();
        let (condition, weight) = if idx == 4 {
            (BoundaryCondition::Tangent, 0.7)
        } else {
            (BoundaryCondition::Free, 1.0)
        };
        let def = BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
            version: 1,
            automatic_edge_chain: true,
            edge_sets: vec![BoundaryEdgeSet {
                edges,
                condition,
                weight,
            }],
            guide_rails: vec![],
        })
        .unwrap();
        let token = history
            .begin(&format!("Boundary Patch{}", idx + 1))
            .unwrap();
        history
            .edit(&token, |s, ids| {
                s.create_boundary_patch(&format!("Boundary Patch{}", idx + 1), def, ids)
            })
            .unwrap();
        history.commit(&token, true).unwrap();
    }
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let eval = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let surfaces: Vec<_> = eval.ordered_bodies().filter(|b| b.is_surface()).collect();
    assert_eq!(
        surfaces.len(),
        5,
        "expected five surface bodies Srf1..Srf5; health={:?}",
        eval.health()
    );
    for srf in surfaces {
        let faces = srf.face_reference_keys(&history, document()).unwrap();
        assert_eq!(faces.len(), 1);
        let area = srf.face_area(&history, document(), &faces[0]).unwrap();
        assert!(
            (area - 0.6).abs() < 1e-4,
            "expected 1.2 * 0.5 = 0.6 cm^2, got {area}"
        );
    }
    // Round-trip part recipe at Program v70, Source v60.
    let program = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(program.version, 70);
    let restored = oblikovati_model::feature::FeatureSequence::from_part_recipe(&program).unwrap();
    let fifth = restored.features()[4].definition();
    let PartFeatureDefinition::BoundaryPatch(patch5) = fifth else {
        panic!("expected BoundaryPatch definition");
    };
    assert_eq!(patch5.edge_sets()[0].condition, BoundaryCondition::Tangent);
    assert!((patch5.edge_sets()[0].weight - 0.7).abs() < 1e-12);
}

#[test]
fn boundary_patch_rejects_single_edge_and_builds_two_edge_and_guide_rail_surface() {
    let mut snapshot = selected_prism_document();
    snapshot.work_geometry = Default::default();
    snapshot.sketches = Default::default();
    snapshot
        .sketches
        .add_on_origin_plane(
            DocumentSketchId(1),
            "Sketch1",
            WorkPlaneOrigin::XY,
            rect_sketch(false),
        )
        .unwrap();
    solve_selected_sketches(&mut snapshot);
    let mut history = owner(snapshot);

    // Single edge fails evaluation.
    let one_edge = BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
        version: 1,
        automatic_edge_chain: false,
        edge_sets: vec![BoundaryEdgeSet {
            edges: vec![BoundaryCurveRef::SketchCurve {
                sketch: DocumentSketchId(1),
                entity: SketchEntityId(6),
            }],
            condition: BoundaryCondition::Free,
            weight: 1.0,
        }],
        guide_rails: vec![],
    })
    .unwrap();
    let token = history.begin("SingleEdge").unwrap();
    let f1 = history
        .edit(&token, |s, ids| {
            s.create_boundary_patch("Boundary Patch1", one_edge, ids)
        })
        .unwrap();
    history.commit(&token, true).unwrap();
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let eval1 = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    assert!(matches!(
        eval1.health()[0].status,
        FeatureEvaluationStatus::Failed(_)
    ));

    // Two edges + guide rail succeeds and builds a valid surface body.
    let two_with_rail = BoundaryPatchDefinition::from_recipe(BoundaryPatchRecipe {
        version: 1,
        automatic_edge_chain: false,
        edge_sets: vec![
            BoundaryEdgeSet {
                edges: vec![BoundaryCurveRef::SketchCurve {
                    sketch: DocumentSketchId(1),
                    entity: SketchEntityId(6),
                }],
                condition: BoundaryCondition::Free,
                weight: 1.0,
            },
            BoundaryEdgeSet {
                edges: vec![BoundaryCurveRef::SketchCurve {
                    sketch: DocumentSketchId(1),
                    entity: SketchEntityId(7),
                }],
                condition: BoundaryCondition::Free,
                weight: 1.0,
            },
        ],
        guide_rails: vec![BoundaryCurveRef::SketchCurve {
            sketch: DocumentSketchId(1),
            entity: SketchEntityId(8),
        }],
    })
    .unwrap();
    let token = history.begin("EditTwoWithRail").unwrap();
    history
        .edit(&token, |s, _| s.edit_boundary_patch(f1, two_with_rail))
        .unwrap();
    history.commit(&token, true).unwrap();
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let eval2 = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let body = eval2
        .body(f1)
        .unwrap_or_else(|| panic!("expected valid surface body; health={:?}", eval2.health()));
    assert!(body.is_surface());
    let faces = body.face_reference_keys(&history, document()).unwrap();
    let area = body.face_area(&history, document(), &faces[0]).unwrap();
    assert!((area - 0.6).abs() < 1e-4, "expected 0.6 cm^2, got {area}");
}
