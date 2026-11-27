// SPDX-License-Identifier: GPL-2.0-only
//! Collect exact sketch identities from viewport picks in a Loft draft.
use super::*;
use oblikovati_automation::PartUpdateAccess;
use oblikovati_gui::{GuiLoftAction, GuiLoftElement, GuiLoftPathMode, GuiLoftSection};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_loft::{SketchLoftOutput, SketchLoftSection},
    },
    history::{DocumentSnapshot, ResolvedDocumentSketch},
};

pub(super) struct LoftSelection {
    document: u64,
    generation: u64,
    feature: Option<FeatureId>,
    path_mode: GuiLoftPathMode,
    output: SketchLoftOutput,
    sections: Vec<GuiLoftSection>,
    preview: Option<Result<(), String>>,
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
            GuiLoftAction::Open | GuiLoftAction::OpenEdit(_) => Ok(false),
            GuiLoftAction::Clear => {
                if self.loft_selection.is_none() {
                    return Ok(false);
                }
                self.clear_loft_preview();
                self.loft_selection = None;
                Ok(true)
            }
            GuiLoftAction::SetPath { document, mode } => self.choose_loft_path(document, mode),
            GuiLoftAction::SetOutput { document, output } => {
                self.choose_loft_output(document, output)
            }
            GuiLoftAction::Commit(document) => self.commit_loft(document),
            GuiLoftAction::Begin { document, feature } => self.begin_loft(document, feature),
        }
    }

    fn begin_loft(&mut self, document: u64, feature: Option<FeatureId>) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let (sections, output) = feature
            .map(|id| loft_sources(session.snapshot(), id))
            .transpose()?
            .unwrap_or_default();
        self.clear_extrusion_preview();
        let preview = feature.map(|id| {
            self.preview_selected_blend(document, |_, _| Ok(id))
                .map(|_| ())
        });
        self.loft_selection = Some(LoftSelection {
            document,
            generation,
            feature,
            path_mode: GuiLoftPathMode::Rails,
            output,
            sections,
            preview,
        });
        Ok(true)
    }

    fn choose_loft_path(&mut self, document: u64, mode: GuiLoftPathMode) -> Result<bool, String> {
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let Some(draft) = self.loft_selection.as_ref().filter(|draft| {
            draft.document == document && draft.generation == generation && draft.path_mode != mode
        }) else {
            return Ok(false);
        };
        let feature = draft.feature;
        let output = draft.output;
        let sections = circle_sections(&draft.sections);
        self.clear_loft_preview();
        let preview = match (mode, feature, sections) {
            (GuiLoftPathMode::Rails, Some(id), _) => Some(
                self.preview_selected_blend(document, |_, _| Ok(id))
                    .map(|_| ()),
            ),
            (GuiLoftPathMode::Rails, None, Some(sections)) => {
                Some(self.preview_loft(document, sections, output))
            }
            _ => None,
        };
        if let Some(draft) = &mut self.loft_selection {
            draft.path_mode = mode;
            draft.preview = preview;
        }
        Ok(true)
    }

    fn choose_loft_output(
        &mut self,
        document: u64,
        output: SketchLoftOutput,
    ) -> Result<bool, String> {
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let Some(draft) = self.loft_selection.as_ref().filter(|draft| {
            draft.document == document
                && draft.generation == generation
                && draft.feature.is_none()
                && draft.output != output
        }) else {
            return Ok(false);
        };
        let sections = (draft.path_mode == GuiLoftPathMode::Rails)
            .then(|| circle_sections(&draft.sections))
            .flatten();
        self.clear_loft_preview();
        let preview = sections.map(|sections| self.preview_loft(document, sections, output));
        if let Some(draft) = &mut self.loft_selection {
            draft.output = output;
            draft.preview = preview;
        }
        Ok(true)
    }

    pub(super) fn gui_loft_sections(&self, document: u64, generation: u64) -> Vec<GuiLoftSection> {
        self.loft_selection
            .as_ref()
            .filter(|draft| draft.document == document && draft.generation == generation)
            .map_or_else(Vec::new, |draft| draft.sections.clone())
    }

    pub(super) fn gui_loft_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<Result<(), String>> {
        let draft = self.loft_selection.as_ref()?;
        if draft.document != document || draft.generation != generation {
            return None;
        }
        draft.preview.clone()
    }

    fn clear_loft_preview(&mut self) {
        if self
            .loft_selection
            .as_ref()
            .and_then(|draft| draft.preview.as_ref())
            .is_some_and(Result::is_ok)
        {
            self.clear_extrusion_preview();
        }
    }

    fn commit_loft(&mut self, document: u64) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let draft = self
            .loft_selection
            .as_ref()
            .filter(|draft| {
                draft.document == document
                    && draft.feature.is_none()
                    && draft.path_mode == GuiLoftPathMode::Rails
                    && draft.generation == session.generation()
                    && draft.preview.as_ref().is_some_and(Result::is_ok)
            })
            .ok_or("Loft requires a valid preview in the current Part.")?;
        let sections =
            circle_sections(&draft.sections).ok_or("Loft requires two circular sections.")?;
        let prepared =
            oblikovati_automation::prepare_loft(session.snapshot(), sections, draft.output)
                .map_err(|cause| cause.to_string())?;
        oblikovati_automation::edit_part_with_checkpoints(
            &mut self.files,
            &mut self.part_scenes,
            document,
            "Loft",
            |candidate, ids| prepared.create(candidate, ids),
        )
        .map_err(|cause| cause.to_string())?;
        self.clear_loft_preview();
        self.loft_selection = None;
        Ok(true)
    }

    fn preview_loft(
        &mut self,
        document: u64,
        sections: [SketchLoftSection; 2],
        output: SketchLoftOutput,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let prepared = oblikovati_automation::prepare_loft(session.snapshot(), sections, output)
            .map_err(|cause| cause.to_string())?;
        self.preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))?;
        Ok(())
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
        if self.loft_selection.as_ref().is_some_and(|draft| {
            draft.feature.is_some() || draft.path_mode != GuiLoftPathMode::Rails
        }) {
            return Ok(Some(false));
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
        let output = draft.output;
        let sections = circle_sections(&draft.sections);
        self.clear_loft_preview();
        let preview = sections.map(|sections| self.preview_loft(document, sections, output));
        if let Some(draft) = &mut self.loft_selection {
            draft.preview = preview;
        }
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

pub(super) fn loft_outputs(
    snapshot: &DocumentSnapshot,
) -> std::collections::BTreeMap<FeatureId, SketchLoftOutput> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| match feature.definition() {
            PartFeatureDefinition::SketchLoft(definition) => {
                Some((feature.id(), definition.recipe().output))
            }
            _ => None,
        })
        .collect()
}

