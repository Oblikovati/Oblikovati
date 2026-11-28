// SPDX-License-Identifier: GPL-2.0-only
//! DesktopOwner Work Plane authoring: viewport picking, live plane preview, creation, and visibility (#3918).

use super::*;
#[path = "owner_work_plane_pick.rs"]
mod pick;
use oblikovati_gui::{
    GuiOffsetPlaneBase, GuiOriginPlaneKind, GuiWorkPlaneAction, GuiWorkPlaneMode, GuiWorkPlanePick,
    GuiWorkPlaneRequest, GuiWorkPlaneViewportPick,
};
use oblikovati_model::{
    feature::{
        FeatureId, work_axis::WorkAxisReference, work_geometry::WorkPlaneDisplaySettings,
        work_plane::WorkPlaneOrigin,
    },
    quantity::Dimension,
    sketch::SketchPlane,
};

/// Active Work Plane command state owned by `DesktopOwner`.
#[derive(Default)]
pub(in crate::owner) struct WorkPlaneOwnerState {
    pub(in crate::owner) active_mode: Option<GuiWorkPlaneMode>,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) last_pick: Option<GuiWorkPlaneViewportPick>,
    pub(in crate::owner) preview: Option<(u64, SketchPlane)>,
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
    pub(in crate::owner) fn work_plane_command(
        &mut self,
        action: GuiWorkPlaneAction,
    ) -> Result<bool, String> {
        match action {
            GuiWorkPlaneAction::Start(mode) => {
                self.create_features.work_plane.active_mode = Some(mode);
                self.create_features.work_plane.preview = None;
                Ok(true)
            }
            GuiWorkPlaneAction::Cancel | GuiWorkPlaneAction::ClearPreview => {
                let changed = self.create_features.work_plane.active_mode.is_some()
                    || self.create_features.work_plane.preview.is_some();
                if matches!(action, GuiWorkPlaneAction::Cancel) {
                    self.create_features.work_plane.active_mode = None;
                }
                self.create_features.work_plane.preview = None;
                Ok(changed)
            }
            GuiWorkPlaneAction::PreviewOffset {
                document,
                base,
                offset,
            } => self.preview_offset_work_plane(document, base, &offset),
            GuiWorkPlaneAction::Preview(request) => self.preview_extended_work_plane(&request),
            GuiWorkPlaneAction::Commit(request) => self.commit_extended_work_plane(&request),
        }
    }

    pub(in crate::owner) fn set_datum_visible(
        &mut self,
        document: u64,
        feature: FeatureId,
        visible: bool,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let current = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .work_geometry
            .get(feature)
            .map_err(|cause| cause.to_string())?
            .is_visible();
        if current == visible {
            return Ok(false);
        }
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Visibility", |snapshot, _| {
                snapshot
                    .work_geometry
                    .set_datum_visible(feature, visible)
                    .map_err(|cause| cause.to_string())
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }

    fn preview_offset_work_plane(
        &mut self,
        document: u64,
        base: GuiOffsetPlaneBase,
        offset: &str,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let Ok(quantity) = session
            .snapshot()
            .parameters
            .evaluate_scalar_expression(offset, Dimension::Length)
        else {
            self.create_features.work_plane.preview = None;
            return Ok(true);
        };
        let Some(base_plane) = self.resolve_offset_base_plane(document, base)? else {
            self.create_features.work_plane.preview = None;
            return Ok(true);
        };
        let mut recipe = base_plane.recipe();
        let normal = base_plane.normal();
        recipe.origin = std::array::from_fn(|i| recipe.origin[i] + normal[i] * quantity.value);
        let shifted = SketchPlane::from_recipe(recipe, base_plane.options())
            .map_err(|cause| cause.to_string())?;
        self.create_features.work_plane.preview = Some((document, shifted));
        Ok(true)
    }

    fn resolve_offset_base_plane(
        &self,
        document: u64,
        base: GuiOffsetPlaneBase,
    ) -> Result<Option<SketchPlane>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluation = self.part_scenes.current(session)?;
        let datums = evaluation.datum_evaluation();
        match base {
            GuiOffsetPlaneBase::Origin(kind) => Ok(Some(origin_kind_plane(kind))),
            GuiOffsetPlaneBase::WorkPlane(id) => Ok(
                oblikovati_model::feature::work_plane::WorkPlaneResolver::plane(&datums, id).ok(),
            ),
            GuiOffsetPlaneBase::SelectedFace => {
                let Some(face) = self.chosen_model_face(document) else {
                    return Ok(None);
                };
                let guid = workspace
                    .identity(document)
                    .map_err(|cause| cause.to_string())?
                    .internal_name;
                let plane = evaluation
                    .inspect_body_reference(face.feature, &face.key, |body| {
                        body.face_plane(session.history(), guid, &face.key)
                    })
                    .ok();
                Ok(plane)
            }
        }
    }

    fn preview_extended_work_plane(
        &mut self,
        request: &GuiWorkPlaneRequest,
    ) -> Result<bool, String> {
        self.require_browser_document(request.document)?;
        if request.mode != GuiWorkPlaneMode::AngleToPlaneAroundEdge || request.picks.len() != 2 {
            self.create_features.work_plane.preview = None;
            return Ok(false);
        }
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let normalized = normalize_angle_text(&request.angle);
        let Ok(angle_qty) = session
            .snapshot()
            .parameters
            .evaluate_scalar_expression(&normalized, Dimension::Angle)
        else {
            self.create_features.work_plane.preview = None;
            return Ok(true);
        };
        let Some(line) = self.resolve_pick_line(request.document, &request.picks[0])? else {
            self.create_features.work_plane.preview = None;
            return Ok(true);
        };
        let Some(base) = self.resolve_pick_plane(request.document, &request.picks[1])? else {
            self.create_features.work_plane.preview = None;
            return Ok(true);
        };
        let rotated = base
            .at_angle_about_axis(line.origin(), line.direction(), angle_qty.value)
            .ok();
        self.create_features.work_plane.preview = rotated.map(|plane| (request.document, plane));
        Ok(true)
    }

    fn resolve_pick_plane(
        &self,
        document: u64,
        pick: &GuiWorkPlanePick,
    ) -> Result<Option<SketchPlane>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluation = self.part_scenes.current(session)?;
        let datums = evaluation.datum_evaluation();
        match pick {
            GuiWorkPlanePick::OriginPlane(kind) => Ok(Some(origin_kind_plane(*kind))),
            GuiWorkPlanePick::WorkPlane(id) => Ok(
                oblikovati_model::feature::work_plane::WorkPlaneResolver::plane(&datums, *id).ok(),
            ),
            GuiWorkPlanePick::TopologyFace(key) => {
                let guid = workspace
                    .identity(document)
                    .map_err(|cause| cause.to_string())?
                    .internal_name;
                for body in evaluation.ordered_bodies() {
                    if let Ok(plane) = body.face_plane(session.history(), guid, key) {
                        return Ok(Some(plane));
                    }
                }
                Ok(None)
            }
            _ => Ok(None),
        }
    }

    fn resolve_pick_line(
        &self,
        document: u64,
        pick: &GuiWorkPlanePick,
    ) -> Result<Option<oblikovati_model::feature::work_axis::WorkAxisLine>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluation = self.part_scenes.current(session)?;
        let datums = evaluation.datum_evaluation();
        match pick {
            GuiWorkPlanePick::OriginAxis(axis) => Ok(axis.line(&datums).ok()),
            GuiWorkPlanePick::WorkAxis(id) => Ok(datums.axis(*id).ok()),
            GuiWorkPlanePick::TopologyEdge(key) => {
                let guid = workspace
                    .identity(document)
                    .map_err(|cause| cause.to_string())?
                    .internal_name;
                for body in evaluation.ordered_bodies() {
                    if let Ok(line) = body.resolve_straight_edge(session.history(), guid, key) {
                        return Ok(Some(line));
                    }
                }
                Ok(None)
            }
            _ => Ok(None),
        }
    }

    fn commit_extended_work_plane(
        &mut self,
        request: &GuiWorkPlaneRequest,
    ) -> Result<bool, String> {
        self.require_browser_document(request.document)?;
        let refs: Vec<String> = request
            .picks
            .iter()
            .map(|pick| self.encode_work_plane_pick(request.document, pick))
            .collect::<Result<_, _>>()?;
        let kind = api_kind_for_mode(request.mode)?;
        let angle = if request.mode == GuiWorkPlaneMode::AngleToPlaneAroundEdge {
            normalize_angle_text(&request.angle)
        } else {
            String::new()
        };
        let args = oblikovati_api::wire::CreateWorkPlaneArgs {
            kind: kind.to_owned(),
            refs: Some(refs),
            angle,
            ..Default::default()
        };
        oblikovati_api::client::Client::new(self.document_host())
            .work_planes()
            .create(&args)
            .map_err(|cause| cause.to_string())?;
        self.create_features.work_plane.active_mode = None;
        self.create_features.work_plane.preview = None;
        Ok(true)
    }

    fn encode_work_plane_pick(
        &self,
        document: u64,
        pick: &GuiWorkPlanePick,
    ) -> Result<String, String> {
        use oblikovati_api::types::{
            WORK_REF_CENTER, WORK_REF_X_AXIS, WORK_REF_XY_PLANE, WORK_REF_XZ_PLANE,
            WORK_REF_Y_AXIS, WORK_REF_YZ_PLANE, WORK_REF_Z_AXIS,
        };
        match pick {
            GuiWorkPlanePick::OriginPlane(GuiOriginPlaneKind::Xy) => Ok(WORK_REF_XY_PLANE.into()),
            GuiWorkPlanePick::OriginPlane(GuiOriginPlaneKind::Xz) => Ok(WORK_REF_XZ_PLANE.into()),
            GuiWorkPlanePick::OriginPlane(GuiOriginPlaneKind::Yz) => Ok(WORK_REF_YZ_PLANE.into()),
            GuiWorkPlanePick::OriginAxis(WorkAxisReference::OriginX) => Ok(WORK_REF_X_AXIS.into()),
            GuiWorkPlanePick::OriginAxis(WorkAxisReference::OriginY) => Ok(WORK_REF_Y_AXIS.into()),
            GuiWorkPlanePick::OriginAxis(WorkAxisReference::OriginZ) => Ok(WORK_REF_Z_AXIS.into()),
            GuiWorkPlanePick::OriginAxis(WorkAxisReference::Feature(id))
            | GuiWorkPlanePick::WorkAxis(id)
            | GuiWorkPlanePick::WorkPoint(id) => Ok(format!("datum/v1/{}", id.0)),
            GuiWorkPlanePick::OriginPoint => Ok(WORK_REF_CENTER.into()),
            GuiWorkPlanePick::WorkPlane(id) => {
                let session = self
                    .files
                    .workspace()
                    .session(document)
                    .map_err(|cause| cause.to_string())?;
                oblikovati_automation::work_plane_reference(session.snapshot(), *id)
                    .map_err(|cause| cause.to_string())
            }
            GuiWorkPlanePick::TopologyFace(key)
            | GuiWorkPlanePick::TopologyEdge(key)
            | GuiWorkPlanePick::TopologyVertex(key) => Ok(hex_key(key)),
        }
    }
}

