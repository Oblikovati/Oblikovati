// SPDX-License-Identifier: GPL-2.0-only
//! Session-scoped inspection state and the settings draft.
use super::*;
use crate::HeadSketchEdit;
use oblikovati_gui::{GuiCurvatureAction, GuiCurvatureSettings, GuiCurvatureSliders};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use oblikovati_viewport::ViewportSketchCurvature;
use std::collections::BTreeMap;

const START: GuiCurvatureSliders = GuiCurvatureSliders {
    density: 25,
    scale: 35,
};

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
struct CurveKey(u64, u64, SketchEntityId);

impl CurveKey {
    fn new(edit: HeadSketchEdit, entity: SketchEntityId) -> Self {
        Self(edit.document, edit.sketch.0, entity)
    }

    fn belongs_to(self, edit: HeadSketchEdit) -> bool {
        self.0 == edit.document && self.1 == edit.sketch.0
    }
}

struct CurveDisplay {
    shown: bool,
    sliders: GuiCurvatureSliders,
}

struct SettingsDraft {
    key: CurveKey,
    generation: u64,
    view: GuiCurvatureSettings,
}

#[derive(Default)]
pub(super) struct CurvaturePresentation {
    curves: BTreeMap<CurveKey, CurveDisplay>,
    draft: Option<SettingsDraft>,
    serial: u64,
}

impl CurvaturePresentation {
    pub(super) fn discard_draft(&mut self) {
        self.draft = None;
    }

    pub(super) fn forget_document(&mut self, document: u64) {
        self.curves.retain(|key, _| key.0 != document);
        if self
            .draft
            .as_ref()
            .is_some_and(|draft| draft.key.0 == document)
        {
            self.draft = None;
        }
    }

    fn shown(&self, key: CurveKey) -> bool {
        self.curves.get(&key).is_some_and(|display| display.shown)
    }

    fn toggle(&mut self, key: CurveKey) {
        let display = self.curves.entry(key).or_insert(CurveDisplay {
            shown: false,
            sliders: START,
        });
        display.shown = !display.shown;
        if !display.shown && self.draft.as_ref().is_some_and(|draft| draft.key == key) {
            self.draft = None;
        }
    }

    fn setup(&mut self, key: CurveKey, generation: u64) -> Result<(), String> {
        let sliders = self
            .curves
            .get(&key)
            .filter(|display| display.shown)
            .ok_or_else(|| format!("curvature settings {key:?}: expected enabled curvature"))?
            .sliders;
        self.serial = self
            .serial
            .checked_add(1)
            .ok_or("curvature dialog sequence: expected an unused token")?;
        let view = GuiCurvatureSettings {
            token: self.serial,
            applied: sliders,
            preview: sliders,
        };
        self.draft = Some(SettingsDraft {
            key,
            generation,
            view,
        });
        Ok(())
    }

    fn view(&self, edit: HeadSketchEdit, generation: u64) -> Option<&SettingsDraft> {
        self.draft
            .as_ref()
            .filter(|draft| draft.key.belongs_to(edit) && draft.generation == generation)
    }

    fn preview(&mut self, sliders: GuiCurvatureSliders) -> Result<(), String> {
        if sliders.density > 100 || sliders.scale > 100 {
            return Err(format!(
                "curvature sliders {sliders:?}: expected values in 0..=100"
            ));
        }
        if let Some(draft) = &mut self.draft {
            draft.view.preview = sliders;
        }
        Ok(())
    }

    fn accept(&mut self, close: bool) {
        let Some(draft) = &mut self.draft else {
            return;
        };
        if let Some(display) = self.curves.get_mut(&draft.key) {
            display.sliders = draft.view.preview;
            draft.view.applied = draft.view.preview;
        }
        if close {
            self.draft = None;
        }
    }

    pub(super) fn overlays(
        &self,
        edit: HeadSketchEdit,
        generation: u64,
        geometry: &Sketch2D,
    ) -> Vec<ViewportSketchCurvature> {
        let draft = self.view(edit, generation);
        let first = CurveKey::new(edit, SketchEntityId(u64::MIN));
        let last = CurveKey::new(edit, SketchEntityId(u64::MAX));
        self.curves
            .range(first..=last)
            .filter(|(key, display)| display.shown && geometry.ellipse(key.2).is_ok())
            .map(|(key, display)| {
                let sliders = draft
                    .filter(|draft| draft.key == *key)
                    .map_or(display.sliders, |draft| draft.view.preview);
                overlay(key.2, sliders)
            })
            .collect()
    }
}

