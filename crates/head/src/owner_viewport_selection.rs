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
        if let Some(changed) = self.pick_hole_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_slot_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_combine_geometry(document, line, depth_window)? {
        if let Some(changed) = self.pick_draft_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_fillet_v2_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_chamfer_v2_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_shell_v2_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_revolve_geometry(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_loft_section(document, line, depth_window)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_sweep_profile(document, line)? {
            return Ok(changed);
        }
        if let Some(changed) = self.pick_sweep_path(document, line, depth_window)? {
            return Ok(changed);
        }
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
        self.click_part_with_window(document, line, depth_window, toggle, false)
    }
    pub(in crate::owner) fn context_click_model(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        self.click_part_with_window(document, line, depth_window, false, true)
    }
    fn click_part_with_window(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
        depth_window: Option<f64>,
        toggle: bool,
        preserve_on_miss: bool,
    ) -> Result<bool, String> {
        if !preserve_on_miss {
            self.model_pick_cue = None;
        }
        let Some(allowed) = self.part_pick_filter(document, depth_window.is_some()) else {
            return Ok(false);
        };
        let mut candidates =
            self.part_scenes
                .part_candidates(&self.files, document, line, &allowed)?;
        if allowed.accepts(oblikovati_session::selection::SelectionKind::Edge)
            && let Some(tolerance) = depth_window
        {
            if self.part_scenes.edge_pick_available() {
                if let Err(cause) = self.part_scenes.append_edge_candidates(
                    &self.files,
                    document,
                    line,
                    tolerance,
                    &mut candidates,
                ) {
                    self.command_notice =
                        Some(crate::DesktopCommandNotice::SelectionFailed { cause });
                    return Ok(true);
                }
                candidates.include_edges();
            } else if allowed.is_restricted()
                && !allowed.accepts(oblikovati_session::selection::SelectionKind::Face)
                && !allowed.accepts(oblikovati_session::selection::SelectionKind::Body)
            {
                self.command_notice = Some(crate::DesktopCommandNotice::SelectionFailed {
                    cause: "edge selection: expected a certified model resolution".into(),
                });
                return Ok(true);
            }
        }
        if allowed.accepts(oblikovati_session::selection::SelectionKind::WorkPlane) {
            self.append_visible_plane_hits(document, line, &mut candidates)?;
        }
        let picked = candidates.resolve_hit(
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
        if preserve_on_miss && picked.is_none() {
            return Ok(false);
        }
        let cue = if toggle {
            None
        } else {
            self.face_cue_from_hit(document, line, picked.as_ref())?
        };
        let changed = self.apply_part_candidate(document, picked.map(|hit| hit.target), toggle)?;
        self.model_pick_cue = cue;
        Ok(changed)
    }
    fn part_pick_filter(
        &self,
        document: u64,
        has_window: bool,
    ) -> Option<oblikovati_session::selection::SelectionFilter> {
        use oblikovati_session::selection::{SelectionFilter, SelectionKind};
        let measure = self.measuring_part(document);
        if !measure && !has_window && !self.ambient_selection.filter().accepts(SelectionKind::Face)
        {
            return None;
        }
        let allowed = if measure && has_window {
            SelectionFilter::only([SelectionKind::Edge, SelectionKind::Face])
        } else if measure || !has_window {
            SelectionFilter::only([SelectionKind::Face])
        } else {
            self.ambient_selection.filter()
        };
        (allowed.accepts(SelectionKind::Face)
            || allowed.accepts(SelectionKind::Edge)
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
        if let Some(crate::part_scene::pick_candidates::PickTarget::Edge(feature, key)) = &picked {
            let changed = self.select_model_edge(document, *feature, key.clone(), toggle)?;
            self.measure_part_edges(document)?;
            return Ok(changed);
        }
        if !toggle {
            self.model_edges.clear();
        }
        if let Some(crate::part_scene::pick_candidates::PickTarget::WorkPlane(feature)) = &picked {
            if toggle {
                return self.unsupported_selection_toggle("work plane");
            }
            return self.select_browser_feature(document, *feature);
        }
        if let Some(crate::part_scene::pick_candidates::PickTarget::Body(feature, key)) = &picked {
            if toggle
                && self.chosen_model_face(document).is_some()
                && self.current_face_appearance_picks(document).is_empty()
            {
                return self.unsupported_selection_toggle("unresolved face");
            }
            return self.browser_choice(
                document,
                &oblikovati_gui::GuiBrowserChoice::Body {
                    feature: *feature,
                    key: key.clone(),
                    toggle,
                },
            );
        }
        if toggle && picked.is_none() {
            return Ok(false);
        }
        self.selected_occurrence = None;
        let bodies_selected = !self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .selected_browser_bodies()
            .is_empty();
        if !toggle || !bodies_selected {
            self.files
                .commands(document)
                .map_err(|cause| cause.to_string())?
                .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
                .map_err(|cause| cause.to_string())?;
        }
        let picked = match picked {
            Some(crate::part_scene::pick_candidates::PickTarget::Face(feature, role)) => {
                Some((feature, role))
            }
            None => None,
            Some(other) => return Err(format!("part hit {other:?}: expected a face or body")),
        };
        self.measure_part_face(document, picked.clone())?;
        let selected = picked.map(|(feature, key)| ChosenModelFace {
            document,
            feature,
            key,
        });
        let previous = self.current_face_appearance_picks(document).to_vec();
        let picked = self.capture_face_appearance_pick(document, selected.clone());
        let unresolved_face = if !toggle && picked.is_none() {
            selected
        } else {
            None
        };
        let mut selected_faces = if toggle { previous } else { Vec::new() };
        if let Some(pick) = picked {
            if let Some(index) = selected_faces.iter().position(|item| item.key == pick.key) {
                selected_faces.remove(index);
            } else {
                selected_faces.push(pick);
            }
        }
        self.model_face = selected_faces
            .last()
            .map(|pick| pick.face.clone())
            .or(unresolved_face);
        self.face_appearance_picks = selected_faces;
        Ok(true)
    }
    fn select_model_edge(
        &mut self,
        document: u64,
        feature: oblikovati_model::feature::FeatureId,
        key: Vec<u8>,
        toggle: bool,
    ) -> Result<bool, String> {
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        if !toggle || self.chosen_model_edges(document).len() != self.model_edges.len() {
            self.model_edges.clear();
        }
        if !toggle {
            self.files
                .commands(document)
                .map_err(|cause| cause.to_string())?
                .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
                .map_err(|cause| cause.to_string())?;
            self.model_face = None;
            self.face_appearance_picks.clear();
        }
        self.selected_occurrence = None;
        if let Some(index) = self.model_edges.iter().position(|edge| edge.key == key) {
            if toggle {
                self.model_edges.remove(index);
            }
        } else {
            self.model_edges.push(ChosenModelEdge {
                document,
                generation,
                feature,
                key,
            });
        }
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
