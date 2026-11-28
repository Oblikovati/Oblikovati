// SPDX-License-Identifier: GPL-2.0-only
//! DesktopOwner lifecycle, browser datum picks, and transactional commits for Work Point (#3918).
#[path = "owner_work_point_pick.rs"]
mod pick;

use super::*;
use oblikovati_gui::{
    GuiGroundedPointState, GuiWorkPointAction, GuiWorkPointMarker, GuiWorkPointMode,
    GuiWorkPointState,
};
use oblikovati_model::feature::{
    AuthoredFeature, FeatureId,
    part::PartFeatureDefinition,
    work_axis::WorkAxisReference,
    work_geometry::{DatumLength, WorkGeometryDefinition, WorkPointReference},
    work_plane::WorkPlaneReference,
    work_point::{WorkPointConstruction, WorkPointDefinition},
};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey};

#[derive(Clone, Debug)]
pub(super) struct WorkPointSession {
    pub(super) document: u64,
    pub(super) mode: GuiWorkPointMode,
    pub(super) planes: Vec<WorkPlaneReference>,
    pub(super) axes: Vec<WorkAxisReference>,
    pub(super) grounded: Option<[f64; 3]>,
}

impl WorkPointSession {
    pub(super) fn prompt(&self) -> String {
        if self.grounded.is_some() {
            return "Move triad or enter coordinates and accept to create grounded work point"
                .into();
        }
        match self.mode {
            GuiWorkPointMode::ThreePlanes => match self.planes.len() {
                0 => "Select three planes".into(),
                1 => "Select second plane".into(),
                _ => "Select third plane".into(),
            },
            GuiWorkPointMode::TwoLines => match self.axes.len() {
                0 => "Select two intersecting lines or axes".into(),
                _ => "Select second line or axis".into(),
            },
            GuiWorkPointMode::PlaneAndLine => {
                match (self.planes.is_empty(), self.axes.is_empty()) {
                    (true, true) => "Select a plane/surface and a line".into(),
                    (false, true) => "Select a line or axis".into(),
                    (true, false) => "Select a plane or surface".into(),
                    (false, false) => "Ready".into(),
                }
            }
            other => other.initial_prompt().into(),
        }
    }

    fn picked_count(&self) -> usize {
        self.planes.len() + self.axes.len() + usize::from(self.grounded.is_some())
    }
}

/// Active desktop Work Point workflow state stored on `CreateFeatureState`.
#[derive(Default)]
pub(in crate::owner) struct WorkPointOwnerState {
    pub(super) session: Option<WorkPointSession>,
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
    pub(in crate::owner) fn gui_work_point_state(
        &self,
        document: u64,
    ) -> Option<GuiWorkPointState> {
        let session = self.create_features.work_point.session.as_ref()?;
        if session.document != document {
            return None;
        }
        Some(GuiWorkPointState {
            document: session.document,
            mode: session.mode,
            prompt: session.prompt(),
            picked_count: session.picked_count(),
            grounded: session.grounded.map(|coordinates| GuiGroundedPointState {
                coordinates,
                screen_origin: None,
                screen_axes: None,
            }),
        })
    }

    pub(in crate::owner) fn gui_work_point_markers(
        &self,
        document: u64,
    ) -> Vec<GuiWorkPointMarker> {
        if self.active_sketch_edit().is_some() {
            return Vec::new();
        }
        let Ok(session) = self.files.workspace().session(document) else {
            return Vec::new();
        };
        let Ok(evaluated) = self.part_scenes.current(session) else {
            return Vec::new();
        };
        let datums = evaluated.datum_evaluation();
        let selected = session.selected_browser_feature();
        let snapshot = session.snapshot();
        let mut markers = Vec::new();
        for (id, position) in datums.points() {
            let Ok(feature) = snapshot.work_geometry.get(id) else {
                continue;
            };
            if !feature.is_visible() {
                continue;
            }
            markers.push(GuiWorkPointMarker {
                id,
                name: feature.name().into(),
                position,
                pixel: None,
                selected: selected == Some(id),
            });
        }
        markers
    }

