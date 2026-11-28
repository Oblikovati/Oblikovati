// SPDX-License-Identifier: GPL-2.0-only
//! Interactive User Coordinate System (UCS) placement, viewport picking, and persistence (#3918).
#[path = "owner_ucs_pick.rs"]
mod pick;

use super::*;
use oblikovati_gui::{GuiUcsAction, GuiUcsPlacementView, GuiUcsStage, GuiUcsState};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition, ucs::UcsDefinition},
    history::DocumentSnapshot,
    identity::ReferenceEnvelope,
};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey};
use pick::{
    DEFAULT_TRIAD_REACH, add_scaled, build_triad, compute_triad_reach, nearest_vertex,
    orthogonal_partner, ray_point_distance, ray_segment_distance, resolve_direction_from_ray,
};
use std::collections::BTreeMap;

pub(crate) type SolidVertex = (Vec<u8>, [f64; 3]);

#[derive(Clone, Debug)]
struct ActiveUcsPlacement {
    document: u64,
    generation: u64,
    redefining: Option<FeatureId>,
    stage: GuiUcsStage,
    origin: Option<[f64; 3]>,
    vertex_ref: Option<Vec<u8>>,
    x_dir: [f64; 3],
    y_dir: [f64; 3],
    reach: f64,
}

#[derive(Default)]
pub(in crate::owner) struct UcsOwnerState {
    active: Option<ActiveUcsPlacement>,
}

