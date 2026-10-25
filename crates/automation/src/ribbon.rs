// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    types::{
        ButtonStyle, COMPACT_ICON_BUTTON, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART,
        DOCUMENT_PRESENTATION, LARGE_ICON_BUTTON, PART_RIBBON, RibbonKey, TEXT_ONLY_BUTTON,
        ZERO_DOC_RIBBON,
    },
    wire::{ListRibbonResult, RibbonControlInfo, RibbonPanelInfo, RibbonTabInfo},
};

/// Build the document ribbon from its selected key; example: `list(PART_RIBBON)`.
pub(crate) fn list(key: RibbonKey) -> ListRibbonResult {
    let tabs = match key.0.as_ref() {
        "ZeroDoc" => zero_document_tabs(),
        "Part" => part_tabs(),
        "Assembly" => common_document_tabs("Assembly"),
        "Drawing" => common_document_tabs("Drawing"),
        "Presentation" => common_document_tabs("Presentation"),
        "IFeatures" => common_document_tabs("iFeatures"),
        _ => common_document_tabs("Tools"),
    };
    ListRibbonResult {
        key,
        tabs: Some(tabs),
    }
}

/// Select the API ribbon key for an active file type; example: `key_for_document(Some(DOCUMENT_PART))`.
pub(crate) fn key_for_document(document: Option<oblikovati_api::types::DocumentType>) -> RibbonKey {
    match document {
        None => ZERO_DOC_RIBBON,
        Some(kind) if kind == DOCUMENT_PART => PART_RIBBON,
        Some(kind) if kind == DOCUMENT_ASSEMBLY => RibbonKey("Assembly".into()),
        Some(kind) if kind == DOCUMENT_DRAWING => RibbonKey("Drawing".into()),
        Some(kind) if kind == DOCUMENT_PRESENTATION => RibbonKey("Presentation".into()),
        Some(_) => RibbonKey("UnknownDocument".into()),
    }
}

fn zero_document_tabs() -> Vec<RibbonTabInfo> {
    vec![tab(
        "Get Started",
        vec![panel(
            "Create",
            vec![large("GetStarted.NewPart", "New Part", "new-part")],
        )],
    )]
}

fn part_tabs() -> Vec<RibbonTabInfo> {
    vec![
        tab(
            "Create & Modify",
            vec![
                panel(
                    "Create",
                    vec![large("Sketch.Create", "Start Sketch", "create-sketch")],
                ),
                panel(
                    "Solid",
                    vec![large("Feature.Extrude", "Extrude", "extrude")],
                ),
            ],
        ),
        tab("Surfaces & Mesh", vec![panel("Surface", vec![])]),
        tab(
            "Manage",
            vec![panel("Parameters", vec![text("File.Save", "Save")])],
        ),
        tab("Sheet Metal", vec![panel("Create", vec![])]),
        tab(
            "View",
            vec![panel(
                "Navigate",
                vec![
                    compact("View.Home", "Home", "home"),
                    compact("View.Previous", "Previous View", "home"),
                ],
            )],
        ),
        tab("Inspect", vec![panel("Measure", vec![])]),
        tab(
            "Sketch",
            vec![panel(
                "Draw",
                vec![
                    large("Sketch.Rectangle", "Rectangle", "rectangle"),
                    text("Sketch.Finish", "Finish Sketch"),
                ],
            )],
        ),
    ]
}

fn common_document_tabs(primary: &str) -> Vec<RibbonTabInfo> {
    vec![
        tab(primary, vec![panel("Create", vec![])]),
        tab(
            "Manage",
            vec![panel("File", vec![text("File.Save", "Save")])],
        ),
        tab(
            "View",
            vec![panel(
                "Navigate",
                vec![
                    compact("View.Home", "Home", "home"),
                    compact("View.Previous", "Previous View", "home"),
                ],
            )],
        ),
    ]
}

fn tab(name: &str, panels: Vec<RibbonPanelInfo>) -> RibbonTabInfo {
    RibbonTabInfo {
        name: name.into(),
        panels: Some(panels),
    }
}

fn panel(name: &str, controls: Vec<RibbonControlInfo>) -> RibbonPanelInfo {
    RibbonPanelInfo {
        name: name.into(),
        controls: Some(controls),
        ..Default::default()
    }
}

fn large(command_id: &str, display_name: &str, icon: &str) -> RibbonControlInfo {
    control(command_id, display_name, icon, LARGE_ICON_BUTTON)
}

fn compact(command_id: &str, display_name: &str, icon: &str) -> RibbonControlInfo {
    control(command_id, display_name, icon, COMPACT_ICON_BUTTON)
}

fn text(command_id: &str, display_name: &str) -> RibbonControlInfo {
    control(command_id, display_name, "", TEXT_ONLY_BUTTON)
}

fn control(
    command_id: &str,
    display_name: &str,
    icon: &str,
    button_style: ButtonStyle,
) -> RibbonControlInfo {
    RibbonControlInfo {
        command_id: command_id.into(),
        display_name: display_name.into(),
        button_style,
        icon: icon.into(),
        enabled: true,
        ..Default::default()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn every_document_type_selects_its_own_ribbon() {
        assert_eq!(key_for_document(None), ZERO_DOC_RIBBON);
        assert_eq!(key_for_document(Some(DOCUMENT_PART)), PART_RIBBON);
        assert_eq!(key_for_document(Some(DOCUMENT_ASSEMBLY)).0, "Assembly");
        assert_eq!(key_for_document(Some(DOCUMENT_DRAWING)).0, "Drawing");
        assert_eq!(
            key_for_document(Some(DOCUMENT_PRESENTATION)).0,
            "Presentation"
        );
    }

    #[test]
    fn part_ribbon_keeps_go_tab_names_and_panel_boundaries() {
        let tabs = list(PART_RIBBON).tabs.unwrap();
        assert_eq!(tabs[0].name, "Create & Modify");
        assert_eq!(tabs[1].name, "Surfaces & Mesh");
        assert_eq!(tabs[6].name, "Sketch");
        assert_eq!(tabs[0].panels.as_ref().unwrap().len(), 2);
    }
}