fn loft_sources(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<(Vec<GuiLoftSection>, SketchLoftOutput), String> {
    let authored = snapshot
        .work_geometry
        .get(id)
        .map_err(|cause| cause.to_string())?;
    let PartFeatureDefinition::SketchLoft(definition) = authored.definition() else {
        return Err(format!("feature {id:?}: expected a Loft"));
    };
    let sections = definition
        .recipe()
        .sections
        .into_iter()
        .map(|section| {
            let sketch = snapshot
                .sketches
                .get(section.sketch)
                .map_err(|cause| cause.to_string())?;
            Ok(GuiLoftSection {
                sketch: section.sketch,
                name: sketch.name().to_owned(),
                element: GuiLoftElement::Profile(section.circle),
            })
        })
        .collect::<Result<_, String>>()?;
    Ok((sections, definition.recipe().output))
}

fn circle_sections(sections: &[GuiLoftSection]) -> Option<[SketchLoftSection; 2]> {
    let [first, last] = sections else {
        return None;
    };
    let section = |picked: &GuiLoftSection| match picked.element {
        GuiLoftElement::Profile(circle) => Some(SketchLoftSection {
            sketch: picked.sketch,
            circle,
        }),
        GuiLoftElement::Point(_) => None,
    };
    Some([section(first)?, section(last)?])
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
