// SPDX-License-Identifier: GPL-2.0-only
//! Display only current evaluated work planes; this path never changes the model.
use super::*;
use oblikovati_model::{
    feature::{FeatureId, work_geometry::WorkGeometryEvaluation},
    history::DocumentSnapshot,
    sketch::SketchPlane,
    theme::ThemeDefinition,
};
use oblikovati_viewport::ViewportDatumPlane;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn datum_planes_for_view(&mut self) -> Result<crate::HeadDatumPlaneUpdate, String> {
        if self.work_plane_overlays_hidden {
            return Ok(crate::HeadDatumPlaneUpdate::default());
        }
        let Some(document) = self.active_datum_part()? else {
            return Ok(crate::HeadDatumPlaneUpdate::default());
        };
        let theme = datum_theme(&self.themes, &mut self.datum_fallback_theme)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let evaluated = self.part_scenes.current(session)?.datum_evaluation();
        let mut update = planes_for_snapshot(
            session.snapshot(),
            &evaluated,
            session.selected_browser_feature(),
            theme,
        )?;
        if let Some((preview_doc, frame)) = self.create_features.work_plane.preview
            && preview_doc == document
            && let Ok(preview) = super::owner_blend::owner_work_plane::preview_datum_plane(
                session.snapshot(),
                frame,
                theme,
            )
        {
            update.planes.push(preview);
        }
        Ok(update)
    }

    fn active_datum_part(&self) -> Result<Option<u64>, String> {
        if self.active_sketch_edit().is_some() {
            return Ok(None);
        }
        let Some(document) = self.files.workspace().active_document() else {
            return Ok(None);
        };
        let kind = self
            .files
            .document_type(document)
            .map_err(|error| error.to_string())?;
        Ok((kind == oblikovati_api::types::DOCUMENT_PART).then_some(document))
    }
}

fn datum_theme<'a>(
    library: &'a Option<oblikovati_model::theme::ThemeLibrary>,
    fallback: &'a mut Option<Result<ThemeDefinition, String>>,
) -> Result<&'a ThemeDefinition, String> {
    match library {
        Some(library) => Ok(library.active()),
        None => fallback
            .get_or_insert_with(ThemeDefinition::dark)
            .as_ref()
            .map_err(Clone::clone),
    }
}

fn planes_for_snapshot(
    snapshot: &DocumentSnapshot,
    evaluated: &WorkGeometryEvaluation,
    selected: Option<FeatureId>,
    theme: &ThemeDefinition,
) -> Result<crate::HeadDatumPlaneUpdate, String> {
    let mut update = crate::HeadDatumPlaneUpdate::default();
    for (id, frame) in evaluated.planes() {
        let feature = snapshot
            .work_geometry
            .get(id)
            .map_err(|error| error.to_string())?;
        if !feature.is_visible() {
            continue;
        }
        match plane_for_feature(snapshot, feature, frame, selected == Some(id), theme) {
            Ok(plane) => update.planes.push(plane),
            Err(cause) => update.declines.push((id, cause)),
        }
    }
    Ok(update)
}

fn plane_for_feature(
    snapshot: &DocumentSnapshot,
    feature: &oblikovati_model::feature::AuthoredFeature<
        oblikovati_model::feature::work_geometry::WorkGeometryDefinition,
    >,
    frame: SketchPlane,
    selected: bool,
    theme: &ThemeDefinition,
) -> Result<ViewportDatumPlane, String> {
    let corners = feature_corners(snapshot, feature, frame)?;
    let token = if selected {
        oblikovati_api::types::TOKEN_PLANE_SELECTED
    } else {
        oblikovati_api::types::TOKEN_PLANE_FAINT
    };
    let plane = ViewportDatumPlane {
        corners,
        fill: theme.color(&oblikovati_api::types::TOKEN_PLANE_FILL),
        border: theme.color(&token),
    };
    plane.validate()?;
    Ok(plane)
}

pub(super) fn feature_corners(
    snapshot: &DocumentSnapshot,
    feature: &oblikovati_model::feature::AuthoredFeature<
        oblikovati_model::feature::work_geometry::WorkGeometryDefinition,
    >,
    frame: SketchPlane,
) -> Result<[[f64; 3]; 4], String> {
    let units = snapshot.parameters.units();
    let corners = feature
        .plane_display_settings()
        .current_corners(Some(frame), units)
        .map_err(|error| error.to_string())?
        .ok_or_else(|| format!("work plane {:?}: expected display corners", feature.id()))?;
    plane_corners(frame, corners, units.working_scale())
}

