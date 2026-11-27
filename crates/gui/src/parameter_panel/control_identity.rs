// SPDX-License-Identifier: GPL-2.0-only
use imgui::{IdStackToken, Ui};
pub(super) fn parameter_controls_identity(
    ui: &Ui,
    document: Option<u64>,
    parameter: Option<u64>,
) -> IdStackToken<'_> {
    ui.push_id(format!(
        "parameter-controls:document:{document:?}:parameter:{parameter:?}"
    ))
}
