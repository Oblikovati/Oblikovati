// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    json::JsonValue,
    wire::{
        AddConstraintArgs, AddDimensionArgs, AddFeatureArgs, CreateDocumentArgs, CreateSketchArgs,
        ParameterSetArgs,
    },
};
use oblikovati_model::document_sketch::DocumentSketchId;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn apply_cube_action(
        &mut self,
        action: oblikovati_gui::GuiShellAction,
    ) -> Result<bool, String> {
        match action {
            oblikovati_gui::GuiShellAction::NewPart => self.new_cube_part(),
            oblikovati_gui::GuiShellAction::SketchXY => self.start_cube_sketch(),
            oblikovati_gui::GuiShellAction::Rectangle => self.start_cube_rectangle(),
            oblikovati_gui::GuiShellAction::ExitSketch => self.finish_cube_sketch(),
            oblikovati_gui::GuiShellAction::Extrude => self.extrude_cube_sketch(),
            _ => Ok(false),
        }
    }

    pub(super) fn complete_cube_rectangle(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
        report: &oblikovati_model::history::SketchPlacementCreationReport,
    ) -> Result<(), String> {
        if self.cube_rectangle != Some((document, sketch)) {
            return Ok(());
        }
        let points = ids(&report.points)?;
        let lines = ids(&report.members)?;
        constrain_cube_rectangle(self, document, sketch, &points, &lines)?;
        self.cube_rectangle = None;
        self.command_notice = Some(DesktopCommandNotice::PlacementPrompt {
            message: "Cube rectangle constrained. Click Extrude.".into(),
        });
        Ok(())
    }

    fn new_cube_part(&mut self) -> Result<bool, String> {
        Client::new(self.document_host())
            .documents()
            .create(&CreateDocumentArgs {
                type_: "part".into(),
                name: "Cube".into(),
                ..Default::default()
            })
            .map_err(|cause| cause.to_string())?;
        self.placement = None;
        self.sketch_edit = None;
        self.cube_rectangle = None;
        Ok(true)
    }

    fn start_cube_sketch(&mut self) -> Result<bool, String> {
        let document = self.active_part_document()?;
        Client::new(self.document_host())
            .sketch()
            .create(&CreateSketchArgs {
                plane: "XY".into(),
                ..Default::default()
            })
            .map_err(|cause| cause.to_string())?;
        let sketch = self.latest_sketch(document)?;
        self.add_cube_size_if_missing(document)?;
        self.begin_sketch_edit(document, sketch)?;
        Ok(true)
    }

    fn start_cube_rectangle(&mut self) -> Result<bool, String> {
        let edit = self
            .active_sketch_edit()
            .ok_or_else(|| "Rectangle: expected an active XY sketch".to_string())?;
        self.start_sketch_placement(
            edit.document,
            edit.sketch,
            crate::SketchPlacementTool::Rectangle,
        )?;
        self.cube_rectangle = Some((edit.document, edit.sketch));
        Ok(true)
    }

    fn extrude_cube_sketch(&mut self) -> Result<bool, String> {
        let (document, sketch) = self.extrude_sketch()?;
        let args = serde_json::to_value(oblikovati_api::wire::featureargs::Extrude {
            sketch_index: self.sketch_index(document, sketch)?,
            profile_index: 0,
            distance: "size".into(),
            operation: "new".into(),
            ..Default::default()
        })
        .map_err(|cause| cause.to_string())?;
        Client::new(self.document_host())
            .features()
            .add(&AddFeatureArgs {
                kind: "extrude".into(),
                args: JsonValue(Some(args)),
            })
            .map_err(|cause| cause.to_string())?;
        if self.active_sketch_edit().is_some() {
            self.finish_sketch_edit()?;
        }
        Ok(true)
    }

    fn finish_cube_sketch(&mut self) -> Result<bool, String> {
        self.finish_sketch_edit()?;
        Ok(true)
    }

    fn extrude_sketch(&self) -> Result<(u64, DocumentSketchId), String> {
        if let Some(edit) = self.active_sketch_edit() {
            return Ok((edit.document, edit.sketch));
        }
        let document = self.active_part_document()?;
        Ok((document, self.latest_sketch(document)?))
    }

    fn active_part_document(&self) -> Result<u64, String> {
        let document = self
            .files
            .workspace()
            .active_document()
            .ok_or_else(|| "New Sketch: expected an active part document".to_string())?;
        self.require_edit_document(document)?;
        Ok(document)
    }

    fn sketch_index(&self, document: u64, sketch: DocumentSketchId) -> Result<i64, String> {
        let sketches = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .sketches
            .sketches();
        sketches
            .iter()
            .position(|entry| entry.id() == sketch)
            .and_then(|index| i64::try_from(index).ok())
            .ok_or_else(|| {
                format!(
                    "sketch {} in document {document}: expected an ordered sketch index",
                    sketch.0
                )
            })
    }

    fn latest_sketch(&self, document: u64) -> Result<DocumentSketchId, String> {
        self.files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .sketches
            .sketches()
            .last()
            .map(|sketch| sketch.id())
            .ok_or_else(|| format!("document {document}: expected the new XY sketch"))
    }

    fn add_cube_size_if_missing(&mut self, document: u64) -> Result<(), String> {
        let exists = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .parameters
            .find("size")
            .is_some();
        if exists {
            return Ok(());
        }
        Client::new(self.document_host())
            .parameters()
            .add(&ParameterSetArgs {
                name: "size".into(),
                expression: "10 mm".into(),
                ..Default::default()
            })
            .map_err(|cause| cause.to_string())?;
        Ok(())
    }
}

fn ids(ids: &[oblikovati_model::sketch::SketchEntityId]) -> Result<Vec<u64>, String> {
    if ids.len() == 4 {
        return Ok(ids.iter().map(|id| id.0).collect());
    }
    Err(format!(
        "rectangle entities {ids:?}: expected four authored entities"
    ))
}

fn constrain_cube_rectangle<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
>(
    owner: &mut DesktopOwner<O, W, F, G, R, S>,
    document: u64,
    sketch: DocumentSketchId,
    points: &[u64],
    lines: &[u64],
) -> Result<(), String> {
    let sketch_index = owner.sketch_index(document, sketch)?;
    for (kind, entities) in [
        ("horizontal", vec![lines[0]]),
        ("vertical", vec![lines[1]]),
        ("horizontal", vec![lines[2]]),
        ("vertical", vec![lines[3]]),
        ("fix", vec![points[0]]),
    ] {
        Client::new(owner.document_host())
            .constrain()
            .add(&AddConstraintArgs {
                sketch_index,
                kind: kind.into(),
                entities: Some(entities),
                ..Default::default()
            })
            .map_err(|cause| cause.to_string())?;
    }
    for (entities, orientation) in [(&points[0..2], "horizontal"), (&points[1..3], "vertical")] {
        Client::new(owner.document_host())
            .dimension()
            .add(&AddDimensionArgs {
                sketch_index,
                kind: "distance".into(),
                entities: Some(entities.to_vec()),
                expression: "size".into(),
                orientation: orientation.into(),
                ..Default::default()
            })
            .map_err(|cause| cause.to_string())?;
    }
    Ok(())
}
