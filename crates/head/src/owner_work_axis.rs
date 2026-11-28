// SPDX-License-Identifier: GPL-2.0-only
//! Interactive Work Axis command state, reference accumulation, and commit (#3918).
#[path = "owner_work_axis_pick.rs"]
mod pick;

use super::*;
use oblikovati_api::{client::Client, types::WORK_AXIS_LINE, wire::CreateWorkAxisArgs};
use oblikovati_gui::{GuiOriginPlaneKind, GuiWorkAxisAction, GuiWorkAxisMode};
use oblikovati_model::{
    feature::{
        FeatureId,
        work_axis::{WorkAxisLine, WorkAxisReference},
    },
    sketch::SketchPlane,
};

const DEFAULT_AXIS_REACH: f64 = 6.0;

#[derive(Clone, Debug)]
pub(super) struct ActiveWorkAxis {
    pub(super) document: u64,
    pub(super) mode: GuiWorkAxisMode,
    pub(super) first_point: Option<[f64; 3]>,
    pub(super) first_plane: Option<SketchPlane>,
    pub(super) first_line: Option<WorkAxisLine>,
}

impl ActiveWorkAxis {
    pub(super) fn prompt(&self) -> &'static str {
        match self.mode {
            GuiWorkAxisMode::General => "Define work axis by selecting geometry",
            GuiWorkAxisMode::ThroughTwoPoints => {
                if self.first_point.is_none() {
                    "Select first point for work axis"
                } else {
                    "Select second point for work axis"
                }
            }
            GuiWorkAxisMode::OnLineOrEdge => "Select a straight line or edge for work axis",
            GuiWorkAxisMode::IntersectionOfTwoPlanes => {
                if self.first_plane.is_none() {
                    "Select first plane for work axis"
                } else {
                    "Select second plane for work axis"
                }
            }
            GuiWorkAxisMode::ThroughCircularEdge => {
                "Select a circular or elliptical edge for work axis"
            }
            GuiWorkAxisMode::NormalToPlaneThroughPoint => {
                match (self.first_plane, self.first_point) {
                    (None, None) => "Select a plane and a point for work axis",
                    (Some(_), None) => "Select a point for work axis",
                    (None, Some(_)) | (Some(_), Some(_)) => "Select a plane for work axis",
                }
            }
            GuiWorkAxisMode::ParallelToLineThroughPoint => {
                match (self.first_line, self.first_point) {
                    (None, None) => "Select a line and a point for work axis",
                    (Some(_), None) => "Select a point not on the line for work axis",
                    (None, Some(_)) | (Some(_), Some(_)) => "Select a line or axis for work axis",
                }
            }
            GuiWorkAxisMode::ThroughRevolvedFace => {
                "Select a cylindrical or revolved surface for work axis"
            }
        }
    }
}

#[derive(Default)]
pub(in crate::owner) struct WorkAxisOwnerState {
    pub(super) active: Option<ActiveWorkAxis>,
}

