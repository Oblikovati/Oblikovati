// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    types::{
        ButtonStyle, COMPACT_ICON_BUTTON, DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART,
        DOCUMENT_PRESENTATION, LARGE_ICON_BUTTON, PART_RIBBON, RibbonKey, TEXT_ONLY_BUTTON,
        ZERO_DOC_RIBBON,
    },
    wire::{ListRibbonResult, RibbonControlInfo, RibbonPanelInfo, RibbonTabInfo},
};

mod ribbon_presentation;

/// Build the document ribbon from its selected key; example: `list(PART_RIBBON)`.
pub(crate) fn list(key: RibbonKey) -> ListRibbonResult {
    let tabs = match key.0.as_ref() {
        "ZeroDoc" => zero_document_tabs(),
        "Part" => part_tabs(),
        "Assembly" => assembly_tabs(),
        "Drawing" => drawing_tabs(),
        "Presentation" => ribbon_presentation::tabs(),
        "IFeatures" | "UnknownDocument" => Vec::new(),
        _ => Vec::new(),
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
        vec![
            panel(
                "Launch",
                vec![
                    large("GetStarted.NewPart", "New Part", "new-part"),
                    unavailable_large(
                        "GetStarted.NewSheetMetalPart",
                        "New Sheet Metal Part",
                        "sheet-metal-new",
                    ),
                    large("GetStarted.NewAssembly", "New Assembly", "new-assembly"),
                    large("GetStarted.NewDrawing", "New Drawing", "new-drawing"),
                ],
            ),
            panel(
                "Manage",
                vec![
                    unavailable_large(
                        "GetStarted.AddInCatalogue",
                        "AddIn Catalogue",
                        "addin-catalogue",
                    ),
                    large("GetStarted.Preferences", "Preferences", "preferences"),
                ],
            ),
        ],
    )]
}

fn assembly_tabs() -> Vec<RibbonTabInfo> {
    vec![
        tab(
            "Assemble",
            vec![
                panel(
                    "Component",
                    vec![unavailable_large("Assembly.Place", "Place", "place")],
                ),
                unavailable_compact_panel(
                    "Relationships",
                    &["Mate", "Flush", "Angle", "Tangent", "Insert"],
                ),
                unavailable_compact_panel(
                    "Joints",
                    &[
                        "Rigid",
                        "Rotational",
                        "Slider",
                        "Cylindrical",
                        "Planar",
                        "Ball",
                    ],
                ),
                panel(
                    "Representation",
                    vec![unavailable_large(
                        "Assembly.Representations",
                        "Representations",
                        "representation",
                    )],
                ),
                panel(
                    "Contact",
                    vec![unavailable_large(
                        "Assembly.ContactSet",
                        "Contact Set",
                        "contact-set",
                    )],
                ),
                panel(
                    "Sketch",
                    vec![unavailable_large(
                        "Assembly.CreateSketch",
                        "Create 2D Sketch",
                        "create-sketch",
                    )],
                ),
                unavailable_compact_panel(
                    "Modify",
                    &["Extrude", "Revolve", "Hole", "Chamfer", "Fillet"],
                ),
                unavailable_compact_panel(
                    "Pattern",
                    &["Rectangular Pattern", "Circular Pattern", "Mirror", "Copy"],
                ),
                panel(
                    "BOM",
                    vec![unavailable_large(
                        "Assembly.BOM",
                        "Bill of Materials",
                        "bom",
                    )],
                ),
            ],
        ),
        tab(
            "Manage",
            vec![panel(
                "Parameters",
                vec![unavailable_large(
                    "Manage.Parameters",
                    "Parameters",
                    "parameters",
                )],
            )],
        ),
        view_tab(),
    ]
}

fn drawing_tabs() -> Vec<RibbonTabInfo> {
    vec![tab(
        "Drawing",
        vec![
            unavailable_compact_panel("Sheets", &["New Sheet", "Delete Sheet"]),
            unavailable_compact_panel("Setup", &["Model Reference", "Drafting Standard"]),
            unavailable_compact_panel(
                "Views",
                &["Base View", "Projected View", "Section View", "Detail View"],
            ),
            unavailable_compact_panel(
                "Dimension",
                &["Linear Dimension", "Radial Dimension", "Angular Dimension"],
            ),
            unavailable_compact_panel("Annotate", &["Center Mark", "Centerline", "Note"]),
            unavailable_compact_panel("Table", &["Parts List", "Balloon", "Hole Table"]),
            unavailable_compact_panel(
                "Sketch",
                &["Sketch Rectangle", "Sketch Circle", "Hatch Region"],
            ),
            panel(
                "Output",
                vec![unavailable_large(
                    "Drawing.ExportDXF",
                    "Export DXF",
                    "drawing-export-dxf",
                )],
            ),
        ],
    )]
}

