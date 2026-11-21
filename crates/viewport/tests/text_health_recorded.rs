// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn text_input_health_requires_a_live_view_and_excludes_offscreen_targets() {
    let (runtime, _) = runtime();
    let mut viewport = runtime.open_offscreen([64, 48]).unwrap();
    assert_eq!(viewport.text_input_issue().unwrap(), None);
    viewport.close();
    assert!(viewport.text_input_issue().unwrap_err().contains("live"));
}
