// SPDX-License-Identifier: GPL-2.0-only
//! Session display categories in the View ribbon.
use crate::icons::{GuiRibbonIcons, IconGlyph};
use imgui::Ui;

const MENU: &str = "##object-visibility";
const SKETCHES_2D: usize = 7;
const SKETCH_DIMENSIONS: usize = 9;
const UCS_START: usize = 11;
// The second column identifies the categories controlled by All Workfeatures.
const CATEGORIES: [(&str, bool); 15] = [
    ("Origin Planes", true),
    ("Origin Axes", true),
    ("Origin Points", true),
    ("User Work Planes", true),
    ("User Work Axes", true),
    ("User Work Points", true),
    ("Construction Surfaces", false),
    ("2D Sketches", false),
    ("3D Sketches", false),
    ("Sketch Dimensions", false),
    ("3D Annotations", false),
    ("UCS Triad", true),
    ("UCS Planes", true),
    ("UCS Axes", true),
    ("UCS Points", true),
];

#[derive(Debug)]
pub(crate) struct ObjectVisibility {
    workfeatures: bool,
    categories: [bool; CATEGORIES.len()],
}

impl Default for ObjectVisibility {
    fn default() -> Self {
        Self {
            workfeatures: true,
            categories: [true; CATEGORIES.len()],
        }
    }
}

impl ObjectVisibility {
    pub(crate) fn ribbon(
        &mut self,
        ui: &Ui,
        icons: &GuiRibbonIcons,
        size: [f32; 2],
        actions: &mut Vec<crate::GuiShellAction>,
    ) -> bool {
        if icons.button(ui, "Object Visibility", IconGlyph::SketchVisibility, size) {
            ui.open_popup(MENU);
            crate::widget_abi::next_window_position(
                ui,
                [ui.item_rect_min()[0], ui.item_rect_max()[1]],
            );
        }
        let Some(_popup) = ui.begin_popup(MENU) else {
            return false;
        };
        let before = self.categories[SKETCHES_2D];
        let changed = self.checklist(ui);
        let visible = self.categories[SKETCHES_2D];
        if before != visible {
            actions.push(crate::GuiShellAction::SetSketchOverlays(visible));
        }
        changed
    }

    fn checklist(&mut self, ui: &Ui) -> bool {
        let mut changed = ui.checkbox("All Workfeatures", &mut self.workfeatures);
        if changed {
            for ((_, follows_master), shown) in CATEGORIES.iter().zip(&mut self.categories) {
                if *follows_master {
                    *shown = self.workfeatures;
                }
            }
        }
        for (index, ((name, _), shown)) in CATEGORIES.iter().zip(&mut self.categories).enumerate() {
            if index == UCS_START {
                ui.separator();
            }
            changed |= ui.checkbox(name, shown);
        }
        changed
    }

    pub(super) fn shows_origin_row(&self, row: usize) -> bool {
        match row {
            0..=2 => self.categories[0],
            3..=5 => self.categories[1],
            6 => self.categories[2],
            _ => false,
        }
    }

    pub(crate) fn dimension_positions(
        &self,
        ui: &Ui,
        request: &crate::GuiFrameRequest,
        area: super::WorkspaceArea,
    ) -> Vec<Option<crate::dimension_labels::LabelPosition>> {
        // The active sketch's dimensions remain available to its editing tools.
        if self.categories[SKETCH_DIMENSIONS]
            || request.commands.authoring == crate::GuiAuthoringContext::Sketch
        {
            crate::dimension_labels::dimension_label_layout(ui, request, area)
        } else {
            Vec::new()
        }
    }
}