fn plane_corners(
    frame: SketchPlane,
    corners: [[f64; 3]; 2],
    scale: f64,
) -> Result<[[f64; 3]; 4], String> {
    let [first, second] = local_diagonal(frame, corners, scale)?;
    let points = [first, [second[0], first[1]], second, [first[0], second[1]]];
    let mut result = [[0.; 3]; 4];
    for (out, point) in result.iter_mut().zip(points) {
        *out = frame.to_model(point).map_err(|error| error.to_string())?;
    }
    Ok(result)
}

fn local_diagonal(
    frame: SketchPlane,
    corners: [[f64; 3]; 2],
    scale: f64,
) -> Result<[[f64; 2]; 2], String> {
    if !scale.is_finite() || scale <= 0. {
        return Err(format!(
            "work-plane scale {scale:?}: expected a finite positive working scale"
        ));
    }
    let [first, second] = corners.map(|point| point.map(|coordinate| coordinate / scale));
    let first = frame.to_sketch(first).map_err(|error| error.to_string())?;
    let second = frame.to_sketch(second).map_err(|error| error.to_string())?;
    if first[0] == second[0] || first[1] == second[1] {
        return Err(format!(
            "work-plane corners {corners:?}: expected a nondegenerate rectangle"
        ));
    }
    Ok([first, second])
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::feature::{AuthoredFeature, work_geometry::WorkGeometryDefinition};
    use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};

    fn snapshot() -> DocumentSnapshot {
        DocumentSnapshot {
            assembly: None,
            drawing: None,
            model_states: None,
            body_names: Default::default(),
            design_views: Vec::new(),
            part_design_views: None,
            work_geometry: Default::default(),
            assets: Default::default(),
            sketch_settings: None,
            sketches: Default::default(),
            attributes: Default::default(),
            attachments: Default::default(),
            interests: Default::default(),
            resources: Default::default(),
            parameters: ParameterCollection::new(UnitsOfMeasure::default()),
            properties: Default::default(),
        }
    }

    #[test]
    fn corners_use_the_plane_frame_and_working_unit_scale() {
        let frame = SketchPlane::xz();
        let corners = plane_corners(frame, [[-5., 0., -5.], [5., 0., 5.]], 1.).unwrap();
        assert_eq!(
            corners,
            [[-5., 0., -5.], [5., 0., -5.], [5., 0., 5.], [-5., 0., 5.]]
        );
        assert!(plane_corners(frame, [[0.; 3]; 2], 1.).is_err());
        assert!(plane_corners(frame, [[0.; 3]; 2], 0.).is_err());
    }

    #[test]
    fn invalid_display_is_local_to_the_affected_plane() {
        let mut snapshot = snapshot();
        for id in 1..=2 {
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(
                    FeatureId(id),
                    "Plane",
                    WorkGeometryDefinition::Fixed(SketchPlane::xy()),
                ))
                .unwrap();
        }
        let settings = snapshot
            .work_geometry
            .get(FeatureId(2))
            .unwrap()
            .plane_display_settings()
            .edited(None, None, Some([[0.; 3]; 2]))
            .unwrap();
        snapshot
            .work_geometry
            .set_plane_display(FeatureId(2), settings)
            .unwrap();
        let evaluated = snapshot
            .work_geometry
            .evaluate_work_geometry(&snapshot.parameters)
            .unwrap();
        let result = planes_for_snapshot(
            &snapshot,
            &evaluated,
            None,
            &ThemeDefinition::dark().unwrap(),
        )
        .unwrap();
        assert_eq!(result.planes.len(), 1);
        assert_eq!(result.declines.len(), 1);
        assert_eq!(result.declines[0].0, FeatureId(2));
        assert!(
            result.declines[0]
                .1
                .contains("expected a nondegenerate rectangle")
        );
    }

    #[test]
    fn visible_planes_use_evaluation_and_selected_theme_color() {
        let mut snapshot = snapshot();
        for id in 1..=2 {
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(
                    FeatureId(id),
                    "Plane",
                    WorkGeometryDefinition::Fixed(SketchPlane::xy()),
                ))
                .unwrap();
        }
        snapshot
            .work_geometry
            .set_datum_visible(FeatureId(2), false)
            .unwrap();
        let evaluated = snapshot
            .work_geometry
            .evaluate_work_geometry(&snapshot.parameters)
            .unwrap();
        let theme = ThemeDefinition::dark().unwrap();
        let overlays =
            planes_for_snapshot(&snapshot, &evaluated, Some(FeatureId(1)), &theme).unwrap();
        assert!(overlays.declines.is_empty());
        let overlays = overlays.planes;
        assert_eq!(overlays.len(), 1);
        assert_eq!(
            overlays[0].border,
            theme.color(&oblikovati_api::types::TOKEN_PLANE_SELECTED)
        );
        assert_eq!(
            overlays[0].fill,
            theme.color(&oblikovati_api::types::TOKEN_PLANE_FILL)
        );
    }
}