    pub(in crate::owner) fn work_point_command(
        &mut self,
        action: GuiWorkPointAction,
    ) -> Result<bool, String> {
        match action {
            GuiWorkPointAction::Start(mode) => self.start_work_point_mode(mode),
            GuiWorkPointAction::PickOriginPlane(kind) => {
                self.add_work_point_plane(WorkPlaneReference::Origin(pick::origin_plane(kind)))
            }
            GuiWorkPointAction::PickOriginAxis(axis) => self.add_work_point_axis(axis),
            GuiWorkPointAction::PickOriginCenter => self.pick_work_point_origin_center(),
            GuiWorkPointAction::UpdateGroundedCoords(coords) => {
                if !coords.iter().all(|v| v.is_finite()) {
                    return Ok(false);
                }
                if let Some(session) = self.create_features.work_point.session.as_mut()
                    && session.grounded.is_some()
                {
                    session.grounded = Some(coords);
                    return Ok(true);
                }
                Ok(false)
            }
            GuiWorkPointAction::AcceptGrounded { coordinates } => {
                let Some(document) = self
                    .create_features
                    .work_point
                    .session
                    .as_ref()
                    .map(|s| s.document)
                else {
                    return Ok(false);
                };
                let definition = WorkPointDefinition::new(WorkPointConstruction::Grounded(
                    coordinates.map(DatumLength::Constant),
                ))
                .map_err(|cause| cause.to_string())?;
                self.commit_work_point(
                    document,
                    WorkGeometryDefinition::WorkPoint(Box::new(definition)),
                )?;
                self.finish_work_point_session();
                Ok(true)
            }
            GuiWorkPointAction::Cancel => {
                if self.create_features.work_point.session.is_none() {
                    return Ok(false);
                }
                self.finish_work_point_session();
                Ok(true)
            }
        }
    }

    pub(in crate::owner) fn handle_work_point_escape(&mut self, inputs: &[ViewportEvent]) -> bool {
        if self.create_features.work_point.session.is_none() {
            return false;
        }
        let cancelled = inputs.iter().any(|event| {
            matches!(
                event,
                ViewportEvent::Input(
                    ViewportInput::Key {
                        key: ViewportKey::Cancel,
                        pressed: true
                    } | ViewportInput::PhysicalKey {
                        key: ViewportPhysicalKey::Escape,
                        pressed: true,
                        ..
                    }
                )
            )
        });
        if cancelled {
            self.finish_work_point_session();
        }
        cancelled
    }

    pub(in crate::owner) fn handle_work_point_browser_feature(
        &mut self,
        document: u64,
        feature: FeatureId,
    ) -> Result<Option<bool>, String> {
        let Some(mode) = self
            .create_features
            .work_point
            .session
            .as_ref()
            .filter(|s| s.document == document && s.grounded.is_none())
            .map(|s| s.mode)
        else {
            return Ok(None);
        };
        let definition = {
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            session
                .snapshot()
                .work_geometry
                .get(feature)
                .map_err(|cause| cause.to_string())?
                .definition()
                .clone()
        };
        match definition.kind() {
            "work-plane" if mode.accepts_plane() => self
                .add_work_point_plane(WorkPlaneReference::Feature(feature))
                .map(Some),
            "work-axis" if mode.accepts_axis() => self
                .add_work_point_axis(WorkAxisReference::Feature(feature))
                .map(Some),
            "work-point" if mode.accepts_point() => {
                self.pick_existing_work_point(document, feature).map(Some)
            }
            _ => Ok(None),
        }
    }

