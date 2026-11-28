// SPDX-License-Identifier: GPL-2.0-only
//! Parameter edits, undo/redo, and BRep-to-WebGPU scene refresh for `WebSession`.

use super::{DEFAULT_BODY_COLOR, WebSession};
use oblikovati_automation::PartUpdateAccess;
use oblikovati_viewport::{ViewportNavigation, ViewportScene};

impl WebSession {
    pub(super) fn edit_parameter(&mut self, name: &str, expression: &str) -> Result<(), String> {
        let id = self
            .workspace
            .workspace()
            .active_document()
            .ok_or_else(|| "set parameter: expected an active document".to_owned())?;
        let param_id = {
            let session = self
                .workspace
                .workspace()
                .session(id)
                .map_err(|err| err.to_string())?;
            session
                .snapshot()
                .parameters
                .find(name)
                .ok_or_else(|| format!("parameter {name:?}: expected an existing parameter"))?
        };
        let label = format!("Set {name} = {expression}");
        let expr = expression.to_owned();
        let mut commands = self.workspace.commands(id).map_err(|err| err.to_string())?;
        commands
            .run(&label, |snapshot, _ids| -> Result<(), String> {
                snapshot
                    .parameters
                    .set_expression(param_id, &expr)
                    .map_err(|err| err.to_string())?;
                snapshot.sketches = snapshot
                    .sketches
                    .solve_pending(&snapshot.parameters, Default::default(), || false)
                    .candidate;
                Ok(())
            })
            .map_err(|err| err.to_string())?;
        self.status_message = label;
        self.refresh_active_scene(false)
    }

    pub(super) fn apply_undo(&mut self) -> Result<(), String> {
        let id = self
            .workspace
            .workspace()
            .active_document()
            .ok_or_else(|| "undo: expected an active document".to_owned())?;
        self.workspace
            .commands(id)
            .map_err(|err| err.to_string())?
            .undo()
            .map_err(|err| err.to_string())?;
        self.status_message = "Undid last edit".into();
        self.refresh_active_scene(false)
    }

    pub(super) fn apply_redo(&mut self) -> Result<(), String> {
        let id = self
            .workspace
            .workspace()
            .active_document()
            .ok_or_else(|| "redo: expected an active document".to_owned())?;
        self.workspace
            .commands(id)
            .map_err(|err| err.to_string())?
            .redo()
            .map_err(|err| err.to_string())?;
        self.status_message = "Redid edit".into();
        self.refresh_active_scene(false)
    }

    pub(super) fn refresh_active_scene(&mut self, fit_camera: bool) -> Result<(), String> {
        let Some(id) = self.workspace.workspace().active_document() else {
            let empty = ViewportScene {
                background: self.theme.scene_background(),
                triangles: Vec::new(),
            };
            self.viewport.submit(&empty)?;
            self.current_scene = Some(empty);
            self.visible_bodies = 0;
            return Ok(());
        };
        let _errors = self.replay.update_part(&mut self.workspace, id)?;
        let evaluated = self.replay.current_part(&self.workspace, id)?;
        let presentation = ViewportScene::from_part_evaluation(
            evaluated,
            oblikovati_drawing::DISPLAY_MESH,
            self.theme.scene_background(),
            DEFAULT_BODY_COLOR,
        )?;
        self.visible_bodies = presentation.visible_bodies;
        self.viewport.submit(&presentation.scene)?;
        if self.normal_debug {
            self.viewport
                .set_normal_debug(Some(&presentation.scene), true)?;
        }
        self.current_scene = Some(presentation.scene);
        if fit_camera && self.visible_bodies > 0 {
            let _ = self
                .viewport
                .navigate(ViewportNavigation::Home { margin: 0.14 });
        }
        Ok(())
    }
}
