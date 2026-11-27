// SPDX-License-Identifier: GPL-2.0-only
//! Resolve profile and path picks against the current Sweep draft and sketch source.
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
    pub(in crate::owner) fn sweep_profile_pick_sketch(&self) -> Option<DocumentSketchId> {
        self.sweep_profile_target(self.files.workspace().active_document()?)
            .map(|(_, sketch)| sketch)
    }

    pub(in crate::owner) fn pick_sweep_profile(
        &mut self,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<Option<bool>, String> {
        let Some(mut request) = self.current_sweep_request(document).cloned() else {
            return Ok(None);
        };
        if request.profile != GuiSweepProfile::Empty {
            return Ok(None);
        }
        let Some(sketch) = self.sweep_source(&request, false)? else {
            return Ok(Some(false));
        };
        let Some(boundary) = self.profile_on_view_line(document, sketch, line)? else {
            return Ok(Some(false));
        };
        request.profile = GuiSweepProfile::Picked(boundary);
        self.sweep_command(GuiSweepAction::Preview(request))
            .map(Some)
    }

    fn sweep_profile_target(&self, document: u64) -> Option<(&GuiSweepRequest, DocumentSketchId)> {
        let request = self.current_sweep_request(document)?;
        if request.profile != GuiSweepProfile::Empty {
            return None;
        }
        let session = self.files.workspace().session(document).ok()?;
        let sweep = sweep_definition(session.snapshot(), request.feature?).ok()?;
        Some((request, sweep.recipe().profile_sketch))
    }

    fn current_sweep_request(&self, document: u64) -> Option<&GuiSweepRequest> {
        let request = &self.sweep_preview.as_ref()?.request;
        let session = self.files.workspace().session(document).ok()?;
        (request.document == document && request.generation == session.generation())
            .then_some(request)
    }

    pub(in crate::owner) fn pick_sweep_path(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let Some(mut request) = self
            .current_sweep_request(document)
            .filter(|request| request.path_cleared)
            .cloned()
        else {
            return Ok(None);
        };
        let Some((sketch, tolerance)) = self.sweep_source(&request, true)?.zip(tolerance) else {
            return Ok(Some(false));
        };
        let [start, end] = self.sweep_path_endpoints(document, sketch)?;
        if !ray_hits_segment(ray, [start, end], tolerance) {
            return Ok(Some(false));
        }
        request.path_cleared = false;
        self.sweep_command(GuiSweepAction::Preview(request))
            .map(Some)
    }

    fn sweep_source(
        &mut self,
        request: &GuiSweepRequest,
        path: bool,
    ) -> Result<Option<DocumentSketchId>, String> {
        let document = request.document;
        if let Some(feature) = request.feature {
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            let recipe = sweep_definition(session.snapshot(), feature)?.recipe();
            return Ok(Some(if path {
                recipe.path_sketch
            } else {
                recipe.profile_sketch
            }));
        }
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("sweep inputs: expected a certified resolution policy")?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        Ok(
            automatic_sweep(snapshot, &datums, policy).and_then(|source| {
                let index = if path {
                    source.path_sketch_index
                } else {
                    source.sketch_index
                };
                let index = usize::try_from(index).ok()?;
                snapshot
                    .sketches
                    .sketches()
                    .get(index)
                    .map(|sketch| sketch.id())
            }),
        )
    }

    fn sweep_path_endpoints(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<[[f64; 3]; 2], String> {
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let source = session
            .snapshot()
            .require_converged_resolved_sketch_with_datums(sketch, &datums)
            .map_err(|cause| cause.to_string())?;
        path_segment(source)
    }

    fn profile_on_view_line(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        line: [[f64; 3]; 2],
    ) -> Result<Option<oblikovati_model::feature::sketch_prism::SketchPrismSelection>, String> {
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("sweep profile: expected a certified resolution policy")?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let source = session
            .snapshot()
            .require_converged_resolved_sketch_with_datums(sketch, &datums)
            .map_err(|cause| cause.to_string())?;
        oblikovati_martensite::pick_sketch_region(source, policy, line)
            .map_err(|cause| cause.to_string())
    }
}

fn path_segment(
    source: oblikovati_model::history::ResolvedDocumentSketch<'_>,
) -> Result<[[f64; 3]; 2], String> {
    let geometry = source.sketch.geometry();
    let mut straight = geometry
        .lines()
        .iter()
        .filter(|line| matches!(geometry.is_construction(line.id), Ok(false)));
    let (Some(line), None) = (straight.next(), straight.next()) else {
        return Err("Sweep path selection requires one straight curve".into());
    };
    let point = |entity| {
        let point = geometry.point(entity).map_err(|cause| cause.to_string())?;
        source
            .plane
            .to_model(point.position)
            .map_err(|cause| cause.to_string())
    };
    Ok([point(line.endpoints[0])?, point(line.endpoints[1])?])
}

pub(in crate::owner) fn ray_hits_segment(
    [origin, direction]: [[f64; 3]; 2],
    [start, end]: [[f64; 3]; 2],
    tolerance: f64,
) -> bool {
    let dot = |a: [f64; 3], b: [f64; 3]| (0..3).map(|i| a[i] * b[i]).sum::<f64>();
    let direction_squared = dot(direction, direction);
    if direction_squared <= 0. || !direction_squared.is_finite() {
        return false;
    }
    let screen_vector = |point: [f64; 3]| {
        let offset = std::array::from_fn(|i| point[i] - origin[i]);
        let along_ray = dot(offset, direction) / direction_squared;
        std::array::from_fn(|i| offset[i] - along_ray * direction[i])
    };
    let base = screen_vector(start);
    let target = screen_vector(end);
    let segment = std::array::from_fn(|i| target[i] - base[i]);
    let squared = dot(segment, segment);
    let fraction = if squared > 0. {
        (-dot(base, segment) / squared).clamp(0., 1.)
    } else {
        0.
    };
    let nearest = std::array::from_fn(|i| base[i] + fraction * segment[i]);
    dot(nearest, nearest) <= tolerance * tolerance
}