pub(in crate::owner) fn preview_datum_plane(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    frame: SketchPlane,
    theme: &oblikovati_model::theme::ThemeDefinition,
) -> Result<oblikovati_viewport::ViewportDatumPlane, String> {
    let units = snapshot.parameters.units();
    let corners = WorkPlaneDisplaySettings::default()
        .current_corners(Some(frame), units)
        .map_err(|cause| cause.to_string())?
        .ok_or("work plane preview: expected display corners")?;
    let scale = units.working_scale();
    let [first, second] = corners.map(|pt| pt.map(|c| c / scale));
    let a = frame.to_sketch(first).map_err(|cause| cause.to_string())?;
    let b = frame.to_sketch(second).map_err(|cause| cause.to_string())?;
    let pts = [a, [b[0], a[1]], b, [a[0], b[1]]];
    let mut world = [[0.0; 3]; 4];
    for (out, pt) in world.iter_mut().zip(pts) {
        *out = frame.to_model(pt).map_err(|cause| cause.to_string())?;
    }
    let plane = oblikovati_viewport::ViewportDatumPlane {
        corners: world,
        fill: theme.color(&oblikovati_api::types::TOKEN_PLANE_FILL),
        border: theme.color(&oblikovati_api::types::TOKEN_PLANE_SELECTED),
    };
    plane.validate()?;
    Ok(plane)
}

