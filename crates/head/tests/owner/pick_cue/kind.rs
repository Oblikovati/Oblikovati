// SPDX-License-Identifier: GPL-2.0-only
//! Pick cue types come from owned sketch identities.
use super::*;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};

#[test]
fn identifies_owned_point_and_line_without_geometric_guessing() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    sketch.add_point(SketchEntityId(2), [2., 0.]).unwrap();
    sketch
        .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
        .unwrap();
    assert_eq!(
        pick_kind(&sketch, SketchPickTarget::Entity(SketchEntityId(1))).unwrap(),
        "Point"
    );
    assert_eq!(
        pick_kind(&sketch, SketchPickTarget::Entity(SketchEntityId(3))).unwrap(),
        "Line"
    );
    assert!(pick_kind(&sketch, SketchPickTarget::Entity(SketchEntityId(99))).is_err());
}

#[test]
fn preserves_unrecognized_kind_for_diagnostics() {
    assert_eq!(kind_label("offsetSpline"), "Spline");
    assert_eq!(kind_label("futureKind"), "futureKind");
}

#[test]
fn cue_failure_does_not_claim_selection_failed() {
    let notice = DesktopCommandNotice::SelectionCueFailed {
        cause: "projection unavailable".into(),
    };
    assert_eq!(
        notice.message(),
        "Selection succeeded; pick cue unavailable: projection unavailable"
    );
}

#[test]
fn hover_failure_reports_the_hover_without_claiming_selection_failed() {
    let notice = DesktopCommandNotice::HoverFailed {
        cause: "projection unavailable".into(),
    };
    assert_eq!(
        notice.message(),
        "Sketch hover unavailable: projection unavailable"
    );
}
