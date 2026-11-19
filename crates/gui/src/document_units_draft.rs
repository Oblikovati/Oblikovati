// SPDX-License-Identifier: GPL-2.0-only
//! Stage document unit edits before a single owner transaction (#4096).
use oblikovati_api::wire::{DocumentUnitsInfo, SetDocumentUnitsArgs};

#[derive(Clone, Debug)]
pub(crate) struct UnitDraft {
    baseline: DocumentUnitsInfo,
    pub(crate) value: DocumentUnitsInfo,
    pending: bool,
    pub(crate) error: Option<String>,
}

impl UnitDraft {
    pub(crate) fn new(units: &DocumentUnitsInfo) -> Self {
        Self {
            baseline: units.clone(),
            value: units.clone(),
            pending: false,
            error: None,
        }
    }
    pub(crate) fn dirty(&self) -> bool {
        self.value != self.baseline
    }
    pub(crate) fn pending(&self) -> bool {
        self.pending
    }
    pub(crate) fn stale(&self, current: &DocumentUnitsInfo) -> bool {
        self.dirty() && *current != self.baseline
    }
    pub(crate) fn ready(&self, current: &DocumentUnitsInfo) -> bool {
        self.dirty()
            && !self.pending
            && *current == self.baseline
            && self.value.length_display_precision >= 0
            && self.value.angle_display_precision >= 0
    }
    pub(crate) fn sync(&mut self, current: &DocumentUnitsInfo) {
        if !self.dirty() && !self.pending && self.baseline != *current {
            self.baseline = current.clone();
            self.value = current.clone();
        }
    }
    pub(crate) fn submit(&mut self, current: &DocumentUnitsInfo) -> Option<SetDocumentUnitsArgs> {
        if !self.ready(current) {
            return None;
        }
        self.pending = true;
        self.error = None;
        Some(self.changed_fields())
    }
    pub(crate) fn acknowledge(&mut self, result: Result<(), String>) {
        if !self.pending {
            return;
        }
        self.pending = false;
        match result {
            Ok(()) => self.baseline = self.value.clone(),
            Err(error) => self.error = Some(error),
        }
    }
    fn changed_fields(&self) -> SetDocumentUnitsArgs {
        SetDocumentUnitsArgs {
            length_unit: changed(&self.baseline.length_unit, &self.value.length_unit),
            angle_unit: changed(&self.baseline.angle_unit, &self.value.angle_unit),
            mass_unit: changed(&self.baseline.mass_unit, &self.value.mass_unit),
            time_unit: changed(&self.baseline.time_unit, &self.value.time_unit),
            length_display_precision: changed(
                &self.baseline.length_display_precision,
                &self.value.length_display_precision,
            ),
            angle_display_precision: changed(
                &self.baseline.angle_display_precision,
                &self.value.angle_display_precision,
            ),
            length_display_format: changed(
                &self.baseline.length_display_format,
                &self.value.length_display_format,
            ),
        }
    }
}
fn changed<T: Clone + PartialEq>(before: &T, after: &T) -> Option<Box<T>> {
    (before != after).then(|| Box::new(after.clone()))
}

#[cfg(test)]
#[path = "document_units_draft/tests.rs"]
mod tests;