pub(in crate::owner) fn browser_entries(snapshot: &DocumentSnapshot) -> BTreeMap<FeatureId, bool> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Ucs(definition) = feature.definition() else {
                return None;
            };
            Some((
                feature.id(),
                feature.is_visible() && definition.is_visible(),
            ))
        })
        .collect()
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
    pub(in crate::owner) fn gui_ucs_state(&self, document: u64, generation: u64) -> GuiUcsState {
        let active = self
            .create_features
            .ucs
            .active
            .as_ref()
            .filter(|state| state.document == document && state.generation == generation)
            .map(|state| {
                let preview_triad = state.origin.map(|origin| {
                    build_triad(
                        state.redefining,
                        String::new(),
                        origin,
                        state.x_dir,
                        state.y_dir,
                        state.reach,
                        Some(state.stage),
                    )
                });
                GuiUcsPlacementView {
                    document: state.document,
                    redefining: state.redefining,
                    stage: state.stage,
                    prompt: state.stage.prompt(state.redefining.is_some()).into(),
                    preview_triad,
                }
            });
        let reach = self.ucs_triad_reach(document);
        let triads = self
            .files
            .workspace()
            .session(document)
            .ok()
            .map(|session| {
                session
                    .snapshot()
                    .work_geometry
                    .features()
                    .iter()
                    .filter_map(|feature| {
                        let PartFeatureDefinition::Ucs(def) = feature.definition() else {
                            return None;
                        };
                        if !(feature.is_visible() && def.is_visible()) {
                            return None;
                        }
                        if active
                            .as_ref()
                            .is_some_and(|view| view.redefining == Some(feature.id()))
                        {
                            return None;
                        }
                        Some(build_triad(
                            Some(feature.id()),
                            feature.name().into(),
                            def.origin(),
                            def.x_dir(),
                            def.y_dir(),
                            reach,
                            None,
                        ))
                    })
                    .collect()
            })
            .unwrap_or_default();
        GuiUcsState { active, triads }
    }

    pub(in crate::owner) fn ucs_command(&mut self, action: GuiUcsAction) -> Result<bool, String> {
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(false);
        };
        self.require_browser_document(document)?;
        match action {
            GuiUcsAction::Start => self.start_ucs(document, None),
            GuiUcsAction::Redefine(feature) => self.start_ucs(document, Some(feature)),
            GuiUcsAction::Cancel => Ok(self.cancel_ucs()),
            GuiUcsAction::SetVisibility { feature, visible } => {
                self.set_ucs_visibility(document, feature, visible)
            }
        }
    }

    pub(in crate::owner) fn process_ucs_inputs(&mut self, inputs: &[ViewportEvent]) -> bool {
        if self.create_features.ucs.active.is_none() {
            return false;
        }
        let cancelled = inputs.iter().any(|event| {
            matches!(
                event,
                ViewportEvent::Input(ViewportInput::Key {
                    key: ViewportKey::Cancel,
                    pressed: true,
                }) | ViewportEvent::Input(ViewportInput::PhysicalKey {
                    key: ViewportPhysicalKey::Escape,
                    pressed: true,
                    ..
                })
            )
        });
        if cancelled {
            return self.cancel_ucs();
        }
        false
    }

    pub(in crate::owner) fn pick_ucs_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        depth_window: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(mut state) = self.create_features.ucs.active.clone() else {
            return Ok(None);
        };
        if state.document != document {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        if state.generation != generation {
            self.create_features.ucs.active = None;
            return Ok(None);
        }
        match state.stage {
            GuiUcsStage::PickOrigin => {
                let vertices = self.solid_vertices(document)?;
                let tolerance = depth_window.unwrap_or(0.08).mul_add(8.0, state.reach * 0.4);
                let Some((key, point)) = nearest_vertex(&vertices, ray, tolerance) else {
                    return Ok(Some(false));
                };
                state.origin = Some(point);
                state.vertex_ref = Some(key);
                state.stage = GuiUcsStage::PickXDirection;
                let prompt = state.stage.prompt(state.redefining.is_some()).into();
                self.create_features.ucs.active = Some(state);
                self.command_notice =
                    Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
                Ok(Some(true))
            }
            GuiUcsStage::PickXDirection => {
                let Some(origin) = state.origin else {
                    return Ok(Some(false));
                };
                let x_tip = add_scaled(origin, state.x_dir, state.reach);
                let on_handle = ray_point_distance(x_tip, ray) <= state.reach * 0.55
                    || ray_segment_distance(origin, x_tip, ray) <= state.reach * 0.30;
                if !on_handle {
                    let vertices = self.solid_vertices(document)?;
                    if let Some(next_x) =
                        resolve_direction_from_ray(origin, ray, &vertices, state.reach, None)
                    {
                        state.x_dir = next_x;
                        state.y_dir = orthogonal_partner(next_x, state.y_dir);
                    }
                }
                state.stage = GuiUcsStage::PickYDirection;
                let prompt = state.stage.prompt(state.redefining.is_some()).into();
                self.create_features.ucs.active = Some(state);
                self.command_notice =
                    Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
                Ok(Some(true))
            }
            GuiUcsStage::PickYDirection => {
                let Some(origin) = state.origin else {
                    return Ok(Some(false));
                };
                let y_tip = add_scaled(origin, state.y_dir, state.reach);
                let on_handle = ray_point_distance(y_tip, ray) <= state.reach * 0.65
                    || ray_segment_distance(origin, y_tip, ray) <= state.reach * 0.35;
                if !on_handle {
                    let vertices = self.solid_vertices(document)?;
                    if let Some(next_y) = resolve_direction_from_ray(
                        origin,
                        ray,
                        &vertices,
                        state.reach,
                        Some(state.x_dir),
                    ) {
                        state.y_dir = next_y;
                    }
                }
                self.commit_ucs(document, &state)?;
                self.create_features.ucs.active = None;
                self.command_notice = Some(DesktopCommandNotice::Ready);
                Ok(Some(true))
            }
        }
    }

    fn start_ucs(&mut self, document: u64, redefining: Option<FeatureId>) -> Result<bool, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let (origin, vertex_ref, x_dir, y_dir) = if let Some(feature_id) = redefining {
            let feature = session
                .snapshot()
                .work_geometry
                .get(feature_id)
                .map_err(|cause| cause.to_string())?;
            let PartFeatureDefinition::Ucs(def) = feature.definition() else {
                return Err(format!("feature {feature_id:?}: expected a UCS feature"));
            };
            (
                Some(def.origin()),
                def.vertex_ref().map(ToOwned::to_owned),
                def.x_dir(),
                def.y_dir(),
            )
        } else {
            (None, None, [1.0, 0.0, 0.0], [0.0, 1.0, 0.0])
        };
        let reach = self.ucs_triad_reach(document);
        let stage = GuiUcsStage::PickOrigin;
        self.create_features.ucs.active = Some(ActiveUcsPlacement {
            document,
            generation,
            redefining,
            stage,
            origin,
            vertex_ref,
            x_dir,
            y_dir,
            reach,
        });
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: stage.prompt(redefining.is_some()).into(),
        });
        Ok(true)
    }

    fn cancel_ucs(&mut self) -> bool {
        let had = self.create_features.ucs.active.take().is_some();
        if had {
            self.command_notice = Some(DesktopCommandNotice::Ready);
        }
        had
    }

    fn commit_ucs(&mut self, document: u64, state: &ActiveUcsPlacement) -> Result<(), String> {
        let origin = state
            .origin
            .ok_or("UCS commit: expected a selected origin location")?;
        let definition = UcsDefinition::new(
            origin,
            state.x_dir,
            state.y_dir,
            state.vertex_ref.clone(),
            true,
        )?;
        match state.redefining {
            Some(feature) => {
                self.files
                    .commands(document)
                    .map_err(|cause| cause.to_string())?
                    .run("Redefine UCS", |snapshot, _| {
                        snapshot
                            .edit_ucs(feature, definition)
                            .map_err(|cause| oblikovati_api::Error(cause.to_string()))
                    })
                    .map_err(|cause| cause.to_string())?;
            }
            None => {
                self.files
                    .commands(document)
                    .map_err(|cause| cause.to_string())?
                    .run("User Coordinate System", |snapshot, ids| {
                        let name = snapshot.next_ucs_name();
                        snapshot
                            .create_ucs(&name, definition, ids)
                            .map_err(|cause| oblikovati_api::Error(cause.to_string()))
                    })
                    .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }

    fn set_ucs_visibility(
        &mut self,
        document: u64,
        feature: FeatureId,
        visible: bool,
    ) -> Result<bool, String> {
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("UCS Visibility", |snapshot, _| {
                snapshot
                    .set_ucs_visible(feature, visible)
                    .map_err(|cause| oblikovati_api::Error(cause.to_string()))
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }

    #[cfg(test)]
    pub(crate) fn ucs_solid_vertices(&self, document: u64) -> Result<Vec<SolidVertex>, String> {
        self.solid_vertices(document)
    }

    fn solid_vertices(&self, document: u64) -> Result<Vec<SolidVertex>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let history = session.history();
        let mut vertices = Vec::new();
        for body in evaluation.ordered_bodies() {
            let Ok(keys) = body.subentity_reference_keys(history, guid) else {
                continue;
            };
            for key in keys {
                let Ok(envelope) = ReferenceEnvelope::decode(&key) else {
                    continue;
                };
                if envelope.kind != 3 {
                    continue;
                }
                if let Ok(point) = body.subentity_point(history, guid, &key) {
                    vertices.push((key, point));
                }
            }
        }
        Ok(vertices)
    }

    fn ucs_triad_reach(&self, document: u64) -> f64 {
        self.solid_vertices(document)
            .map(|vertices| compute_triad_reach(&vertices))
            .unwrap_or(DEFAULT_TRIAD_REACH)
    }
}
