// SPDX-License-Identifier: GPL-2.0-only
//! Ordered standard commands for the quick-access toolbar.

/// Commands the customization menu lists, in its order.
pub(super) const STANDARD: [StandardCommand; 22] = [
    StandardCommand::New,
    StandardCommand::Open,
    StandardCommand::Save,
    StandardCommand::Undo,
    StandardCommand::Redo,
    StandardCommand::Home,
    StandardCommand::Projects,
    StandardCommand::Update,
    StandardCommand::Select,
    StandardCommand::RefineAppearance,
    StandardCommand::MaterialBrowser,
    StandardCommand::Materials,
    StandardCommand::AppearanceBrowser,
    StandardCommand::AppearanceOverride,
    StandardCommand::AdjustAppearance,
    StandardCommand::Parameters,
    StandardCommand::DesignDoctor,
    StandardCommand::Print,
    StandardCommand::Measure,
    StandardCommand::IProperties,
    StandardCommand::Return,
    StandardCommand::ClearAppearance,
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
    RefineAppearance,
    MaterialBrowser,
    Materials,
    AppearanceBrowser,
    AppearanceOverride,
    AdjustAppearance,
    Parameters,
    DesignDoctor,
    Print,
    Measure,
    IProperties,
    Return,
    ClearAppearance,
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
            Self::RefineAppearance => "Refine Appearance",
            Self::MaterialBrowser => "Material Browser",
            Self::Materials => "Materials",
            Self::AppearanceBrowser => "Appearance Browser",
            Self::AppearanceOverride => "Appearance Override",
            Self::AdjustAppearance => "Adjust",
            Self::Parameters => "Parameters",
            Self::DesignDoctor => "Design Doctor",
            Self::Print => "Print",
            Self::Measure => "Measure",
            Self::IProperties => "iProperties",
            Self::Return => "Return",
            Self::ClearAppearance => "Clear",
        }
    }
    pub(super) fn default_shown(self) -> bool {
        !matches!(
            self,
            Self::Print
                | Self::Measure
                | Self::IProperties
                | Self::Return
                | Self::ClearAppearance
                | Self::RefineAppearance
                | Self::AppearanceOverride
                | Self::AdjustAppearance
                | Self::DesignDoctor
        )
    }
    pub(super) fn implemented(self) -> bool {
        !matches!(
            self,
            Self::RefineAppearance
                | Self::AppearanceOverride
                | Self::AdjustAppearance
                | Self::DesignDoctor
        )
    }
}
