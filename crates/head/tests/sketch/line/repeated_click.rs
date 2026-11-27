// SPDX-License-Identifier: GPL-2.0-only
//! The recorded repeated-click trials need their captured sketch and view (#3933).
use super::super::super::{ApplicationWorld, gap_runner};
use cucumber::given;

const MISSING_FIXTURE: &str = "line-02 and line-07: captured sketch, view, input state and both pointer events are unavailable; see line/BLOCKERS.md";

#[given("an editable sketch with the captured geometry is active")]
#[given(
    "the recorded sketch fixture is active after the Line start point was clicked twice at the same screen coordinate"
)]
fn recorded_repeated_click_fixture(_: &mut ApplicationWorld) {
    gap_runner::block_fixture(MISSING_FIXTURE);
}

#[test]
fn recorded_repeated_click_fixture_reports_why_it_is_blocked() {
    let error =
        std::panic::catch_unwind(|| gap_runner::block_fixture(MISSING_FIXTURE)).unwrap_err();
    let message = error.downcast_ref::<String>().unwrap();
    assert!(message.contains("captured sketch") && message.contains("both pointer events"));
}
