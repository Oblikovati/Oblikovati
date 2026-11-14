// SPDX-License-Identifier: GPL-2.0-only
//! The zoom fixture selects a face with the selection mode shown in the UI.
use super::*;

#[test]
fn preselected_face_uses_a_visible_pick_and_face_selection_mode() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    shrink_the_block(&mut world);
    select_face(&mut world);
    assert!(world.state().model_face_chosen);
    world.close();
}
