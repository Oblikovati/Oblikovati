// SPDX-License-Identifier: GPL-2.0-only
//! Keep the selected Center Line attached to its authored sketch curve.
use super::*;
use crate::owner::owner_blend::owner_sweep::ray_hits_segment;

pub(super) struct CenterLinePick {
    pub(super) source: SketchLoftCenterLine,
    name: String,
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
    pub(in crate::owner) fn gui_loft_sections(
        &self,
        document: u64,
        generation: u64,
    ) -> Vec<GuiLoftSection> {
        self.loft_selection
            .as_ref()
            .filter(|draft| draft.document == document && draft.generation == generation)
            .map_or_else(Vec::new, |draft| draft.sections.clone())
    }

    pub(in crate::owner) fn gui_loft_preview(
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

    pub(in crate::owner) fn gui_loft_center_line(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<String> {
        let draft = self.loft_selection.as_ref().filter(|draft| {
            draft.document == document
                && draft.generation == generation
                && draft.path_mode != GuiLoftPathMode::Rails
        })?;
        draft.center_line.as_ref().map(|picked| picked.name.clone())
    }

    pub(super) fn arm_center_line(&mut self, document: u64) -> Result<bool, String> {
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let Some(draft) = &mut self.loft_selection else {
            return Ok(false);
        };
        if draft.document != document
            || draft.generation != generation
            || draft.feature.is_some()
            || draft.path_mode == GuiLoftPathMode::Rails
        {
            return Ok(false);
        }
        draft.pick_center_line = true;
        Ok(true)
    }

    pub(super) fn pick_center_line_at(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<bool, String> {
        let Some(tolerance) = tolerance else {
            return Ok(false);
        };
        let Some(picked) = self.find_center_line(document, ray, tolerance)? else {
            return Ok(false);
        };
        self.clear_loft_preview();
        let Some(draft) = &mut self.loft_selection else {
            return Ok(false);
        };
        draft.center_line = Some(picked);
        draft.pick_center_line = false;
        let output = draft.output;
        let input = circle_sections(&draft.sections).zip(draft.ready_path());
        let preview =
            input.map(|(sections, path)| self.preview_loft(document, sections, output, path));
        if let Some(draft) = &mut self.loft_selection {
            draft.preview = preview;
        }
        Ok(true)
    }

    fn find_center_line(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Option<CenterLinePick>, String> {
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let mut hits = snapshot
            .sketches
            .sketches()
            .iter()
            .filter(|sketch| sketch.visible())
            .filter_map(|sketch| {
                snapshot
                    .require_converged_resolved_sketch_with_datums(sketch.id(), &datums)
                    .ok()
            })
            .filter_map(|source| line_hit(&source, ray, tolerance));
        Ok(match (hits.next(), hits.next()) {
            (Some(picked), None) => Some(picked),
            _ => None,
        })
    }
}

fn line_hit(
    source: &ResolvedDocumentSketch<'_>,
    ray: [[f64; 3]; 2],
    tolerance: f64,
) -> Option<CenterLinePick> {
    let geometry = source.sketch.geometry();
    let mut lines = geometry
        .lines()
        .iter()
        .filter(|line| geometry.is_construction(line.id) == Ok(false));
    let (Some(line), None) = (lines.next(), lines.next()) else {
        return None;
    };
    let point = |endpoint| {
        source
            .plane
            .to_model(geometry.point(endpoint).ok()?.position)
            .ok()
    };
    let ends = [point(line.endpoints[0])?, point(line.endpoints[1])?];
    ray_hits_segment(ray, ends, tolerance).then(|| CenterLinePick {
        source: SketchLoftCenterLine {
            sketch: source.sketch.id(),
            line: line.id,
        },
        name: source.sketch.name().to_owned(),
    })
}

pub(in crate::owner) fn loft_edits(
    snapshot: &DocumentSnapshot,
) -> std::collections::BTreeMap<FeatureId, GuiLoftEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| match feature.definition() {
            PartFeatureDefinition::SketchLoft(definition) => {
                let recipe = definition.recipe();
                let path_mode = path_mode(recipe.center_line.is_some(), recipe.placed.is_some());
                Some((
                    feature.id(),
                    GuiLoftEditSeed {
                        output: recipe.output,
                        path_mode,
                        takeoff: recipe.takeoff,
                    },
                ))
            }
            _ => None,
        })
        .collect()
}

#[derive(Default)]
pub(super) struct LoftDraftSources {
    pub(super) sections: Vec<GuiLoftSection>,
    pub(super) output: SketchLoftOutput,
    pub(super) center_line: Option<CenterLinePick>,
    pub(super) placed: Option<SketchLoftPlacedSection>,
    pub(super) takeoff: Option<LoftTakeoffCondition>,
}

impl LoftDraftSources {
    pub(super) fn mode(&self) -> GuiLoftPathMode {
        path_mode(self.center_line.is_some(), self.placed.is_some())
    }
}

fn path_mode(center_line: bool, placed: bool) -> GuiLoftPathMode {
    match (center_line, placed) {
        (_, true) => GuiLoftPathMode::PlacedSections,
        (true, false) => GuiLoftPathMode::CenterLine,
        (false, false) => GuiLoftPathMode::Rails,
    }
}

pub(super) fn loft_sources(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<LoftDraftSources, String> {
    let authored = snapshot
        .work_geometry
        .get(id)
        .map_err(|cause| cause.to_string())?;
    let PartFeatureDefinition::SketchLoft(definition) = authored.definition() else {
        return Err(format!("feature {id:?}: expected a Loft"));
    };
    let recipe = definition.recipe();
    let name = |sketch| {
        snapshot
            .sketches
            .get(sketch)
            .map(|source| source.name().to_owned())
            .map_err(|cause| cause.to_string())
    };
    let sections = recipe
        .sections
        .into_iter()
        .map(|section| {
            Ok(GuiLoftSection {
                sketch: section.sketch,
                name: name(section.sketch)?,
                element: GuiLoftElement::Profile(section.circle),
            })
        })
        .collect::<Result<_, String>>()?;
    let center_line = recipe
        .center_line
        .map(|source| {
            Ok::<_, String>(CenterLinePick {
                name: name(source.sketch)?,
                source,
            })
        })
        .transpose()?;
    Ok(LoftDraftSources {
        sections,
        output: recipe.output,
        center_line,
        placed: recipe.placed,
        takeoff: recipe.takeoff,
    })
}
