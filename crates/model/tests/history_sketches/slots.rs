// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::SketchSlotCreation,
    sketch::{SketchPathMember, SketchSlotPlacement},
};
fn slot_input(curved: bool) -> SketchSlotCreation {
    SketchSlotCreation {
        placement: if curved {
            SketchSlotPlacement::Arc {
                center: [0.0; 2],
                endpoints: [[4.0, 0.0], [0.0, 4.0]],
                counter_clockwise: true,
            }
        } else {
            SketchSlotPlacement::Centers([[0.0; 2], [4.0, 0.0]])
        },
        width: 2.0,
        construction: true,
    }
}
#[test]
fn slots_share_boundary_points_and_keep_separate_centers_through_history_and_recipe() {
    for curved in [false, true] {
        let mut history = super::polyline::polyline_history();
        let before = history.snapshot().sketches.clone();
        let input = slot_input(curved);
        let layout = input.placement.layout(input.width).unwrap();
        let token = history.begin("Slot").unwrap();
        let report = history
            .edit(&token, |snapshot, ids| {
                snapshot.create_sketch_slot(DocumentSketchId(1), &input, ids)
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        assert_eq!(report.points.len(), if curved { 8 } else { 6 });
        verify_owned_slot(history.snapshot(), &layout, &report);
        let after = history.snapshot().sketches.clone();
        history.undo().unwrap();
        assert!(before.same_recipe(&history.snapshot().sketches));
        history.redo().unwrap();
        assert!(after.same_recipe(&history.snapshot().sketches));
    }
}
fn verify_owned_slot(
    snapshot: &DocumentSnapshot,
    layout: &oblikovati_model::sketch::SketchSlotLayout,
    report: &oblikovati_model::history::SketchSlotCreationReport,
) {
    let geometry = snapshot.sketches.get(report.sketch).unwrap().geometry();
    assert!(geometry.constraints().is_empty());
    assert_eq!(geometry.entity_ids().len(), 4);
    for (member, id) in layout.members.iter().zip(&report.members) {
        assert!(geometry.is_construction(*id).unwrap());
        match member {
            SketchPathMember::Line(pair) => assert_eq!(
                geometry.line(*id).unwrap().endpoints,
                pair.map(|i| report.points[i])
            ),
            SketchPathMember::Arc {
                center,
                endpoints,
                counter_clockwise,
            } => {
                let arc = geometry.arc(*id).unwrap();
                assert_eq!(
                    (arc.center, arc.start, arc.end, arc.counter_clockwise),
                    (
                        report.points[*center],
                        report.points[endpoints[0]],
                        report.points[endpoints[1]],
                        *counter_clockwise
                    )
                );
            }
        }
    }
    assert!(geometry.same_recipe(&Sketch2D::from_recipe(&geometry.recipe()).unwrap()));
}
#[test]
fn every_slot_allocation_failure_is_atomic_and_retires_reserved_ids() {
    for curved in [false, true] {
        for remaining in 0..if curved { 12 } else { 10 } {
            let mut history = super::polyline::polyline_history()
                .with_sketch_reservations(SketchReservationMarks {
                    sketch: 1,
                    entity: u64::MAX - remaining,
                    ..Default::default()
                })
                .unwrap();
            let before = history.snapshot().sketches.clone();
            let token = history.begin("Exhaust slot").unwrap();
            assert!(
                history
                    .edit(&token, |snapshot, ids| snapshot.create_sketch_slot(
                        DocumentSketchId(1),
                        &slot_input(curved),
                        ids
                    ))
                    .is_err()
            );
            assert!(before.same_recipe(&history.snapshot().sketches));
            assert_eq!(history.sketch_reservations().entity, u64::MAX);
        }
    }
}
