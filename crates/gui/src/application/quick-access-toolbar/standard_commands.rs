// SPDX-License-Identifier: GPL-2.0-only
//! Ordered standard commands for the quick-access toolbar.

/// Commands the customization menu lists, in its order.
pub(super) const STANDARD: [StandardCommand; 15] = [
    StandardCommand::New,
    StandardCommand::Open,
    StandardCommand::Save,
    StandardCommand::Undo,
    StandardCommand::Redo,
    StandardCommand::Home,
    StandardCommand::Projects,
    StandardCommand::Update,
    StandardCommand::Select,
    StandardCommand::MaterialBrowser,
    StandardCommand::Materials,
    StandardCommand::AppearanceBrowser,
    StandardCommand::Parameters,
    StandardCommand::Print,
    StandardCommand::Measure,
];

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(super) enum StandardCommand {
    New,
    Open,
    Save,
    Undo,
    Redo,
    Home,
    Projects,
    Update,
    Select,
    MaterialBrowser,
    Materials,
    AppearanceBrowser,
    Parameters,
    Print,
    Measure,
}
impl StandardCommand {
    pub(super) fn label(self) -> &'static str {
        match self {
            Self::New => "New",
            Self::Open => "Open",
            Self::Save => "Save",
            Self::Undo => "Undo",
            Self::Redo => "Redo",
            Self::Home => "Home",
            Self::Projects => "Projects",
            Self::Update => "Update",
            Self::Select => "Select",
            Self::MaterialBrowser => "Material Browser",
            Self::Materials => "Materials",
            Self::AppearanceBrowser => "Appearance Browser",
            Self::Parameters => "Parameters",
            Self::Print => "Print",
            Self::Measure => "Measure",
        }
    }
    pub(super) fn default_shown(self) -> bool {
        !matches!(self, Self::Print | Self::Measure)
    }
}