    fn start_work_point_mode(&mut self, mode: GuiWorkPointMode) -> Result<bool, String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or("work point: expected an active part document")?;
        self.require_browser_document(document)?;
        let session = WorkPointSession {
            document,
            mode,
            planes: Vec::new(),
            axes: Vec::new(),
            grounded: None,
        };
        let prompt = session.prompt();
        self.create_features.work_point.session = Some(session);
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
        Ok(true)
    }

    pub(super) fn finish_work_point_session(&mut self) {
        self.create_features.work_point.session = None;
        self.command_notice = Some(DesktopCommandNotice::Ready);
    }

    fn add_work_point_plane(&mut self, reference: WorkPlaneReference) -> Result<bool, String> {
        let Some(active) = self.create_features.work_point.session.as_mut() else {
            return Ok(false);
        };
        if active.grounded.is_some() || !active.mode.accepts_plane() {
            return Ok(false);
        }
        if !active.planes.contains(&reference) {
            active.planes.push(reference);
        }
        let document = active.document;
        let ready = pick::ready_construction(active);
        if let Some(construction) = ready {
            let def = WorkPointDefinition::new(construction).map_err(|cause| cause.to_string())?;
            self.commit_work_point(document, WorkGeometryDefinition::WorkPoint(Box::new(def)))?;
            self.finish_work_point_session();
            return Ok(true);
        }
        let prompt = active.prompt();
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
        Ok(true)
    }

    fn add_work_point_axis(&mut self, reference: WorkAxisReference) -> Result<bool, String> {
        let Some(active) = self.create_features.work_point.session.as_mut() else {
            return Ok(false);
        };
        if active.grounded.is_some() || !active.mode.accepts_axis() {
            return Ok(false);
        }
        if !active.axes.contains(&reference) {
            active.axes.push(reference);
        }
        let document = active.document;
        let ready = pick::ready_construction(active);
        if let Some(construction) = ready {
            let def = WorkPointDefinition::new(construction).map_err(|cause| cause.to_string())?;
            self.commit_work_point(document, WorkGeometryDefinition::WorkPoint(Box::new(def)))?;
            self.finish_work_point_session();
            return Ok(true);
        }
        let prompt = active.prompt();
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
        Ok(true)
    }

    fn pick_work_point_origin_center(&mut self) -> Result<bool, String> {
        let Some(active) = self.create_features.work_point.session.as_mut() else {
            return Ok(false);
        };
        let document = active.document;
        if active.mode == GuiWorkPointMode::Grounded {
            active.grounded = Some([0.0, 0.0, 0.0]);
            let prompt = active.prompt();
            self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
            return Ok(true);
        }
        self.commit_work_point(
            document,
            WorkGeometryDefinition::PointAtReference(WorkPointReference::Origin),
        )?;
        self.finish_work_point_session();
        Ok(true)
    }

    fn pick_existing_work_point(
        &mut self,
        document: u64,
        feature: FeatureId,
    ) -> Result<bool, String> {
        let mode = self
            .create_features
            .work_point
            .session
            .as_ref()
            .map(|s| s.mode);
        if mode == Some(GuiWorkPointMode::Grounded) {
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            let evaluated = self.part_scenes.current(session)?.datum_evaluation();
            let pos = evaluated
                .point(feature)
                .map_err(|cause| cause.to_string())?;
            if let Some(active) = self.create_features.work_point.session.as_mut() {
                active.grounded = Some(pos);
                let prompt = active.prompt();
                self.command_notice =
                    Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
            }
            return Ok(true);
        }
        self.commit_work_point(
            document,
            WorkGeometryDefinition::PointAtReference(WorkPointReference::Feature(feature)),
        )?;
        self.finish_work_point_session();
        Ok(true)
    }

    pub(super) fn commit_work_point(
        &mut self,
        document: u64,
        definition: WorkGeometryDefinition,
    ) -> Result<FeatureId, String> {
        self.validate_work_point_before_commit(document, &definition)?;
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Work Point", |snapshot, ids| {
                let id = ids
                    .allocate_feature()
                    .map_err(|cause| oblikovati_api::Error(cause.to_string()))?;
                let name = pick::next_work_point_name(snapshot);
                let feature = AuthoredFeature::datum(id, &name, definition);
                snapshot
                    .work_geometry
                    .append(feature)
                    .map_err(|cause| oblikovati_api::Error(cause.to_string()))?;
                snapshot
                    .work_geometry
                    .set_datum_visible(id, true)
                    .map_err(|cause| oblikovati_api::Error(cause.to_string()))?;
                Ok::<_, oblikovati_api::Error>(id)
            })
            .map_err(|cause| cause.to_string())
    }

    fn validate_work_point_before_commit(
        &self,
        document: u64,
        definition: &WorkGeometryDefinition,
    ) -> Result<(), String> {
        let PartFeatureDefinition::WorkPoint(wp) = definition else {
            return Ok(());
        };
        if wp.requires_named_part_replay() {
            return Ok(());
        }
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluated = self.part_scenes.current(session)?.datum_evaluation();
        wp.evaluate(&session.snapshot().parameters, &evaluated)
            .map(|_| ())
            .map_err(|cause| cause.to_string())
    }
}
