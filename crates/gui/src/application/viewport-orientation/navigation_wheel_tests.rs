// SPDX-License-Identifier: GPL-2.0-only
//! Navigation wheel hit testing.
use super::*;
#[test]
fn the_full_ring_has_its_tools_where_a_pointer_finds_them() {
    assert_eq!(tool_at(WheelLayout::Full, [0., -0.4]), Some("Zoom"));
    assert_eq!(tool_at(WheelLayout::Full, [0.4, 0.]), Some("Orbit"));
    assert_eq!(tool_at(WheelLayout::Full, [0., -0.1]), Some("Center"));
    assert_eq!(tool_at(WheelLayout::Full, [0.1, 0.1]), Some("Walk"));
    assert_eq!(tool_at(WheelLayout::Full, [-0.1, 0.1]), Some("Look"));
    assert_eq!(tool_at(WheelLayout::Full, [-0.1, -0.1]), Some("Up Down"));
    assert_eq!(tool_at(WheelLayout::Full, [0.6, 0.]), None);
}
#[test]
fn the_mini_ring_has_rewind_on_its_right() {
    assert_eq!(
        tool_at(WheelLayout::MiniViewObject, [0.3, 0.]),
        Some("Rewind")
    );
    assert!(WheelLayout::MiniFull.tools().0.len() > WheelLayout::MiniViewObject.tools().0.len());
    assert!(WheelLayout::MiniFull.side() < WheelLayout::Full.side());
}
