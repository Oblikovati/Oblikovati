// SPDX-License-Identifier: GPL-2.0-only

// Fixed chrome and transient modal file pickers, wizards, confirmations, and help
// have a separate owner. Every entry is exact, so another window fails the guard.
pub(super) const INFRASTRUCTURE: &[(&str, &str)] = &[
    ("appearance_editor.rs", "modal:Save Edits"),
    ("appearance_editor/image_chooser.rs", "modal:TITLE"),
    (
        "application/document-creation/assembly_place_dialog.rs",
        "modal:FILE_CHOOSER",
    ),
    ("application/document-creation/workspaces.rs", "title"),
    (
        "application/document-tabs/mod.rs",
        "##document-list-control",
    ),
    ("application/document-tabs/previews.rs", "##tab-preview"),
    (
        "application/home-recent-documents/home_layout.rs",
        "##home-workspace",
    ),
    (
        "application/home-recent-documents/mod.rs",
        "##home-side-panel",
    ),
    (
        "application/quick-access-toolbar/mod.rs",
        "##quick-access-below",
    ),
    (
        "application/quick-access-toolbar/plane_selection.rs",
        "##plane-selection",
    ),
    (
        "application/ribbon-display/panel_selector_help.rs",
        "modal:HELP",
    ),
    (
        "application/viewport-orientation/axis_triad.rs",
        "##viewport-axis-triad",
    ),
    ("application/viewport-orientation/mod.rs", "CONTROL_WINDOW"),
    (
        "application/viewport-orientation/navigation_capsule.rs",
        "NAVIGATION_WINDOW",
    ),
    (
        "application/viewport-orientation/navigation_wheel.rs",
        "WINDOW",
    ),
    (
        "application/workspace-docking/mod.rs",
        "##workspace-outer-host",
    ),
    (
        "application/workspace-layout/browser_feature.rs",
        "SELECTION_TOOLBAR",
    ),
    ("application/workspace-layout/mod.rs", "##panel-plus"),
    ("application/workspace-layout/status_bar.rs", "##status-bar"),
    ("chrome/side.rs", "title"),
    ("chrome.rs", "##application-ribbon"),
    ("window_frame.rs", "##application-window-controls"),
    ("detached_drawing.rs", "##detached-drawing"),
    ("document_tabs/strip_render.rs", "##open-document-tabs"),
    // The extrusion handle follows its projected distance tip in the viewport.
    ("extrude_handle.rs", "WINDOW"),
    ("ribbon_panel_additional.rs", "<dynamic>"),
    ("ribbon_panel_additional/bend_notice.rs", "modal:TITLE"),
    (
        "ribbon_panel_additional/coordinate_restriction.rs",
        "modal:MESSAGE",
    ),
    ("ribbon_panels.rs", "##ribbon-temporary-panel"),
    ("sketch/command_alias.rs", "##command-alias"),
    ("sketch/line/mod.rs", "##sketch-hud"),
    ("dockable_window.rs", "title"),
    ("dockable_window.rs", "title"),
    ("ribbon_reset.rs", "modal:RESET_TITLE"),
    (
        "application/document-creation/chooser_modal.rs",
        "modal:TITLE",
    ),
    (
        "application/document-creation/component_dialog.rs",
        "modal:TITLE",
    ),
    (
        "application/document-creation/component_dialog.rs",
        "modal:FILE_CHOOSER",
    ),
    (
        "application/document-creation/component_dialog.rs",
        "modal:WARNING",
    ),
    (
        "application/document-creation/workspaces.rs",
        "modal:INSERT",
    ),
    (
        "application/project-dialog/project_picker.rs",
        "modal:TITLE",
    ),
    (
        "application/document-saving/close_prompt.rs",
        "modal:EXIT_ENVIRONMENT",
    ),
    (
        "application/document-saving/close_prompt.rs",
        "modal:SAVE_CHANGES",
    ),
    ("application/document-saving/save_as.rs", "modal:TITLE"),
    (
        "application/document-saving/save_as.rs",
        "modal:IMAGE_CHOOSER",
    ),
    (
        "application/document-saving/save_as.rs",
        "modal:CONFIRM_REPLACE",
    ),
    ("application/document-tabs/mod.rs", "modal:SELECT_WINDOW"),
    ("application/document-opening/mod.rs", "modal:TITLE"),
    ("application/document-opening/mod.rs", "modal:DEFER_TITLE"),
    (
        "application/home-recent-documents/mod.rs",
        "modal:NOTICE_TITLE",
    ),
    ("application/model-browser/color_dialog.rs", "modal:TITLE"),
    (
        "application/model-browser/find_prompts.rs",
        "modal:GROUP_TITLE",
    ),
    (
        "application/model-browser/find_prompts.rs",
        "modal:NO_MATCH_TITLE",
    ),
    (
        "application/model-browser/find_prompts.rs",
        "modal:SAVE_TITLE",
    ),
    (
        "application/model-browser/find_prompts.rs",
        "modal:OPEN_TITLE",
    ),
    (
        "application/project-dialog/new_project_wizard.rs",
        "modal:TITLE",
    ),
    (
        "application/project-dialog/new_project_wizard.rs",
        "modal:BROWSER_TITLE",
    ),
    (
        "application/ribbon-customization/mod.rs",
        "modal:Ribbon Settings Error",
    ),
    (
        "application/ribbon-customization/settings_dialog.rs",
        "modal:<unknown>",
    ),
    (
        "application/ribbon-customization/settings_dialog.rs",
        "modal:MISSING_FILE",
    ),
    ("sketch/dimension-edit/mod.rs", "modal:WARNING"),
    // The same dimension choice modal is also opened to close it when its draft ends.
    ("sketch/dimension-edit/mod.rs", "modal:TITLE"),
    ("sketch/dimension-edit/mod.rs", "modal:TITLE"),
    // Validation prompts are transient modals; their owning tools remain dockable.
    ("sketch/text_number.rs", "modal:spec.title"),
    ("browser_part_view_edit.rs", "modal:View Name Already Used"),
    ("model_state_table_names.rs", "modal:Model state names"),
    (
        "model_state_table_verification.rs",
        "modal:Table contains an error",
    ),
    // One transient confirmation; the Move tool itself remains dockable (#4054).
    ("sketch/move_tool.rs", "modal:REJECTION_POPUP"),
    // Cut rejection blocks acceptance while the owning Sweep tool remains dockable (#3917).
    ("sweep_output.rs", "modal:EDIT_WARNING"),
    // The failed edit and intermediate section prompts keep their draft in the dockable Loft.
    ("loft_edit_warning.rs", "modal:TITLE"),
    ("loft_section_dimensions.rs", "modal:TITLE"),
    // Emboss diagnostic, failed edit, and range warning prompts keep their draft in dockable Emboss.
    ("derive_dialog/picker.rs", "modal:PICKER_TITLE"),
    ("emboss_dialog/modals.rs", "modal:SKETCH_DOCTOR_TITLE"),
    ("emboss_dialog/modals.rs", "modal:EDIT_FAILED_TITLE"),
    ("emboss_dialog/modals.rs", "modal:RANGE_WARNING_TITLE"),
    ("rib_dialog/modals.rs", "modal:SKETCH_TIP_TITLE"),
    ("rib_dialog/modals.rs", "modal:CREATE_FAILED_TITLE"),
    ("rib_dialog/modals.rs", "modal:RANGE_WARNING_TITLE"),
];
