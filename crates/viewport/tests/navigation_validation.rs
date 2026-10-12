// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn stationary_navigation_does_not_invalidate_a_completed_camera() {
    for command in [
        ViewportNavigation::Pan([0, 0]),
        ViewportNavigation::Orbit { yaw: 0., pitch: 0. },
        ViewportNavigation::Zoom {
            factor: 1.,
            anchor: Some([12, 14]),
        },
    ] {
        command.validate([64, 48]).unwrap();
        assert!(command.is_stationary());
    }
    assert!(!ViewportNavigation::Pan([1, 0]).is_stationary());
}
fn require_navigation_decline(command: ViewportNavigation) {
    let error = command.validate([64, 48]).unwrap_err();
    assert!(error.contains(&format!("{command:?}")));
    assert!(error.contains("expected finite camera movement"));
}
#[test]
fn nonfinite_orbits_and_unrepresentable_pointer_moves_are_refused() {
    require_navigation_decline(ViewportNavigation::Pan([0, i32::MIN]));
    require_navigation_decline(ViewportNavigation::Orbit {
        yaw: f64::NAN,
        pitch: 0.,
    });
    require_navigation_decline(ViewportNavigation::Orbit {
        yaw: 0.,
        pitch: f64::INFINITY,
    });
    require_navigation_decline(ViewportNavigation::Zoom {
        factor: 0.9,
        anchor: Some([0, i32::MIN]),
    });
}
#[test]
fn nonpositive_nonfinite_and_unrepresentable_zoom_factors_are_refused() {
    for factor in [0., -1., f64::INFINITY, f64::from_bits(1)] {
        require_navigation_decline(ViewportNavigation::Zoom {
            factor,
            anchor: None,
        });
    }
}
#[test]
fn valid_zoom_anchors_include_captured_pointer_positions_outside_the_window() {
    assert!(valid_zoom_anchor(None, [64, 48]));
    assert!(valid_zoom_anchor(Some([-20, 90]), [64, 48]));
    assert!(!valid_zoom_anchor(Some([0, 0]), [64, u32::MAX]));
    ViewportNavigation::Pan([-10, 20])
        .validate([64, 48])
        .unwrap();
    ViewportNavigation::Zoom {
        factor: 0.9,
        anchor: None,
    }
    .validate([64, 48])
    .unwrap();
}

#[test]
fn framing_margins_require_a_finite_fraction_less_than_one() {
    for margin in [-0.1, 1., f64::INFINITY, f64::NAN] {
        require_navigation_decline(ViewportNavigation::Fit { margin });
        require_navigation_decline(ViewportNavigation::Home { margin });
    }
    for margin in [0., 0.1, 0.99] {
        for command in [
            ViewportNavigation::Fit { margin },
            ViewportNavigation::Home { margin },
        ] {
            command.validate([64, 48]).unwrap();
            assert!(!command.is_stationary());
        }
    }
}
