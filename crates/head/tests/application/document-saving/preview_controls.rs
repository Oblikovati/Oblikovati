// SPDX-License-Identifier: GPL-2.0-only
//! Locate an action rendered beside a Save As preview source.
use super::super::{diagnostics, world::squeezed};
use super::ApplicationWorld;

pub(super) fn position(world: &ApplicationWorld, label: &str, action: &str) -> [i32; 2] {
    let option = world.run(label);
    let button = world.run(action);
    if option == button {
        // Adjacent labels can share a glyph run. Click inside the last label.
        let text = format!("{}{}", squeezed(label), squeezed(action));
        assert_eq!(button.text, text);
        return diagnostics::pointer_pixels([button.maximum[0] - 2., button.center()[1]]);
    }
    assert!(
        button.list == option.list
            && button.minimum[0] > option.maximum[0]
            && (button.center()[1] - option.center()[1]).abs() < 4.,
        "{action} {button:?}: expected the control beside {option:?}"
    );
    diagnostics::pointer_pixels(button.center())
}