/// Compute finite 3D display endpoints for an infinite work axis within optional model bounds.
pub(in crate::owner) fn axis_display_endpoints(
    axis: WorkAxisLine,
    bounds: Option<[[f64; 3]; 2]>,
) -> [[f64; 3]; 2] {
    let origin = axis.origin();
    let dir = axis.direction();
    let (t_min, t_max) = match bounds {
        Some([lo, hi])
            if lo.iter().chain(&hi).all(|v| v.is_finite())
                && lo[0] <= hi[0]
                && lo[1] <= hi[1]
                && lo[2] <= hi[2] =>
        {
            let mut min_t = f64::INFINITY;
            let mut max_t = f64::NEG_INFINITY;
            for &x in &[lo[0], hi[0]] {
                for &y in &[lo[1], hi[1]] {
                    for &z in &[lo[2], hi[2]] {
                        let t = (x - origin[0]) * dir[0]
                            + (y - origin[1]) * dir[1]
                            + (z - origin[2]) * dir[2];
                        min_t = min_t.min(t);
                        max_t = max_t.max(t);
                    }
                }
            }
            let span = (max_t - min_t).max(0.5);
            let margin = (span * 0.35).max(0.3);
            (min_t - margin, max_t + margin)
        }
        _ => (-DEFAULT_AXIS_REACH, DEFAULT_AXIS_REACH),
    };
    let point = |t: f64| {
        [
            origin[0] + dir[0] * t,
            origin[1] + dir[1] * t,
            origin[2] + dir[2] * t,
        ]
    };
    [point(t_min), point(t_max)]
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
    pub(in crate::owner) fn gui_work_axis_prompt(&self, document: u64) -> Option<String> {
        self.create_features
            .work_axis
            .active
            .as_ref()
            .filter(|state| state.document == document)
            .map(|state| state.prompt().to_owned())
    }

    pub(in crate::owner) fn work_axis_command(
        &mut self,
        action: GuiWorkAxisAction,
    ) -> Result<bool, String> {
        match action {
            GuiWorkAxisAction::Cancel => {
                if self.create_features.work_axis.active.take().is_some() {
                    self.command_notice = Some(DesktopCommandNotice::Ready);
                    return Ok(true);
                }
                Ok(false)
            }
            GuiWorkAxisAction::Start(mode) => {
                let document = self
                    .files
                    .workspace()
                    .active_document()
                    .ok_or("work axis: expected an active part document")?;
                self.clear_extrusion_preview();
                let active = ActiveWorkAxis {
                    document,
                    mode,
                    first_point: None,
                    first_plane: None,
                    first_line: None,
                };
                let prompt = active.prompt().to_owned();
                self.create_features.work_axis.active = Some(active);
                self.command_notice =
                    Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
                Ok(true)
            }
            GuiWorkAxisAction::PickOriginPlane(kind) => {
                let Some(document) = self.active_work_axis_document() else {
                    return Ok(false);
                };
                self.feed_work_axis_plane(document, origin_plane_frame(kind))
            }
            GuiWorkAxisAction::PickOriginAxis(reference) => {
                let Some(document) = self.active_work_axis_document() else {
                    return Ok(false);
                };
                let line = origin_axis_line(reference)?;
                self.feed_work_axis_line(document, line)
            }
            GuiWorkAxisAction::PickOriginPoint => {
                let Some(document) = self.active_work_axis_document() else {
                    return Ok(false);
                };
                self.feed_work_axis_point(document, [0.0, 0.0, 0.0])
            }
        }
    }

    pub(in crate::owner) fn handle_work_axis_browser_feature(
        &mut self,
        document: u64,
        feature: FeatureId,
    ) -> Result<bool, String> {
        let Some(active) = self.create_features.work_axis.active.as_ref() else {
            return Ok(false);
        };
        if active.document != document {
            return Ok(false);
        }
        let mode = active.mode;
        let prompt = active.prompt().to_owned();
        let datum = self
            .files
            .workspace()
            .session(document)
            .ok()
            .and_then(|session| self.part_scenes.current(session).ok())
            .map(|eval| eval.datum_evaluation());
        if let Some(evaluated) = datum {
            if let Ok(plane) =
                oblikovati_model::feature::work_plane::WorkPlaneResolver::plane(&evaluated, feature)
                && matches!(
                    mode,
                    GuiWorkAxisMode::IntersectionOfTwoPlanes
                        | GuiWorkAxisMode::NormalToPlaneThroughPoint
                        | GuiWorkAxisMode::General
                )
            {
                return self.feed_work_axis_plane(document, plane);
            }
            if let Ok(axis) = evaluated.axis(feature)
                && matches!(
                    mode,
                    GuiWorkAxisMode::OnLineOrEdge
                        | GuiWorkAxisMode::ParallelToLineThroughPoint
                        | GuiWorkAxisMode::General
                )
            {
                return self.feed_work_axis_line(document, axis);
            }
            if let Ok(point) = evaluated.point(feature)
                && matches!(
                    mode,
                    GuiWorkAxisMode::ThroughTwoPoints
                        | GuiWorkAxisMode::NormalToPlaneThroughPoint
                        | GuiWorkAxisMode::ParallelToLineThroughPoint
                        | GuiWorkAxisMode::General
                )
            {
                return self.feed_work_axis_point(document, point);
            }
        }
        // Clicking an unrelated browser node (e.g. Hole1 during Through Revolved Face) keeps prompting.
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt { message: prompt });
        Ok(false)
    }

    pub(in crate::owner) fn work_axis_displays_for_view(
        &self,
        bounds: Option<[[f64; 3]; 2]>,
    ) -> Vec<crate::HeadWorkAxisSegment> {
        if self.active_sketch_edit().is_some() {
            return Vec::new();
        }
        let Some(document) = self.files.workspace().active_document() else {
            return Vec::new();
        };
        let Ok(session) = self.files.workspace().session(document) else {
            return Vec::new();
        };
        let Ok(evaluated) = self.part_scenes.current(session) else {
            return Vec::new();
        };
        let selected = session.selected_browser_feature();
        let snapshot = session.snapshot();
        evaluated
            .datum_evaluation()
            .axes()
            .filter_map(|(id, axis)| {
                let feature = snapshot.work_geometry.get(id).ok()?;
                if !feature.is_visible() {
                    return None;
                }
                Some(crate::HeadWorkAxisSegment {
                    id,
                    name: feature.name().to_owned(),
                    endpoints: axis_display_endpoints(axis, bounds),
                    selected: selected == Some(id),
                })
            })
            .collect()
    }

    pub(super) fn active_work_axis_document(&self) -> Option<u64> {
        let active = self.create_features.work_axis.active.as_ref()?;
        (self.files.workspace().active_document() == Some(active.document))
            .then_some(active.document)
    }

    pub(super) fn feed_work_axis_point(
        &mut self,
        document: u64,
        point: [f64; 3],
    ) -> Result<bool, String> {
        let Some(active) = self.create_features.work_axis.active.clone() else {
            return Ok(false);
        };
        if active.document != document {
            return Ok(false);
        }
        match active.mode {
            GuiWorkAxisMode::ThroughTwoPoints | GuiWorkAxisMode::General => {
                if let Some(first) = active.first_point {
                    let Ok(line) = WorkAxisLine::through_points(first, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(plane) = active.first_plane {
                    let Ok(line) = WorkAxisLine::normal_to_plane_through_point(plane, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(axis) = active.first_line {
                    let Ok(line) = WorkAxisLine::parallel_to_line_through_point(axis, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(state) = self.create_features.work_axis.active.as_mut() {
                    state.first_point = Some(point);
                }
                self.keep_work_axis_prompt()
            }
            GuiWorkAxisMode::NormalToPlaneThroughPoint => {
                if let Some(plane) = active.first_plane {
                    let Ok(line) = WorkAxisLine::normal_to_plane_through_point(plane, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(state) = self.create_features.work_axis.active.as_mut() {
                    state.first_point = Some(point);
                }
                self.keep_work_axis_prompt()
            }
            GuiWorkAxisMode::ParallelToLineThroughPoint => {
                if let Some(axis) = active.first_line {
                    let Ok(line) = WorkAxisLine::parallel_to_line_through_point(axis, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(state) = self.create_features.work_axis.active.as_mut() {
                    state.first_point = Some(point);
                }
                self.keep_work_axis_prompt()
            }
            _ => self.keep_work_axis_prompt(),
        }
    }

    pub(super) fn feed_work_axis_plane(
        &mut self,
        document: u64,
        plane: SketchPlane,
    ) -> Result<bool, String> {
        let Some(active) = self.create_features.work_axis.active.clone() else {
            return Ok(false);
        };
        if active.document != document {
            return Ok(false);
        }
        match active.mode {
            GuiWorkAxisMode::IntersectionOfTwoPlanes | GuiWorkAxisMode::General => {
                if let Some(first) = active.first_plane {
                    let Ok(line) = WorkAxisLine::plane_intersection(first, plane) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(point) = active.first_point {
                    let Ok(line) = WorkAxisLine::normal_to_plane_through_point(plane, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(state) = self.create_features.work_axis.active.as_mut() {
                    state.first_plane = Some(plane);
                }
                self.keep_work_axis_prompt()
            }
            GuiWorkAxisMode::NormalToPlaneThroughPoint => {
                if let Some(point) = active.first_point {
                    let Ok(line) = WorkAxisLine::normal_to_plane_through_point(plane, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, line);
                }
                if let Some(state) = self.create_features.work_axis.active.as_mut() {
                    state.first_plane = Some(plane);
                }
                self.keep_work_axis_prompt()
            }
            _ => self.keep_work_axis_prompt(),
        }
    }

    pub(super) fn feed_work_axis_line(
        &mut self,
        document: u64,
        line: WorkAxisLine,
    ) -> Result<bool, String> {
        let Some(active) = self.create_features.work_axis.active.clone() else {
            return Ok(false);
        };
        if active.document != document {
            return Ok(false);
        }
        match active.mode {
            GuiWorkAxisMode::OnLineOrEdge => self.commit_work_axis_line(document, line),
            GuiWorkAxisMode::ParallelToLineThroughPoint => {
                if let Some(point) = active.first_point {
                    let Ok(axis) = WorkAxisLine::parallel_to_line_through_point(line, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, axis);
                }
                if let Some(state) = self.create_features.work_axis.active.as_mut() {
                    state.first_line = Some(line);
                }
                self.keep_work_axis_prompt()
            }
            GuiWorkAxisMode::General => {
                if let Some(point) = active.first_point {
                    let Ok(axis) = WorkAxisLine::parallel_to_line_through_point(line, point) else {
                        return self.keep_work_axis_prompt();
                    };
                    return self.commit_work_axis_line(document, axis);
                }
                self.commit_work_axis_line(document, line)
            }
            _ => self.keep_work_axis_prompt(),
        }
    }

    pub(super) fn commit_work_axis_line(
        &mut self,
        document: u64,
        axis: WorkAxisLine,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let request = CreateWorkAxisArgs {
            kind: WORK_AXIS_LINE.0.into(),
            origin: Some(axis.origin().to_vec()),
            direction: Some(axis.direction().to_vec()),
            ..Default::default()
        };
        Client::new(self.document_host())
            .work_axes()
            .create(&request)
            .map_err(|cause| cause.to_string())?;
        self.create_features.work_axis.active = None;
        self.model_face = None;
        self.model_edges.clear();
        self.face_appearance_picks.clear();
        self.command_notice = Some(DesktopCommandNotice::Ready);
        Ok(true)
    }

    pub(super) fn keep_work_axis_prompt(&mut self) -> Result<bool, String> {
        if let Some(active) = self.create_features.work_axis.active.as_ref() {
            self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
                message: active.prompt().to_owned(),
            });
        }
        Ok(true)
    }
}

fn origin_plane_frame(kind: GuiOriginPlaneKind) -> SketchPlane {
    match kind {
        GuiOriginPlaneKind::Xy => SketchPlane::xy(),
        GuiOriginPlaneKind::Xz => SketchPlane::xz(),
        GuiOriginPlaneKind::Yz => SketchPlane::yz(),
    }
}

fn origin_axis_line(reference: WorkAxisReference) -> Result<WorkAxisLine, String> {
    let direction = match reference {
        WorkAxisReference::OriginX => [1.0, 0.0, 0.0],
        WorkAxisReference::OriginY => [0.0, 1.0, 0.0],
        WorkAxisReference::OriginZ => [0.0, 0.0, 1.0],
        WorkAxisReference::Feature(id) => {
            return Err(format!(
                "origin axis {id:?}: expected OriginX, OriginY, or OriginZ"
            ));
        }
    };
    WorkAxisLine::new([0.0; 3], direction).map_err(|cause| cause.to_string())
}
