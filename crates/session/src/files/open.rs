// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::history::DocumentSnapshot;
use oblikovati_persistence::load::{PreparedParameterPackage, read_parameter_package};

#[derive(Clone, Copy, PartialEq)]
enum OpenContentMode<'a> {
    Full,
    Deferred,
    Express((f64, f64), &'a str),
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Reuse an open canonical path or validate and register a new file.
    /// Example: `host.open(path, observer, "alpha", source, OpenActivation::Activate)?;`.
    pub fn open(
        &mut self,
        path: &Path,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        self.open_with_factory(path, || Ok(observer), version, source, activation)
    }

    /// Create an observer only for a new canonical document owner.
    /// Example: `host.open_with_factory(path, || factory.create(), "alpha", source, activation)?`.
    pub fn open_with_factory(
        &mut self,
        path: &Path,
        factory: impl FnOnce() -> Result<O, String>,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        self.open_in_mode(
            path,
            factory,
            version,
            source,
            activation,
            OpenContentMode::Full,
        )
    }

    /// Open an assembly from a validated saved view without loading its components.
    /// Example: `files.open_express_with_factory(path, factory, version, source, (0.05, 0.5), producer)?`.
    pub fn open_express_with_factory(
        &mut self,
        path: &Path,
        factory: impl FnOnce() -> Result<O, String>,
        version: &str,
        source: &mut dyn GuidSource,
        display_limits: (f64, f64),
        producer: &str,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        self.open_in_mode(
            path,
            factory,
            version,
            source,
            OpenActivation::Activate,
            OpenContentMode::Express(display_limits, producer),
        )
    }

    /// Open a drawing with its saved views frozen; example: `files.open_deferred_with_factory(path, factory, version, source)?`.
    pub fn open_deferred_with_factory(
        &mut self,
        path: &Path,
        factory: impl FnOnce() -> Result<O, String>,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        self.open_in_mode(
            path,
            factory,
            version,
            source,
            OpenActivation::Activate,
            OpenContentMode::Deferred,
        )
    }

    fn open_in_mode(
        &mut self,
        path: &Path,
        factory: impl FnOnce() -> Result<O, String>,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
        mode: OpenContentMode<'_>,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        if let Some(&id) = self.paths.get(path) {
            self.require_matching_defer_mode(id, mode)?;
            return self.reuse_open(id, activation);
        }
        let canonical = checked_path(path, self.files.canonical_path(path))?;
        if let Some(&id) = self.paths.get(&canonical) {
            self.require_matching_defer_mode(id, mode)?;
            return self.reuse_open(id, activation);
        }
        let observer = factory().map_err(FileWorkspaceError::ObserverFactory)?;
        self.open_new(canonical, observer, version, source, activation, mode)
    }

    fn require_matching_defer_mode(
        &self,
        id: u64,
        mode: OpenContentMode<'_>,
    ) -> Result<(), FileWorkspaceError> {
        if self.retained_file(id)?.express_view.is_some()
            != matches!(mode, OpenContentMode::Express(..))
        {
            return Err(FileWorkspaceError::ExpressMode(
                id,
                matches!(mode, OpenContentMode::Express(..)),
            ));
        }
        if let OpenContentMode::Express((chord, angular), producer) = mode {
            let retained = self.retained_file(id)?;
            if let Some(view) = &retained.express_view {
                view.validate_producer(producer).map_err(|reason| {
                    FileWorkspaceError::ExpressOpen(
                        retained.path.clone().unwrap_or_default(),
                        reason.to_string(),
                    )
                })?;
                view.validate_display_quality(chord, angular)
                    .map_err(|reason| {
                        FileWorkspaceError::ExpressOpen(
                            retained.path.clone().unwrap_or_default(),
                            reason.to_string(),
                        )
                    })?;
            }
        }
        if mode == OpenContentMode::Deferred && self.deferred_drawing_views(id)?.is_none() {
            return Err(FileWorkspaceError::DeferredDrawing(
                id,
                "expected a drawing already open in deferred mode".into(),
            ));
        }
        Ok(())
    }

    fn reuse_open(
        &mut self,
        id: u64,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        if activation == OpenActivation::Activate {
            self.workspace
                .activate(id)
                .map_err(FileWorkspaceError::Workspace)?;
        }
        Ok(FileOpenOutcome::Existing(id))
    }

    fn open_new(
        &mut self,
        path: PathBuf,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
        mode: OpenContentMode<'_>,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        let prepared = self.prepare_open(&path, version, source)?;
        let express_view = if let OpenContentMode::Express((chord, angular), producer) = mode {
            let view = self
                .validated_assembly_view(&path, &prepared.package, producer)
                .map_err(|reason| FileWorkspaceError::ExpressOpen(path.clone(), reason))?
                .ok_or_else(|| {
                    FileWorkspaceError::ExpressOpen(
                        path.clone(),
                        "expected a saved assembly view".into(),
                    )
                })?;
            view.validate_display_quality(chord, angular)
                .map_err(|reason| {
                    FileWorkspaceError::ExpressOpen(path.clone(), reason.to_string())
                })?;
            Some(view)
        } else {
            None
        };
        let views = if mode == OpenContentMode::Deferred {
            Some(
                prepared
                    .package
                    .restore_saved_drawing_views()
                    .map_err(FileWorkspaceError::Package)?
                    .ok_or_else(|| {
                        FileWorkspaceError::DeferredOpen(
                            path.clone(),
                            "expected saved drawing view geometry".into(),
                        )
                    })?,
            )
        } else {
            None
        };
        let feature_mark = prepared.work_geometry.ids.high_water();
        let snapshot = DocumentSnapshot {
            work_geometry: prepared.work_geometry.program,
            sketches: prepared.sketches.sketches,
            attributes: prepared.attributes,
            attachments: prepared.attachments,
            interests: prepared.interests,
            resources: prepared.resources,
            assets: prepared.assets,
            model_states: prepared.model_states,
            drawing: prepared.drawing,
            assembly: prepared.assembly,
            body_names: prepared.body_names,
            design_views: prepared.design_views,
            part_design_views: prepared.part_design_views,
            sketch_settings: prepared.sketch_settings,
            parameters: prepared.restored.parameters,
            properties: prepared.properties,
        };
        let sketch_marks = oblikovati_model::history::SketchReservationMarks {
            sketch: prepared.sketches.sketch_ids.high_water(),
            entity: prepared.sketches.entity_ids.high_water(),
            constraints: prepared.sketches.constraint_ids.marks(),
            dimension_names: prepared.sketches.dimension_names.marks(),
        };
        let history = super::history_restore::restore_history(
            snapshot,
            prepared.restored.ids,
            feature_mark,
            sketch_marks,
            prepared.restored.derived_high_water,
        )?;
        let registration = self
            .workspace
            .register(history, prepared.identity, observer, source, activation)
            .map_err(FileWorkspaceError::Workspace)?;
        let outcome = self.install_file(path, prepared.package, registration);
        if let Some(view) = express_view {
            self.retained
                .get_mut(&registration.session_id)
                .expect("installed file")
                .express_view = Some(view);
        }
        if let Some(views) = views {
            self.retained
                .get_mut(&registration.session_id)
                .expect("installed file")
                .deferred_views = Some(views);
        }
        self.workspace
            .notify_unavailable_interest_clients(registration.session_id);
        Ok(outcome)
    }

    fn prepare_open(
        &mut self,
        path: &Path,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<PreparedParameterPackage, FileWorkspaceError> {
        self.workspace
            .approve_open(path)
            .map_err(|reason| FileWorkspaceError::OpenVeto {
                path: path.into(),
                reason,
            })?;
        let prepared = read_parameter_package(path, version, source, &mut self.files)
            .map_err(FileWorkspaceError::Load)?;
        super::create::require_document_kind(prepared.package.manifest().document_type)?;
        if let Some(settings) = &prepared.sketch_settings {
            crate::sketch_policy::require_supported_sketch_policies(settings)
                .map_err(FileWorkspaceError::UnsupportedSketchPolicy)?;
        }
        Ok(prepared)
    }

    pub(super) fn install_file(
        &mut self,
        path: PathBuf,
        package: DocumentPackage,
        registration: RegisteredDocument,
    ) -> FileOpenOutcome {
        let display_override = name::display_override(&path, &package.manifest().display_name);
        let package = if display_override.is_none() {
            package.with_display_name("")
        } else {
            package
        };
        self.paths.insert(path.clone(), registration.session_id);
        self.retained.insert(
            registration.session_id,
            RetainedFile {
                visible: true,
                path: Some(path),
                package,
                display_override,
                picture: None,
                drawing_views: None,
                assembly_view: None,
                express_view: None,
                deferred_views: None,
            },
        );
        self.refresh_display_name(registration.session_id);
        FileOpenOutcome::Opened(registration)
    }
}