fn part_tabs() -> Vec<RibbonTabInfo> {
    vec![
        tab(
            "Create & Modify",
            vec![
                panel(
                    "Sketch",
                    vec![
                        large("Sketch.Create", "New 2D Sketch", "create-sketch"),
                        unavailable_large("Sketch.Create3D", "New 3D Sketch", "create-sketch-3d"),
                    ],
                ),
                panel(
                    "Create",
                    vec![
                        large("Feature.Extrude", "Extrude", "extrude"),
                        unavailable_large("Create.Revolve", "Revolve", "revolve"),
                        unavailable_large("Create.Sweep", "Sweep", "sweep"),
                        unavailable_large("Create.Loft", "Loft", "loft"),
                        unavailable_large("Create.Coil", "Coil", "coil"),
                    ],
                ),
                panel(
                    "Work Features",
                    vec![
                        unavailable_large("WorkPlane.Offset", "Work Plane", "work-plane-offset"),
                        unavailable_large("WorkAxis.OnEdge", "Work Axis", "work-axis"),
                        unavailable_large("WorkPoint.AtVertex", "Work Point", "work-point"),
                    ],
                ),
                unavailable_compact_panel(
                    "Modify",
                    &[
                        "Hole",
                        "Boss",
                        "Lip",
                        "Chamfer",
                        "Fillet",
                        "Shell",
                        "Draft",
                        "Thicken",
                        "Combine",
                        "Split",
                        "Move Face",
                        "Move Bodies",
                    ],
                ),
                unavailable_compact_panel(
                    "Pattern",
                    &["Rectangular", "Circular", "Mirror", "Sketch Driven"],
                ),
            ],
        ),
        tab(
            "Surfaces & Mesh",
            vec![
                panel(
                    "Sketch",
                    vec![
                        large("Sketch.Create", "New 2D Sketch", "create-sketch"),
                        unavailable_large("Sketch.Create3D", "New 3D Sketch", "create-sketch-3d"),
                    ],
                ),
                unavailable_compact_panel(
                    "Surface",
                    &[
                        "Patch", "Trim", "Extend", "Stitch", "Offset", "Thicken", "Ruled", "Loft",
                    ],
                ),
                unavailable_compact_panel("Freeform", &["Create Form", "Edit Form", "Finish Form"]),
                unavailable_compact_panel("Mesh", &["Import Mesh", "Repair", "Convert"]),
            ],
        ),
        tab(
            "Manage",
            vec![panel("Parameters", vec![text("File.Save", "Save")])],
        ),
        tab("Sheet Metal", vec![panel("Create", vec![])]),
        view_tab(),
        tab("Inspect", vec![panel("Measure", vec![])]),
    ]
}

/// Return the contextual tab shown only while a 2D sketch is edited.
/// Example: `contextual_sketch_tab()` appends the Go Sketch tab to a Part ribbon.
pub fn contextual_sketch_tab() -> RibbonTabInfo {
    tab(
        "Sketch",
        vec![
            panel(
                "Create",
                vec![
                    large("Sketch.Rectangle", "Rectangle", "rectangle"),
                    large("Sketch.Circle", "Circle", "circle"),
                ],
            ),
            panel(
                "Exit",
                vec![large("Sketch.Finish", "Finish Sketch", "finish-sketch")],
            ),
        ],
    )
}

