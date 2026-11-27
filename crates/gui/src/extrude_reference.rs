// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiExtrudeReferenceExtent;
use imgui::{StyleColor, Ui};

#[derive(Clone, Copy, Default, PartialEq, Eq)]
enum ExtentMode {
    #[default]
    Distance,
    DistanceFromPlane,
    ThroughAll,
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
        matches!(
            self.mode,
            ExtentMode::Distance | ExtentMode::DistanceFromPlane
        )
    }

    pub(super) fn needs_start_plane(&self) -> bool {
        self.mode == ExtentMode::DistanceFromPlane
    }

    pub(super) fn through_all(&self) -> bool {
        self.mode == ExtentMode::ThroughAll
    }

    pub(super) fn request(&self) -> Option<GuiExtrudeReferenceExtent> {
        match self.mode {
            ExtentMode::Distance => None,
            ExtentMode::DistanceFromPlane => Some(GuiExtrudeReferenceExtent::DistanceFromPlane {
                from: self.from.clone()?,
            }),
            ExtentMode::ThroughAll => Some(GuiExtrudeReferenceExtent::ThroughAll),
            ExtentMode::ToFace => Some(GuiExtrudeReferenceExtent::ToFace {
                to: self.to.clone()?,
            }),
            ExtentMode::FromTo => Some(GuiExtrudeReferenceExtent::FromTo {
                from: self.from.clone()?,
                to: self.to.clone()?,
            }),
        }
    }

    pub(super) fn clear_picks(&mut self) {
        self.from = None;
        self.to = None;
        self.active = None;
        self.changed_by_pick = false;
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

    pub(super) fn draw(&mut self, ui: &Ui, through_all_available: bool) -> bool {
        let before = self.mode;
        self.draw_extent_modes(ui, through_all_available);
        if matches!(
            self.mode,
            ExtentMode::DistanceFromPlane | ExtentMode::FromTo
        ) {
            self.draw_target(ui, Slot::From);
        }
        if matches!(self.mode, ExtentMode::ToFace | ExtentMode::FromTo) {
            self.draw_target(ui, Slot::To);
        }
        std::mem::take(&mut self.changed_by_pick) || self.mode != before
    }

    fn draw_extent_modes(&mut self, ui: &Ui, through_all_available: bool) {
        crate::theme::property_label(ui, "From");
        let from = match self.mode {
            ExtentMode::DistanceFromPlane => 1,
            ExtentMode::FromTo => 2,
            _ => 0,
        };
        match extent_combo(
            ui,
            "##extrude-start",
            from,
            ["Sketch Plane", "Reference Plane", "From / To"],
            [true; 3],
        ) {
            Some(0) => self.set_mode(ExtentMode::Distance),
            Some(1) => self.set_mode(ExtentMode::DistanceFromPlane),
            Some(2) => self.set_mode(ExtentMode::FromTo),
            _ => {}
        }

        crate::theme::property_label(ui, "Termination");
        let to = match self.mode {
            ExtentMode::Distance | ExtentMode::DistanceFromPlane => 0,
            ExtentMode::ThroughAll => 2,
            _ => 1,
        };
        match extent_combo(
            ui,
            "##extrude-termination",
            to,
            ["Distance", "To Face", "Through All"],
            [true, true, through_all_available],
        ) {
            Some(0) if self.mode == ExtentMode::FromTo => {
                self.set_mode(ExtentMode::DistanceFromPlane)
            }
            Some(0) => self.set_mode(ExtentMode::Distance),
            Some(1) if self.mode == ExtentMode::DistanceFromPlane => {
                self.set_mode(ExtentMode::FromTo)
            }
            Some(1) => self.set_mode(ExtentMode::ToFace),
            Some(2) => self.set_mode(ExtentMode::ThroughAll),
            _ => {}
        }
    }

    fn set_mode(&mut self, mode: ExtentMode) {
        self.mode = mode;
        self.active = None;
    }

    fn draw_target(&mut self, ui: &Ui, slot: Slot) {
        let (label, face, id) = match slot {
            Slot::From => ("From", &self.from, "extrude-from-face"),
            Slot::To => ("To", &self.to, "extrude-to-face"),
        };
        crate::theme::property_label(ui, label);
        let caption = match face.as_deref() {
            Some(reference)
                if reference.starts_with("origin/plane/") || reference.starts_with("datum/v1/") =>
            {
                "Work Plane"
            }
            Some(_) => "Face",
            None => "Select Face",
        };
        let color = ui.style_color(StyleColor::ButtonActive);
        let _active =
            (self.active == Some(slot)).then(|| ui.push_style_color(StyleColor::Button, color));
        if ui.button(format!("{caption}##{id}")) {
            self.active = Some(slot);
        }
    }
}

impl super::ExtrudeDialog {
    pub(crate) fn awaits_reference_pick(&self) -> bool {
        self.document.is_some() && self.reference.awaits_pick()
    }
    pub(crate) fn pick_reference(&mut self, document: u64, reference: String) {
        if self.document == Some(document) {
            self.reference.pick(reference);
        }
    }
}

// Selecting the already active row only dismisses the popup.
fn extent_combo<const N: usize>(
    ui: &Ui,
    id: &str,
    selected: usize,
    labels: [&str; N],
    enabled: [bool; N],
) -> Option<usize> {
    let _combo = ui.begin_combo(id, labels[selected])?;
    labels.into_iter().enumerate().find_map(|(index, label)| {
        (ui.selectable_config(label)
            .selected(index == selected)
            .disabled(!enabled[index])
            .build()
            && index != selected)
            .then_some(index)
    })
}
