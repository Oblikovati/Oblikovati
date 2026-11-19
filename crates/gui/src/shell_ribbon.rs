// SPDX-License-Identifier: GPL-2.0-only
//! One Shell command and its persistent ribbon placement (#4106).
use crate::GuiShellAction;
pub(crate) const ID: &str = "part.model.modify.shell";
pub(crate) const LABEL: &str = "Shell";

pub(crate) fn is_expanded(
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> bool {
    customization.builtin_expanded(ID)
}

pub(crate) fn additional_action(label: &str) -> Option<GuiShellAction> {
    match label {
        "Move Bodies" => Some(GuiShellAction::OpenMoveBody),
        LABEL => Some(GuiShellAction::OpenShellFeature),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn expanded_commands_keep_their_real_actions() {
        assert_eq!(
            additional_action("Shell"),
            Some(GuiShellAction::OpenShellFeature)
        );
        assert_eq!(
            additional_action("Move Bodies"),
            Some(GuiShellAction::OpenMoveBody)
        );
        assert_eq!(additional_action("Bend Part"), None);
    }
}
