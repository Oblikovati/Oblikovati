// SPDX-License-Identifier: GPL-2.0-only
//! One Shell command and its persistent ribbon placement (#4106).
pub(crate) const ID: &str = "part.model.modify.shell";
pub(crate) const LABEL: &str = "Shell";

pub(crate) fn is_expanded(
    customization: &crate::application::ribbon_customization::RibbonCustomization,
) -> bool {
    customization.builtin_expanded(ID)
}
