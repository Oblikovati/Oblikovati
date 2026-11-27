// SPDX-License-Identifier: GPL-2.0-only
//! Convert close-prompt accessibility bounds to framebuffer coordinates.

pub(super) fn scale_optional(
    prompt: Option<crate::GuiClosePrompt>,
    scale: [f32; 2],
) -> Option<crate::GuiClosePrompt> {
    prompt.map(|prompt| scale_prompt(prompt, scale))
}

fn scale_prompt(mut prompt: crate::GuiClosePrompt, scale: [f32; 2]) -> crate::GuiClosePrompt {
    for bounds in std::iter::once(&mut prompt.bounds)
        .chain(prompt.buttons.iter_mut().map(|button| &mut button.bounds))
    {
        for axis in 0..2 {
            bounds[axis] *= scale[axis];
            bounds[axis + 2] *= scale[axis];
        }
    }
    prompt
}

#[cfg(test)]
mod tests {
    #[test]
    fn scales_prompt_and_buttons_on_both_axes() {
        let prompt = crate::GuiClosePrompt {
            title: "Confirm",
            message: "Close?".into(),
            bounds: [1., 2., 3., 4.],
            buttons: vec![crate::GuiClosePromptButton {
                label: "Cancel",
                choice: crate::GuiClosePromptChoice::Cancel,
                bounds: [2., 3., 4., 5.],
                focused: true,
            }],
        };
        let scaled = super::scale_optional(Some(prompt), [2., 3.]).unwrap();
        assert!(super::scale_optional(None, [2., 3.]).is_none());
        assert_eq!(scaled.bounds, [2., 6., 6., 12.]);
        assert_eq!(scaled.buttons[0].bounds, [4., 9., 8., 15.]);
        assert!(scaled.buttons[0].focused);
    }
}
