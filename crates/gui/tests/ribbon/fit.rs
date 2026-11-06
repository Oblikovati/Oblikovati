// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn exact_fit_keeps_all_panels_expanded() {
    for available in [620., 5000.] {
        assert_eq!(
            expanded_panel_count(&[100., 200., 300.], 10., 50., available),
            3
        );
    }
}
#[test]
fn panels_collapse_from_the_right_at_each_boundary() {
    for (available, expected) in [(619., 2), (370., 2), (369., 1), (220., 1), (219., 0)] {
        assert_eq!(
            expanded_panel_count(&[100., 200., 300.], 10., 50., available),
            expected
        );
    }
}
#[test]
fn narrower_than_the_collapsed_row_requires_the_scroll_tier() {
    assert_eq!(expanded_panel_count(&[100., 200., 300.], 10., 50., 40.), 0);
}
#[test]
fn unmeasured_panels_expand_once_to_obtain_their_widths() {
    assert_eq!(expanded_panel_count(&[0.; 3], 10., 50., 30.), 3);
}
#[test]
fn empty_tabs_and_single_panels_have_no_extra_separator() {
    assert_eq!(expanded_panel_count(&[], 10., 50., 500.), 0);
    assert_eq!(expanded_panel_count(&[100.], 10., 50., 100.), 1);
    assert_eq!(expanded_panel_count(&[100.], 10., 50., 99.), 0);
}
#[test]
fn resizing_restores_the_full_row_without_changing_panel_order() {
    let widths = [100., 200., 300.];
    let counts: Vec<usize> = [620., 619., 369., 40., 620.]
        .into_iter()
        .map(|width| expanded_panel_count(&widths, 10., 50., width))
        .collect();
    assert_eq!(counts, [3, 2, 1, 0, 3]);
    assert_eq!(widths, [100., 200., 300.]);
}

#[test]
fn each_pair_has_one_separator_in_every_collapse_state() {
    for (expanded, expected) in [(0, 170.), (1, 220.), (2, 370.), (3, 620.)] {
        assert_eq!(
            ribbon_row_width(&[100., 200., 300.], expanded, 10., 50.),
            expected
        );
    }
    assert_eq!(ribbon_row_width(&[], 0, 10., 50.), 0.);
    assert_eq!(ribbon_row_width(&[100.], 1, 10., 50.), 100.);
}
#[test]
fn popup_anchor_moves_left_only_when_its_measured_width_needs_it() {
    for (anchor, width, viewport, expected) in [
        (10., 400., 3840., 10.),
        (3700., 400., 3840., 3440.),
        (10., 5000., 3840., 0.),
        (3700., 0., 3840., 3700.),
    ] {
        assert_eq!(clamp_panel_flyout_x(anchor, width, viewport), expected);
    }
}