fn view_tab() -> RibbonTabInfo {
    tab(
        "View",
        vec![panel(
            "Navigate",
            vec![
                unavailable_large("View.Fit", "Zoom All", "zoom-all"),
                compact("View.Home", "Home", "home"),
                unavailable_large("View.LookAt", "Look At", "look-at"),
                unavailable_large("View.ZoomWindow", "Zoom Window", "zoom-window"),
                unavailable_large(
                    "View.ConstrainedOrbit",
                    "Constrained Orbit",
                    "orbit-constrained",
                ),
                compact("View.Previous", "Previous View", "home"),
            ],
        )],
    )
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

fn unavailable_large(command_id: &str, display_name: &str, icon: &str) -> RibbonControlInfo {
    unavailable(command_id, display_name, icon, LARGE_ICON_BUTTON)
}

fn unavailable_compact_panel(name: &str, labels: &[&str]) -> RibbonPanelInfo {
    panel(
        name,
        labels
            .iter()
            .map(|label| unavailable("", label, "", COMPACT_ICON_BUTTON))
            .collect(),
    )
}

fn unavailable(
    command_id: &str,
    display_name: &str,
    icon: &str,
    button_style: ButtonStyle,
) -> RibbonControlInfo {
    let mut control = control(command_id, display_name, icon, button_style);
    control.enabled = false;
    control
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

        for key in [
            ZERO_DOC_RIBBON,
            PART_RIBBON,
            RibbonKey("Assembly".into()),
            RibbonKey("Drawing".into()),
            RibbonKey("Presentation".into()),
        ] {
            assert!(
                !list(key).tabs.unwrap().is_empty(),
                "supported document ribbon must have a visible tab"
            );
        }
    }

    #[test]
    fn part_ribbon_keeps_go_tab_names_and_panel_boundaries() {
        let tabs = list(PART_RIBBON).tabs.unwrap();
        assert_eq!(tabs[0].name, "Create & Modify");
        assert_eq!(tabs[1].name, "Surfaces & Mesh");
        assert_eq!(
            tabs[0]
                .panels
                .as_ref()
                .unwrap()
                .iter()
                .map(|panel| panel.name.as_str())
                .collect::<Vec<_>>(),
            ["Sketch", "Create", "Work Features", "Modify", "Pattern"]
        );
        let sketch = contextual_sketch_tab();
        assert_eq!(sketch.name, "Sketch");
        assert_eq!(sketch.panels.as_ref().unwrap()[0].name, "Create");
        assert_eq!(sketch.panels.as_ref().unwrap()[1].name, "Exit");
    }

    #[test]
    fn sketch_context_exposes_rectangle_and_circle_creation() {
        let create = &contextual_sketch_tab().panels.unwrap()[0];
        let controls = create.controls.as_ref().unwrap();
        assert_eq!(create.name, "Create");
        assert_eq!(controls.len(), 2);
        assert_eq!(controls[0].command_id, "Sketch.Rectangle");
        assert_eq!(controls[1].command_id, "Sketch.Circle");
        assert!(controls.iter().all(|control| control.enabled));
    }

    #[test]
    fn document_ribbons_keep_the_go_document_specific_tabs_and_panels() {
        let zero_doc = list(ZERO_DOC_RIBBON).tabs.unwrap();
        assert_eq!(zero_doc[0].name, "Get Started");
        assert_eq!(zero_doc[0].panels.as_ref().unwrap()[0].name, "Launch");
        assert!(
            zero_doc[0].panels.as_ref().unwrap()[0]
                .controls
                .as_ref()
                .unwrap()[2]
                .enabled
        );
        assert!(
            zero_doc[0].panels.as_ref().unwrap()[0]
                .controls
                .as_ref()
                .unwrap()[3]
                .enabled
        );
        assert!(
            zero_doc[0].panels.as_ref().unwrap()[1]
                .controls
                .as_ref()
                .unwrap()[1]
                .enabled
        );

        let assembly = list(RibbonKey("Assembly".into())).tabs.unwrap();
        assert_eq!(assembly[0].name, "Assemble");
        assert_eq!(assembly[0].panels.as_ref().unwrap()[0].name, "Component");

        let drawing = list(RibbonKey("Drawing".into())).tabs.unwrap();
        assert_eq!(drawing[0].name, "Drawing");
        assert_eq!(drawing[0].panels.as_ref().unwrap()[0].name, "Sheets");

        let presentation = list(RibbonKey("Presentation".into())).tabs.unwrap();
        assert_eq!(presentation[0].name, "Presentation");
        assert_eq!(presentation[0].panels.as_ref().unwrap()[0].name, "Create");
        assert_eq!(presentation[1].name, "View");
    }
}
