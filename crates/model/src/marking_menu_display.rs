// SPDX-License-Identifier: GPL-2.0-only
//! Saved presentation of configured viewport menus.
use serde::{Deserialize, Serialize};

/// Text and icon visibility for radial command buttons.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum MarkingMenuStyle {
    #[default]
    IconAndText,
    IconOnly,
    TextOnly,
}

/// Which command rows accompany the radial buttons.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum MarkingMenuOverflow {
    /// Include the configured rows and applicable canvas context commands.
    #[default]
    Full,
    /// Limit the marking menu to the configured radial buttons.
    RadialOnly,
    /// Include only the configured radial buttons and overflow rows.
    Short,
}

/// Application preferences for displaying marking menus.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct MarkingMenuDisplay {
    pub classic: bool,
    pub style: MarkingMenuStyle,
    pub overflow: MarkingMenuOverflow,
}
