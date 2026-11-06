// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn every_native_position_is_retained_for_undecided_incidence() {
    for (native, expected) in [
        (Position::Head, CurvePosition::Start),
        (Position::Middle, CurvePosition::Interior),
        (Position::End, CurvePosition::End),
    ] {
        let result = convert(Transition::undecided(native)).unwrap();
        assert_eq!(
            result,
            CurveTransition {
                position: expected,
                kind: TransitionKind::Undecided
            }
        );
    }
}
#[test]
fn crossing_direction_and_tangent_flag_are_not_recomputed() {
    for tangent in [false, true] {
        for (kind, expected) in [
            (TypeTrans::In, TransitionKind::In { tangent }),
            (TypeTrans::Out, TransitionKind::Out { tangent }),
        ] {
            let result = convert(Transition::crossing(tangent, Position::Middle, kind)).unwrap();
            assert_eq!(result.kind, expected);
        }
    }
}
#[test]
fn touch_side_and_relative_orientation_are_not_guessed() {
    for opposite in [false, true] {
        for (side, expected) in [
            (Situation::Inside, TouchSide::Inside),
            (Situation::Outside, TouchSide::Outside),
            (Situation::Unknown, TouchSide::Unknown),
        ] {
            let native = Transition::touch(true, Position::End, side, opposite);
            let result = convert(native).unwrap();
            assert_eq!(
                result.kind,
                TransitionKind::Touch {
                    tangent: true,
                    side: expected,
                    opposite
                }
            );
        }
    }
}
