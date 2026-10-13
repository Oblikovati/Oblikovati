// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_script::{IsolatedLuaEngine, ScriptLimits, ScriptProgram};
use oblikovati_session::{files::FileWorkspace, workspace::OpenActivation};
use std::path::{Path, PathBuf};
mod arguments;
mod source;
pub use source::{NativeScriptSources, ScriptSources};
type ScriptDocuments<F> =
    FileWorkspace<observations::CliObservations, observations::CliObservations, F>;
pub(super) fn run_native<F: DocumentFiles>(
    args: &[OsString],
    files: F,
    guids: &mut dyn GuidSource,
    output: &mut dyn Write,
) -> Result<(), String> {
    let executable = std::env::current_exe()
        .map_err(|error| format!("script worker: expected current executable: {error}"))?;
    let engine =
        IsolatedLuaEngine::new(&executable, vec![oblikovati_script::WORKER_ARGUMENT.into()]);
    run_script_with_files(
        args,
        files,
        guids,
        output,
        &mut NativeScriptSources,
        &engine,
    )
}
/// Run the CLI Lua command with owned documents and injected source/worker access.
/// Example: `run_script_with_files(&args, files, &mut guids, output, sources, &engine)?;`.
pub fn run_script_with_files<F: DocumentFiles>(
    args: &[OsString],
    files: F,
    guids: &mut dyn GuidSource,
    output: &mut dyn Write,
    sources: &mut dyn ScriptSources,
    engine: &IsolatedLuaEngine,
) -> Result<(), String> {
    let arguments = arguments::parse(args)?;
    let limits = ScriptLimits::default();
    let source = sources.read_source(&arguments.script, limits.source_bytes)?;
    let mut session = ScriptSession::new(files, guids)?;
    session.open(arguments.document.as_deref())?;
    session.execute(source, limits, engine, output)?;
    session.save(arguments.save.as_deref(), output)
}
struct ScriptSession<'a, F> {
    documents: ScriptDocuments<F>,
    guids: &'a mut dyn GuidSource,
    version: String,
}
impl<'a, F: DocumentFiles> ScriptSession<'a, F> {
    fn new(files: F, guids: &'a mut dyn GuidSource) -> Result<Self, String> {
        Ok(Self {
            documents: FileWorkspace::new(observations::CliObservations, files),
            guids,
            version: BuildIdentity::parse(BUILD_MANIFEST)?.version,
        })
    }
    fn open(&mut self, path: Option<&Path>) -> Result<(), String> {
        let Some(path) = path else {
            return Ok(());
        };
        self.documents
            .open(
                path,
                observations::CliObservations,
                &self.version,
                self.guids,
                OpenActivation::Activate,
            )
            .map(|_| ())
            .map_err(|error| format!("script run: open {path:?}: {error}"))
    }
    fn execute(
        &mut self,
        source: String,
        limits: ScriptLimits,
        engine: &IsolatedLuaEngine,
        output: &mut dyn Write,
    ) -> Result<(), String> {
        let program = ScriptProgram {
            source,
            limits,
            methods: oblikovati_automation::file_host_methods(),
            parse_only: false,
        };
        let mut execution = engine.start(program)?;
        let mut factory = observations::CliObservationFactory;
        let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
        let host =
            oblikovati_automation::FileHost::new(&mut self.documents, self.guids, &self.version)
                .with_observer_factory(&mut factory)
                .with_sketch_construction(&construction, Default::default());
        execution
            .wait(&host, output)
            .map(|_| ())
            .map_err(|error| format!("script run: {error}"))
    }
    fn save(&mut self, path: Option<&Path>, output: &mut dyn Write) -> Result<(), String> {
        let Some(path) = path else {
            return Ok(());
        };
        let active = self
            .documents
            .workspace()
            .active_document()
            .ok_or("script run: save requested with no active document")?;
        self.documents
            .save_as(active, path, &self.version, self.guids)
            .map_err(|error| format!("script run: save {path:?}: {error}"))?;
        write_line(
            output,
            &format!("saved active document to {}", path.display()),
        )
    }
}
