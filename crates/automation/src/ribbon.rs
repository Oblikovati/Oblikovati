// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{ListRibbonResult, RibbonControlInfo, RibbonPanelInfo, RibbonTabInfo};

/// Return the native desktop ribbon available to API and GUI clients.
pub(crate) fn list() -> ListRibbonResult {
    ListRibbonResult {
        tabs: Some(vec![
            tab(
                "Part",
                "File",
                vec![
                    control("GetStarted.NewPart", "New Part", "new-part"),
                    control("File.Save", "Save", "save"),
                ],
            ),
            tab(
                "Sketch",
                "Create",
                vec![
                    control("Sketch.Create", "Start Sketch on XY", "create-sketch"),
                    control("Sketch.Rectangle", "Rectangle", "rectangle"),
                    control("Sketch.Finish", "Exit Sketch", "finish-sketch"),
                ],
            ),
            tab(
                "3D Model",
                "Create",
                vec![control("Feature.Extrude", "Extrude", "extrude")],
            ),
            tab(
                "View",
                "Navigate",
                vec![
                    control("View.Home", "Home", "home"),
                    control("View.Previous", "Previous View", "home"),
                ],
            ),
        ]),
        ..Default::default()
    }
}

fn tab(name: &str, panel: &str, controls: Vec<RibbonControlInfo>) -> RibbonTabInfo {
    RibbonTabInfo {
        name: name.into(),
        panels: Some(vec![RibbonPanelInfo {
            name: panel.into(),
            controls: Some(controls),
            ..Default::default()
        }]),
    }
}

fn control(command_id: &str, display_name: &str, icon: &str) -> RibbonControlInfo {
    RibbonControlInfo {
        command_id: command_id.into(),
        display_name: display_name.into(),
        icon: icon.into(),
        enabled: true,
        ..Default::default()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn native_ribbon_declares_each_cube_authoring_environment() {
        let tabs = list().tabs.unwrap();
        assert_eq!(
            tabs.iter().map(|tab| tab.name.as_str()).collect::<Vec<_>>(),
            ["Part", "Sketch", "3D Model", "View"]
        );
        assert!(
            tabs.iter()
                .all(|tab| tab.panels.as_ref().is_some_and(|panels| !panels.is_empty()))
        );
    }

    #[test]
    fn view_ribbon_declares_each_available_camera_command() {
        let tabs = list().tabs.unwrap();
        let view = tabs.iter().find(|tab| tab.name == "View").unwrap();
        let controls = view.panels.as_ref().unwrap()[0].controls.as_ref().unwrap();
        assert_eq!(
            controls
                .iter()
                .map(|control| control.command_id.as_str())
                .collect::<Vec<_>>(),
            ["View.Home", "View.Previous"]
        );
    }
}
