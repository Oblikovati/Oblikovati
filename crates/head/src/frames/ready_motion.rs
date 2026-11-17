// SPDX-License-Identifier: GPL-2.0-only
//! Reduce unseen idle sketch pointer positions while preserving input boundaries (#4035).
use super::*;
use oblikovati_viewport::ViewportInput;

pub(super) fn owner_is_ready(owner: &dyn HeadWork) -> Result<bool, String> {
    let state = owner.gui_command_state()?;
    Ok(
        state.authoring == oblikovati_gui::GuiAuthoringContext::Sketch
            && state.active_sketch.is_some()
            && state.sketch_at_ready
            && !state.transaction_open,
    )
}

pub(super) fn tail(
    inputs: &[ViewportEvent],
    start: usize,
    allowed: impl Fn([i32; 2]) -> bool,
) -> usize {
    let Some(ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers,
    })) = inputs.get(start)
    else {
        return start;
    };
    if !allowed(*position) {
        return start;
    }
    let mut end = start;
    for event in &inputs[start + 1..] {
        match event {
            ViewportEvent::Input(ViewportInput::Motion {
                position,
                modifiers: next,
            }) if next == modifiers && allowed(*position) => end += 1,
            _ => break,
        }
    }
    end
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_viewport::{ViewportButton, ViewportModifiers};
    fn motion(x: i32, shift: bool) -> ViewportEvent {
        ViewportEvent::Input(ViewportInput::Motion {
            position: [x, 3],
            modifiers: ViewportModifiers {
                shift,
                ..Default::default()
            },
        })
    }
    #[test]
    fn reduces_only_adjacent_permitted_motions_with_identical_modifiers() {
        let button = ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: true,
            position: [4, 3],
            modifiers: Default::default(),
        });
        let events = [
            motion(1, false),
            motion(2, false),
            motion(3, true),
            motion(4, true),
            button,
            motion(5, true),
        ];
        assert_eq!(tail(&events, 0, |_| true), 1);
        assert_eq!(tail(&events, 2, |_| true), 3);
        assert_eq!(tail(&events, 4, |_| true), 4);
        assert_eq!(tail(&events, 0, |point| point[0] < 2), 0);
        assert_eq!(tail(&events, 0, |_| false), 0);
    }
}
