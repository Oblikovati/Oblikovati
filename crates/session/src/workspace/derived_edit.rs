// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::history::RecipeReservations;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    pub(crate) fn run_exporting_source<R>(
        &mut self,
        target: u64,
        source: u64,
        names: &[String],
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
    ) -> Result<R, String> {
        self.session(target).map_err(|error| error.to_string())?;
        self.session(source).map_err(|error| error.to_string())?;
        if target == source {
            return Err(format!(
                "source document {source}: expected a different target from {target}"
            ));
        }
        let mut entries = self
            .entries
            .iter_mut()
            .filter(|(id, _)| **id == source || **id == target);
        let (_, first) = entries.next().expect("validated first owner");
        let (_, second) = entries.next().expect("validated second owner");
        let (target, source) = if target < source {
            (&mut first.session, &mut second.session)
        } else {
            (&mut second.session, &mut first.session)
        };
        run_with_export(target, source, names, command)
    }
}

fn run_with_export<O: TransactionObserver, R>(
    target: &mut DocumentSession<O>,
    source: &mut DocumentSession<O>,
    names: &[String],
    command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
) -> Result<R, String> {
    let prepared = source
        .history
        .prepare_refresh(|snapshot| export_source(snapshot, names))
        .map_err(|error| error.to_string())?;
    let result = target
        .run("Edit Derived Parameters", command)
        .map_err(|error| error.to_string())?;
    if prepared.publish() {
        source
            .observer
            .changed(source.document, source.generation());
    }
    Ok(result)
}

fn export_source(snapshot: &mut DocumentSnapshot, names: &[String]) -> Result<(), String> {
    for name in names {
        let id = snapshot.parameters.find(name).ok_or_else(|| {
            format!("source parameter {name:?}: expected an existing numeric user parameter")
        })?;
        let parameter = snapshot.parameters.get(id).expect("resolved parameter");
        if parameter.kind != oblikovati_api::types::USER_PARAM
            || parameter.current_value().is_none()
        {
            return Err(format!(
                "source parameter {name:?}: expected a healthy numeric user parameter"
            ));
        }
        let mut annotations = parameter.annotations.clone();
        annotations.exposed_as_property = true;
        snapshot
            .parameters
            .set_annotations(id, annotations)
            .map_err(|error| error.to_string())?;
    }
    snapshot
        .properties
        .sync_exposed_parameters(&snapshot.parameters)
        .map_err(|error| error.to_string())
}