fn overlay(entity: SketchEntityId, sliders: GuiCurvatureSliders) -> ViewportSketchCurvature {
    // The source defines relative slider travel. The range keeps the overlay legible and bounded.
    ViewportSketchCurvature {
        entity,
        intervals: 12 + 4 * ((u16::from(sliders.density) * 21) / 100),
        envelope_ratio: 0.05 + f64::from(sliders.scale) * 0.0025,
    }
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn curvature_context_shown(
        &self,
        edit: HeadSketchEdit,
        geometry: &Sketch2D,
        curve: Option<SketchEntityId>,
    ) -> Option<bool> {
        let curve = curve.filter(|curve| geometry.ellipse(*curve).is_ok())?;
        Some(
            self.sketch_memory
                .curvature
                .shown(CurveKey::new(edit, curve)),
        )
    }

    pub(super) fn gui_curvature_settings(&self) -> Option<GuiCurvatureSettings> {
        let edit = self.active_sketch_edit()?;
        let session = self.files.workspace().session(edit.document).ok()?;
        self.sketch_memory
            .curvature
            .view(edit, session.generation())
            .map(|draft| draft.view)
    }

    pub(super) fn curvature_scene(&mut self) -> crate::sketch_scene::ActiveCurvature {
        if self.sketch_memory.curvature.draft.is_some() && self.gui_curvature_settings().is_none() {
            self.sketch_memory.curvature.discard_draft();
        }
        let state = &self.sketch_memory.curvature;
        if state.curves.is_empty() {
            return None;
        }
        let edit = self.active_sketch_edit()?;
        let session = self.files.workspace().session(edit.document).ok()?;
        let sketch = session.snapshot().sketches.get(edit.sketch).ok()?;
        let overlays = state.overlays(edit, session.generation(), sketch.geometry());
        (!overlays.is_empty()).then_some((edit.sketch, overlays))
    }

    pub(super) fn sketch_curvature_command(
        &mut self,
        action: GuiCurvatureAction,
    ) -> Result<bool, String> {
        match action {
            GuiCurvatureAction::Toggle { context } | GuiCurvatureAction::Setup { context } => self
                .curvature_context_command(
                    context,
                    matches!(action, GuiCurvatureAction::Setup { .. }),
                ),
            GuiCurvatureAction::Preview { dialog, sliders } => {
                self.curvature_draft_command(dialog, |state| state.preview(sliders))
            }
            GuiCurvatureAction::Apply { dialog, close } => {
                self.curvature_draft_command(dialog, |state| {
                    state.accept(close);
                    Ok(())
                })
            }
            GuiCurvatureAction::Cancel { dialog } => {
                self.curvature_draft_command(dialog, |state| {
                    state.draft = None;
                    Ok(())
                })
            }
        }
    }

    fn curvature_context_command(&mut self, context: u64, setup: bool) -> Result<bool, String> {
        let Some(target) = self
            .sketch_memory
            .context
            .as_ref()
            .filter(|target| target.token == context)
        else {
            return Ok(false);
        };
        if self.sketch_tool.is_some()
            || !self.sketch_context_valid(target.document, target.sketch, target.generation)
        {
            return Ok(false);
        }
        let Some(entity) = target.curve else {
            return Ok(false);
        };
        let session = self
            .files
            .workspace()
            .session(target.document)
            .map_err(|error| error.to_string())?;
        let sketch = session
            .snapshot()
            .sketches
            .get(target.sketch)
            .map_err(|error| error.to_string())?;
        if sketch.geometry().ellipse(entity).is_err() {
            return Ok(false);
        }
        let key = CurveKey(target.document, target.sketch.0, entity);
        if setup {
            self.sketch_memory.curvature.setup(key, target.generation)?;
        } else {
            self.sketch_memory.curvature.toggle(key);
        }
        self.sketch_memory.context = None;
        Ok(true)
    }

    fn curvature_draft_command(
        &mut self,
        dialog: u64,
        update: impl FnOnce(&mut CurvaturePresentation) -> Result<(), String>,
    ) -> Result<bool, String> {
        if self
            .gui_curvature_settings()
            .is_none_or(|view| view.token != dialog)
        {
            return Ok(false);
        }
        update(&mut self.sketch_memory.curvature)?;
        Ok(true)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn apply_sets_the_cancel_checkpoint_for_the_same_curve() {
        let key = CurveKey(1, 2, SketchEntityId(3));
        let mut state = CurvaturePresentation::default();
        assert!(state.setup(key, 7).is_err());
        state.toggle(key);
        state.setup(key, 7).unwrap();
        let applied = GuiCurvatureSliders {
            density: 80,
            ..START
        };
        state.preview(applied).unwrap();
        state.accept(false);
        assert_eq!(state.draft.as_ref().unwrap().view.applied, applied);
        state
            .preview(GuiCurvatureSliders { scale: 85, ..START })
            .unwrap();
        state.discard_draft();
        state.setup(key, 7).unwrap();
        assert_eq!(state.draft.as_ref().unwrap().view.preview, applied);
        state.toggle(key);
        assert!(state.draft.is_none());
        assert_eq!(state.curves.get(&key).unwrap().sliders, applied);
    }

    #[test]
    fn closing_a_part_releases_its_display_and_open_dialog() {
        let first = CurveKey(1, 2, SketchEntityId(3));
        let second = CurveKey(4, 2, SketchEntityId(3));
        let mut state = CurvaturePresentation::default();
        state.toggle(first);
        state.toggle(second);
        state.setup(first, 7).unwrap();
        state.forget_document(first.0);
        assert!(state.draft.is_none());
        assert!(!state.curves.contains_key(&first));
        assert!(state.shown(second));
    }
}
