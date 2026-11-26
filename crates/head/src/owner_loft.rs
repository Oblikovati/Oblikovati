// SPDX-License-Identifier: GPL-2.0-only
//! Collect exact sketch identities from viewport picks in a Loft draft.
use super::*;
use oblikovati_automation::PartUpdateAccess;
use oblikovati_gui::{GuiLoftAction, GuiLoftElement, GuiLoftSection};
use oblikovati_model::history::ResolvedDocumentSketch;

pub(super) struct LoftSelection {
    document: u64,
    generation: u64,
    sections: Vec<GuiLoftSection>,
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
    pub(super) fn loft_command(&mut self, action: GuiLoftAction) -> Result<bool, String> {
        match action {
            GuiLoftAction::Open => Ok(false),
            GuiLoftAction::Clear => Ok(self.loft_selection.take().is_some()),
            GuiLoftAction::Begin(document) => {
                self.require_browser_document(document)?;
                let session = self
                    .files
                    .workspace()
                    .session(document)
                    .map_err(|cause| cause.to_string())?;
                self.loft_selection = Some(LoftSelection {
                    document,
                    generation: session.generation(),
                    sections: Vec::new(),
                });
                Ok(true)
            }
        }
    }

    pub(super) fn gui_loft_sections(&self, document: u64, generation: u64) -> Vec<GuiLoftSection> {
        self.loft_selection
            .as_ref()
            .filter(|draft| draft.document == document && draft.generation == generation)
            .map_or_else(Vec::new, |draft| draft.sections.clone())
    }

    pub(super) fn pick_loft_section(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if !self.loft_selection.as_ref().is_some_and(|draft| {
            draft.document == document && draft.generation == session.generation()
        }) {
            return Ok(None);
        }
        let Some(tolerance) = tolerance else {
            return Ok(Some(false));
        };
        let Some(section) = self.loft_section_at(document, ray, tolerance)? else {
            return Ok(Some(false));
        };
        let Some(draft) = &mut self.loft_selection else {
            return Ok(None);
        };
        if draft.sections.contains(&section) {
            return Ok(Some(false));
        }
        draft.sections.push(section);
        Ok(Some(true))
    }

    fn loft_section_at(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Option<GuiLoftSection>, String> {
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let sources = snapshot
            .sketches
            .sketches()
            .iter()
            .filter(|sketch| sketch.visible())
            .filter_map(|sketch| {
                snapshot
                    .require_converged_resolved_sketch_with_datums(sketch.id(), &datums)
                    .ok()
            });
        let mut closest: Option<LoftHit> = None;
        for source in sources {
            if let Some(hit) = section_hit(source, ray, tolerance)?
                && closest.as_ref().is_none_or(|current| hit.precedes(current))
            {
                closest = Some(hit);
            }
        }
        Ok(closest.map(|hit| hit.section))
    }
}

struct LoftHit {
    depth: f64,
    error: f64,
    section: GuiLoftSection,
}

impl LoftHit {
    fn precedes(&self, other: &Self) -> bool {
        self.error
            .total_cmp(&other.error)
            .then_with(|| self.depth.total_cmp(&other.depth))
            .is_lt()
    }
}

fn section_hit(
    source: ResolvedDocumentSketch<'_>,
    [origin, direction]: [[f64; 3]; 2],
    tolerance: f64,
) -> Result<Option<LoftHit>, String> {
    let Some(local) = oblikovati_martensite::sketch_projection(
        source.plane,
        origin,
        direction,
        oblikovati_martensite::SKETCH_PROJECTION_MIN_NORMAL_COSINE,
    )?
    else {
        return Ok(None);
    };
    let world = source
        .plane
        .to_model(local)
        .map_err(|cause| cause.to_string())?;
    let depth: f64 = world
        .into_iter()
        .zip(origin)
        .zip(direction)
        .map(|((point, eye), heading)| (point - eye) * heading)
        .sum();
    let candidate = section_element(source.sketch, local, tolerance);
    Ok(candidate
        .filter(|_| depth >= 0.)
        .map(|(element, error)| LoftHit {
            depth,
            error,
            section: GuiLoftSection {
                sketch: source.sketch.id(),
                name: source.sketch.name().into(),
                element,
            },
        }))
}

fn section_element(
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    hit: [f64; 2],
    tolerance: f64,
) -> Option<(GuiLoftElement, f64)> {
    let geometry = sketch.geometry();
    let distance = |point: [f64; 2]| (hit[0] - point[0]).hypot(hit[1] - point[1]);
    let points = geometry
        .points()
        .iter()
        .filter(|point| {
            !sketch
                .point_projections()
                .iter()
                .any(|projected| projected.anchor == point.id)
        })
        .map(|point| (GuiLoftElement::Point(point.id), distance(point.position)));
    let circles = geometry
        .circles()
        .iter()
        .filter(|circle| matches!(geometry.is_construction(circle.id), Ok(false)))
        .filter_map(|circle| {
            let center = geometry.point(circle.center).ok()?.position;
            Some((
                GuiLoftElement::Profile(circle.id),
                (distance(center) - circle.radius).abs(),
            ))
        });
    points
        .chain(circles)
        .filter(|(_, error)| *error <= tolerance)
        .min_by(|(_, first), (_, second)| first.total_cmp(second))
}
