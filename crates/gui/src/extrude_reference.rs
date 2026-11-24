// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiExtrudeReferenceExtent;
use imgui::{StyleColor, Ui};

#[derive(Clone, Copy, Default, PartialEq, Eq)]
enum ExtentMode {
    #[default]
    Distance,
    ToFace,
    FromTo,
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Slot {
    From,
    To,
}

#[derive(Default)]
pub(super) struct ReferenceDraft {
    mode: ExtentMode,
    from: Option<String>,
    to: Option<String>,
    active: Option<Slot>,
    changed_by_pick: bool,
}

impl ReferenceDraft {
    pub(super) fn numeric(&self) -> bool {
        self.mode == ExtentMode::Distance
    }

    pub(super) fn request(&self) -> Option<GuiExtrudeReferenceExtent> {
        let to = self.to.clone()?;
        match self.mode {
            ExtentMode::Distance => None,
            ExtentMode::ToFace => Some(GuiExtrudeReferenceExtent::ToFace { to }),
            ExtentMode::FromTo => Some(GuiExtrudeReferenceExtent::FromTo {
                from: self.from.clone()?,
                to,
            }),
        }
    }

    pub(super) fn awaits_pick(&self) -> bool {
        self.active.is_some()
    }

    pub(super) fn pick(&mut self, face: String) {
        let Some(slot) = self.active.take() else {
            return;
        };
        let target = match slot {
            Slot::From => &mut self.from,
            Slot::To => &mut self.to,
        };
        self.changed_by_pick |= target.as_ref() != Some(&face);
        *target = Some(face);
    }

    pub(super) fn draw(&mut self, ui: &Ui) -> bool {
        let before = self.mode;
        self.draw_extent_modes(ui);
        if self.mode == ExtentMode::FromTo {
            self.draw_target(ui, Slot::From);
        }
        if !self.numeric() {
            self.draw_target(ui, Slot::To);
        }
        std::mem::take(&mut self.changed_by_pick) || self.mode != before
    }

    fn draw_extent_modes(&mut self, ui: &Ui) {
        ui.text("From");
        ui.same_line();
        let from = if self.mode == ExtentMode::FromTo {
            "From / To"
        } else {
            "Sketch Plane"
        };
        if ui.button(from) {
            self.toggle(ExtentMode::FromTo);
        }
        ui.same_line();
        ui.text("Termination");
        ui.same_line();
        let to = if self.numeric() {
            "Distance"
        } else {
            "To Face"
        };
        if ui.button(to) {
            self.toggle(ExtentMode::ToFace);
        }
    }

    fn toggle(&mut self, mode: ExtentMode) {
        self.mode = if self.mode == mode {
            ExtentMode::Distance
        } else {
            mode
        };
        self.active = None;
    }

    fn draw_target(&mut self, ui: &Ui, slot: Slot) {
        let (label, face, id) = match slot {
            Slot::From => ("From", &self.from, "extrude-from-face"),
            Slot::To => ("To", &self.to, "extrude-to-face"),
        };
        ui.text(label);
        ui.same_line();
        let caption = if face.is_some() {
            "Face"
        } else {
            "Select Face"
        };
        let color = ui.style_color(StyleColor::ButtonActive);
        let _active =
            (self.active == Some(slot)).then(|| ui.push_style_color(StyleColor::Button, color));
        if ui.button(format!("{caption}##{id}")) {
            self.active = Some(slot);
        }
    }
}