fn origin_kind_plane(kind: GuiOriginPlaneKind) -> SketchPlane {
    match kind {
        GuiOriginPlaneKind::Xy => WorkPlaneOrigin::XY.plane(),
        GuiOriginPlaneKind::Xz => WorkPlaneOrigin::XZ.plane(),
        GuiOriginPlaneKind::Yz => WorkPlaneOrigin::YZ.plane(),
    }
}

fn api_kind_for_mode(mode: GuiWorkPlaneMode) -> Result<&'static str, String> {
    use oblikovati_api::types::*;
    Ok(match mode {
        GuiWorkPlaneMode::MidplaneTwoPlanes => WORK_PLANE_TWO_PLANES.0.as_ref(),
        GuiWorkPlaneMode::ThreePoints => WORK_PLANE_THREE_POINTS.0.as_ref(),
        GuiWorkPlaneMode::ParallelThroughPoint => WORK_PLANE_PLANE_AND_POINT.0.as_ref(),
        GuiWorkPlaneMode::NormalToAxisThroughPoint => WORK_PLANE_NORMAL_TO_CURVE.0.as_ref(),
        GuiWorkPlaneMode::AngleToPlaneAroundEdge => WORK_PLANE_LINE_PLANE_ANGLE.0.as_ref(),
        GuiWorkPlaneMode::TwoCoplanarEdges => WORK_PLANE_TWO_LINES.0.as_ref(),
        GuiWorkPlaneMode::LineThroughPoint => WORK_PLANE_LINE_AND_POINT.0.as_ref(),
        GuiWorkPlaneMode::MidplaneOfTorus => WORK_PLANE_TORUS_MID_PLANE.0.as_ref(),
        GuiWorkPlaneMode::TangentThroughEdge => WORK_PLANE_LINE_AND_TANGENT.0.as_ref(),
        GuiWorkPlaneMode::TangentThroughPoint => WORK_PLANE_POINT_AND_TANGENT.0.as_ref(),
        GuiWorkPlaneMode::TangentParallelToPlane => WORK_PLANE_PLANE_AND_TANGENT.0.as_ref(),
        GuiWorkPlaneMode::General => {
            return Err("general plane mode: select a supported combination of references".into());
        }
    })
}

fn normalize_angle_text(raw: &str) -> String {
    let trimmed = raw.trim();
    if let Some(stem) = trimmed.strip_suffix("degrees") {
        return format!("{}deg", stem);
    }
    if let Some(stem) = trimmed.strip_suffix("degree") {
        return format!("{}deg", stem);
    }
    if let Some(stem) = trimmed.strip_suffix('°') {
        return format!("{} deg", stem.trim_end());
    }
    trimmed.to_owned()
}

fn hex_key(bytes: &[u8]) -> String {
    let mut out = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        use std::fmt::Write as _;
        let _ = write!(&mut out, "{byte:02x}");
    }
    out
}
