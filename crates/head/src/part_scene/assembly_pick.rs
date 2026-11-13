// SPDX-License-Identifier: GPL-2.0-only
//! Pick assembly occurrences against component B-reps, not display meshes.
use super::*;
use oblikovati_gui::GuiOccurrencePath;

struct OccurrenceHit {
    path: GuiOccurrencePath,
    reach: f64,
    ambiguous: bool,
}

impl DesktopPartScenes {
    pub(crate) fn occurrence_on_line<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<Option<GuiOccurrencePath>, String> {
        let mut nearest = None;
        self.pick_occurrences(files, document, line, &[], &mut nearest)?;
        if let Some(hit) = &nearest
            && hit.ambiguous
        {
            return Err(format!(
                "occurrence hit {}: expected an unambiguous nearest placement",
                hit.reach
            ));
        }
        Ok(nearest.map(|hit: OccurrenceHit| hit.path))
    }

    fn pick_occurrences<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        owner: u64,
        line: [[f64; 3]; 2],
        parent: &[(u64, u64, usize)],
        nearest: &mut Option<OccurrenceHit>,
    ) -> Result<(), String> {
        if parent.len() > super::assembly::MAX_ASSEMBLY_DEPTH {
            return Err(format!(
                "assembly document {owner}: expected at most 128 nested levels"
            ));
        }
        let session = files
            .workspace()
            .session(owner)
            .map_err(|error| error.to_string())?;
        let generation = session.generation();
        let rows = session
            .snapshot()
            .assembly
            .clone()
            .ok_or_else(|| format!("document {owner}: expected an assembly"))?;
        for (ordinal, row) in rows
            .iter()
            .enumerate()
            .filter(|(_, row)| !row.suppressed && !row.virtual_)
        {
            let Some(component) = files.document_reference(owner, &row.component) else {
                continue;
            };
            let mut path = parent.to_vec();
            path.push((owner, generation, ordinal));
            let local = oblikovati_martensite::line_in_occurrence(&row.placement()?, line)
                .map_err(|error| format!("occurrence {:?}: {error}", row.name))?;
            self.pick_component(files, component, local, &path, nearest)?;
        }
        Ok(())
    }

    fn pick_component<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
        path: &[(u64, u64, usize)],
        nearest: &mut Option<OccurrenceHit>,
    ) -> Result<(), String> {
        let nested = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?
            .snapshot()
            .assembly
            .is_some();
        if nested {
            return self.pick_occurrences(files, document, line, path, nearest);
        }
        self.ensure_current(files, document)?;
        self.pick_part(files, document, line, path, nearest)
    }

    fn pick_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
        path: &[(u64, u64, usize)],
        nearest: &mut Option<OccurrenceHit>,
    ) -> Result<(), String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let guid = files
            .workspace()
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let evaluated = self.current(session)?;
        let scale = line[1][0].hypot(line[1][1]).hypot(line[1][2]);
        for body in evaluated.ordered_bodies() {
            let key =
                bodies::body_reference_text(body.body_reference_key(session.history(), guid)?);
            if !session.body_visible(&key) {
                continue;
            }
            if let Some((_, reach)) = body.face_on_line(line)? {
                let distance = reach / scale;
                if !distance.is_finite() {
                    return Err(format!(
                        "occurrence {path:?} hit {distance}: expected a finite line parameter"
                    ));
                }
                if let Some(hit) = nearest.as_mut()
                    && distance == hit.reach
                    && hit.path != path
                {
                    hit.ambiguous = true;
                }
                if nearest.as_ref().is_none_or(|hit| distance < hit.reach) {
                    *nearest = Some(OccurrenceHit {
                        path: path.to_vec(),
                        reach: distance,
                        ambiguous: false,
                    });
                }
            }
        }
        Ok(())
    }
}
