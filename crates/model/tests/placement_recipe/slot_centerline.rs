// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn solved_slot_centerline_uses_actual_cap_centers_and_outer_winding() {
    for ccw in [false, true] {
        let recipe =
            SketchPlacementRecipe::arc_slot([0., 0.], [[5., 0.], [0., 5.]], 2., ccw).unwrap();
        let (history, report) = super::rectangles::recipe_history(&recipe, &[]);
        let geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry();
        let index = geometry.read_index();
        let seed = index
            .arc_slot_centerline(report.members[0], [report.members[3], report.members[1]])
            .unwrap();
        assert_eq!(seed.center, [0., 0.]);
        assert_eq!(seed.radius, 5.);
        assert_eq!(seed.arc, Some(([[5., 0.], [0., 5.]], ccw)));
        for (outer, caps) in [
            (report.points[0], [report.members[3], report.members[1]]),
            (report.members[0], [report.points[0], report.members[1]]),
            (report.members[0], [report.members[0], report.members[1]]),
            (
                oblikovati_model::sketch::SketchEntityId(u64::MAX),
                [report.members[3], report.members[1]],
            ),
        ] {
            assert!(index.arc_slot_centerline(outer, caps).is_err());
        }
    }
}
