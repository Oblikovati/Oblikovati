// SPDX-License-Identifier: GPL-2.0-only
//! Commit a source, its named replay and linked readers as one prepared edit (#4122).
use super::*;
use oblikovati_model::history::{
    PreparedChoice, PreparedDependentPublication, PreparedDependentRefresh, PreparedValue,
    RecipeReservations,
};

type UnusedEdit =
    for<'a> fn(&mut DocumentSnapshot, &mut RecipeReservations<'a>) -> Result<(), String>;
enum NamedAction<C, R> {
    Edit(C),
    Navigate(oblikovati_model::history::PreparedNavigation, R),
}

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    /// Verify an unlinked part's candidate replay without changing its session.
    /// Example: `workspace.verify_unlinked_named_candidate(id, &candidate)?`.
    pub fn verify_unlinked_named_candidate(
        &self,
        document: u64,
        candidate: &DocumentSnapshot,
    ) -> Result<(), String> {
        let session = self.session(document).map_err(|error| error.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "document {document}: expected a closed transaction for verification"
            ));
        }
        if session.snapshot().assembly.is_some() || session.snapshot().drawing.is_some() {
            return Err(format!(
                "document {document}: expected a part document for verification"
            ));
        }
        let guid = self
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let policy = session
            .owned_checkpoint_prefix()
            .and_then(|(policy, _)| policy);
        oblikovati_martensite::validate_full_named_snapshot(candidate, guid, policy)
    }

    /// Commit an unlinked document, its named replay and reply in one checked edit.
    /// Example: `workspace.run_unlinked_prepared(id, policy, edit, project)?`.
    pub fn run_unlinked_prepared<R, T>(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
        project: impl FnOnce(&DocumentSnapshot, R, u64) -> Result<T, String>,
    ) -> Result<T, String> {
        self.run_named_prepared(document, policy, Vec::new(), edit, project, |_| {
            Ok(BTreeMap::new())
        })
    }

    pub(crate) fn run_named_prepared<R, T>(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        readers: Vec<u64>,
        edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
        project: impl FnOnce(&DocumentSnapshot, R, u64) -> Result<T, String>,
        stage: impl FnOnce(&DocumentSnapshot) -> Result<BTreeMap<u64, DocumentSnapshot>, String>,
    ) -> Result<T, String> {
        self.run_named_inner(
            document,
            policy,
            readers,
            NamedAction::Edit(edit),
            project,
            stage,
        )
    }

    /// Navigate an unlinked part after preparing its named replay.
    /// Example: `workspace.navigate_unlinked_prepared(id, policy, direction)?`.
    pub fn navigate_unlinked_prepared(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        direction: oblikovati_model::history::PreparedNavigation,
    ) -> Result<(), String> {
        self.run_named_navigation(
            document,
            policy,
            Vec::new(),
            direction,
            |_, (), _| Ok(()),
            |_| Ok(BTreeMap::new()),
        )
    }

    pub(crate) fn run_named_navigation<T>(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        readers: Vec<u64>,
        direction: oblikovati_model::history::PreparedNavigation,
        project: impl FnOnce(&DocumentSnapshot, (), u64) -> Result<T, String>,
        stage: impl FnOnce(&DocumentSnapshot) -> Result<BTreeMap<u64, DocumentSnapshot>, String>,
    ) -> Result<T, String> {
        let action: NamedAction<UnusedEdit, ()> = NamedAction::Navigate(direction, ());
        self.run_named_inner(document, policy, readers, action, project, stage)
    }

    fn run_named_inner<R, T, C>(
        &mut self,
        document: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        readers: Vec<u64>,
        action: NamedAction<C, R>,
        project: impl FnOnce(&DocumentSnapshot, R, u64) -> Result<T, String>,
        stage: impl FnOnce(&DocumentSnapshot) -> Result<BTreeMap<u64, DocumentSnapshot>, String>,
    ) -> Result<T, String>
    where
        C: FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
    {
        let mut readers = readers;
        readers.sort_unstable();
        if readers.windows(2).any(|pair| pair[0] == pair[1]) {
            return Err(format!("readers {readers:?}: expected unique document IDs"));
        }
        let guid = self
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let source = self.session(document).map_err(|error| error.to_string())?;
        if source.snapshot().assembly.is_some() || source.snapshot().drawing.is_some() {
            return Err(format!(
                "source {document}: expected a part document for prepared replay"
            ));
        }
        source.validate_prepared_named_destination(policy)?;
        if readers.contains(&document) {
            return Err(format!("reader {document}: expected a different source"));
        }
        let mut metadata = BTreeMap::new();
        for &id in &readers {
            let target = self.session(id).map_err(|error| error.to_string())?;
            if target.snapshot().drawing.is_some() {
                return Err(format!(
                    "linked reader {id}: expected a part or assembly document"
                ));
            }
            let is_assembly = target.snapshot().assembly.is_some();
            let target_policy = target
                .owned_checkpoint_prefix()
                .and_then(|(policy, _)| policy);
            if !is_assembly {
                target.validate_prepared_named_destination(target_policy)?;
            }
            let target_guid = self
                .identity(id)
                .map_err(|error| error.to_string())?
                .internal_name;
            metadata.insert(id, (target_guid, target_policy, is_assembly));
        }
        let mut source = None;
        let mut targets = Vec::new();
        let mut stores = Vec::new();
        for (&id, entry) in &mut self.entries {
            if id == document {
                source = Some(&mut entry.session);
            } else if let Some(&(target_guid, target_policy, is_assembly)) = metadata.get(&id) {
                let target = &mut entry.session;
                targets.push((
                    id,
                    &mut target.history,
                    target_guid,
                    target_policy,
                    is_assembly,
                ));
                stores.push((id, &mut target.part_replay, target_policy));
            }
        }
        let source =
            source.ok_or_else(|| format!("source {document}: expected an open document"))?;
        let mut output = None;
        let prepare = |candidate: &oblikovati_model::history::PreparedPrivateEdit<'_>, value: R| {
            let generation = candidate
                .source()
                .generation()
                .checked_add(1)
                .ok_or("source generation: expected available next generation")?;
            let reply = project(candidate.snapshot(), value, generation)?;
            let mut snapshots = stage(candidate.snapshot())?;
            let bodies =
                oblikovati_martensite::prepare_full_named_publication(candidate, guid, policy)?;
            let mut prepared = Vec::new();
            for (id, history, target_guid, target_policy, is_assembly) in targets {
                let snapshot = snapshots
                    .remove(&id)
                    .ok_or_else(|| format!("reader {id}: expected a staged snapshot"))?;
                let refresh = history
                    .prepare_refresh(|current| {
                        *current = snapshot;
                        Ok::<_, String>(())
                    })
                    .map_err(|error| error.to_string())?;
                let refresh = PreparedDependentRefresh::new(candidate, refresh);
                if is_assembly {
                    prepared.push(PreparedChoice::Second(refresh));
                } else {
                    let replay = oblikovati_martensite::prepare_dependent_named_publication(
                        candidate,
                        &refresh,
                        target_guid,
                        target_policy,
                    )?;
                    prepared.push(PreparedChoice::First(PreparedDependentPublication::new(
                        refresh, replay,
                    )));
                }
            }
            if !snapshots.is_empty() {
                return Err(format!(
                    "reader snapshots {:?}: expected planned readers",
                    snapshots.keys()
                ));
            }
            Ok((bodies, (prepared, PreparedValue::new(candidate, reply))))
        };
        let publish = |session: &mut DocumentSession<O>, (bodies, (dependents, reply)), _| {
            session.publish_prepared_named_replay(policy, bodies);
            let mut changed = Vec::new();
            for ((id, store, target_policy), dependent) in stores.into_iter().zip(dependents) {
                match dependent {
                    PreparedChoice::First((updated, scope, replay)) => {
                        crate::part_replay_store::install_prepared_named_replay(
                            store,
                            id,
                            target_policy,
                            scope,
                            replay,
                        );
                        changed.push(updated);
                    }
                    PreparedChoice::Second(updated) => changed.push(updated),
                }
            }
            output = Some((reply, changed));
        };
        match action {
            NamedAction::Edit(edit) => source
                .run_prepared(
                    "Edit Model State Table",
                    oblikovati_model::history::HistoryTransitionKind::ModelStateTable,
                    edit,
                    prepare,
                    publish,
                )
                .map_err(|error| error.to_string())?,
            NamedAction::Navigate(direction, value) => source.navigate_prepared(
                direction,
                |candidate| prepare(candidate, value),
                publish,
            )?,
        }
        let (reply, changed) = output.ok_or("prepared publication: expected a result")?;
        self.notify_prepared_readers(&readers, &changed);
        Ok(reply)
    }

    fn notify_prepared_readers(&mut self, readers: &[u64], changed: &[bool]) {
        for (&id, &updated) in readers.iter().zip(changed) {
            if !updated {
                continue;
            }
            let Some(entry) = self.entries.get_mut(&id) else {
                continue;
            };
            let session = &mut entry.session;
            session.sketch_continuity.refresh(&session.history);
            session.observer.changed(id, session.generation());
        }
    }
}
