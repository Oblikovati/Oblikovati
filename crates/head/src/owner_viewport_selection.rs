// SPDX-License-Identifier: GPL-2.0-only
//! Viewport selection dispatch for parts and assemblies.
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Select an eligible viewport hit, or clear the selection on an empty hit.
    /// Example: `owner.click_model(document, view_line)?`.
    pub(in crate::owner) fn click_model(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<bool, String> {
        self.click_model_with_window(document, line, None)
    }
    pub(in crate::owner) fn click_model_with_window(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<bool, String> {
        self.click_model_with_modifiers(document, line, depth_window, false)
    }
    pub(in crate::owner) fn click_model_with_modifiers(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        depth_window: Option<f64>,
        toggle: bool,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        if self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .assembly
            .is_some()
        {
            if !self
                .ambient_selection
                .filter()
                .accepts(oblikovati_session::selection::SelectionKind::Occurrence)
            {
                return Ok(false);
            }
            if toggle {
                return self.unsupported_selection_toggle("assembly occurrence");
            }
            return self.click_assembly(document, line);
        }
        self.click_part_with_window(document, line, depth_window, toggle)
    }
    fn click_part_with_window(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        depth_window: Option<f64>,
        toggle: bool,
    ) -> Result<bool, String> {
        let Some(allowed) = self.part_pick_filter(document, depth_window.is_some()) else {
            return Ok(false);
        };
        let mut candidates = self
            .part_scenes
            .part_candidates(&self.files, document, line)?;
        if allowed.accepts(oblikovati_session::selection::SelectionKind::WorkPlane) {
            self.append_visible_plane_hits(document, line, &mut candidates)?;
        }
        let picked = candidates.resolve(
            &allowed,
            &self.ambient_selection,
            depth_window.unwrap_or(0.),
        );
        let picked = match picked {
            Ok(picked) => picked,
            Err(cause) => {
                self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed { cause });
                return Ok(true);
            }
        };
        self.apply_part_candidate(document, picked, toggle)
    }
    fn part_pick_filter(
        &self,
        document: u64,
        has_window: bool,
    ) -> Option<oblikovati_session::selection::SelectionFilter> {
        use oblikovati_session::selection::{SelectionFilter, SelectionKind};
        let measure = self
            .part_measure
            .as_ref()
            .is_some_and(|tool| tool.document == document);
        if !measure && !has_window && !self.ambient_selection.filter().accepts(SelectionKind::Face)
        {
            return None;
        }
        let allowed = if measure || !has_window {
            SelectionFilter::only([SelectionKind::Face])
        } else {
            self.ambient_selection.filter()
        };
        (allowed.accepts(SelectionKind::Face)
            || allowed.accepts(SelectionKind::Body)
            || allowed.accepts(SelectionKind::WorkPlane))
        .then_some(allowed)
    }
    fn apply_part_candidate(
        &mut self,
        document: u64,
        picked: Option<crate::part_scene::pick_candidates::PickTarget>,
        toggle: bool,
    ) -> Result<bool, String> {
        if let Some(crate::part_scene::pick_candidates::PickTarget::WorkPlane(feature)) = &picked {
            if toggle {
                return self.unsupported_selection_toggle("work plane");
            }
            return self.select_browser_feature(document, *feature);
        }
        if let Some(crate::part_scene::pick_candidates::PickTarget::Body(feature, key)) = &picked {
            return self.browser_choice(
                document,
                &oblikovati_gui::GuiBrowserChoice::Body {
                    feature: *feature,
                    key: key.clone(),
                    toggle,
                },
            );
        }
        if toggle && picked.is_some() {
            return self.unsupported_selection_toggle("face");
        }
        if toggle {
            return Ok(false);
        }
        self.selected_occurrence = None;
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
            .map_err(|cause| cause.to_string())?;
        let picked = match picked {
            Some(crate::part_scene::pick_candidates::PickTarget::Face(feature, role)) => {
                Some((feature, role))
            }
            None => None,
            Some(other) => return Err(format!("part hit {other:?}: expected a face or body")),
        };
        self.measure_part_face(document, picked)?;
        let selected = picked.map(|(feature, role)| ChosenModelFace {
            document,
            feature,
            role,
        });
        self.face_appearance_pick = self.capture_face_appearance_pick(document, selected);
        self.model_face = selected;
        Ok(true)
    }
    fn unsupported_selection_toggle(&mut self, kind: &str) -> Result<bool, String> {
        self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed {
            cause: format!("{kind} viewport toggle: expected multiple selection support"),
        });
        Ok(true)
    }
    fn click_assembly(&mut self, document: u64, line: [[f64; 3]; 2]) -> Result<bool, String> {
        let picked = self
            .part_scenes
            .occurrence_on_line(&mut self.files, document, line);
        let choice = match picked {
            Ok(Some(path)) => oblikovati_gui::GuiBrowserChoice::Occurrence(path),
            Ok(None) => oblikovati_gui::GuiBrowserChoice::Nothing,
            Err(cause) => {
                self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed { cause });
                return Ok(true);
            }
        };
        self.browser_choice(document, &choice)
    }
}
