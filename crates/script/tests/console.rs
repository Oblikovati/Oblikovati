// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_api::{Error, client::Caller};
use oblikovati_script::{
    IsolatedLuaEngine, ScriptLimits,
    console::{ConsoleMode, ScriptConsoleController},
};
use std::{
    cell::RefCell,
    path::Path,
    time::{Duration, Instant},
};
#[derive(Default)]
struct ConsoleHost {
    calls: RefCell<Vec<String>>,
}
impl Caller for ConsoleHost {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        self.calls.borrow_mut().push(method.into());
        Ok(request.to_vec())
    }
}
fn console() -> ScriptConsoleController {
    let worker = IsolatedLuaEngine::new(
        Path::new(env!("CARGO_BIN_EXE_oblikovati-script-worker")),
        Vec::new(),
    );
    ScriptConsoleController::new(
        worker,
        ScriptLimits::gui(),
        vec!["test.echo".into(), "test.echo".into()],
    )
}
fn finish(console: &mut ScriptConsoleController, host: &impl Caller) -> ConsoleMode {
    let deadline = Instant::now() + Duration::from_secs(2);
    loop {
        assert!(
            Instant::now() < deadline,
            "console: expected bounded native completion"
        );
        if let Some(mode) = console.poll(host) {
            return mode;
        }
        std::thread::yield_now();
    }
}
#[test]
fn controller_streams_output_and_model_calls_through_owner_frames() {
    let mut console = console();
    let host = ConsoleHost::default();
    console.run(r#"assert(#oblikovati.methods()==1); local reply=oblikovati.test.echo{value=42};print("hello",reply.value);print("a\nb")"#).unwrap();
    assert!(console.snapshot().running);
    assert_eq!(finish(&mut console, &host), ConsoleMode::Run);
    let view = console.snapshot();
    assert_eq!(view.output, ["hello\t42", "a\nb"]);
    assert!(!view.running && !view.checking);
    assert!(view.last_run.unwrap().result.is_ok());
    assert_eq!(&*host.calls.borrow(), &["test.echo"]);
    assert!(console.poll(&host).is_none());
}
#[test]
fn busy_run_and_check_are_refused_and_stop_records_cancellation() {
    let mut console = console();
    console.stop();
    console.run("while true do end").unwrap();
    assert!(console.run("print('second')").is_err());
    assert!(console.check("print('check')").is_err());
    let host = ConsoleHost::default();
    for _ in 0..16 {
        assert!(console.poll(&host).is_none());
    }
    assert!(console.snapshot().running);
    console.stop();
    assert_eq!(finish(&mut console, &host), ConsoleMode::Run);
    assert_run_error(&console, "cancelled");
    console.run("print('after stop')").unwrap();
    finish(&mut console, &host);
    assert!(console.snapshot().last_run.unwrap().result.is_ok());
}
#[test]
fn checks_never_execute_source_and_keep_run_outcomes_separate() {
    let mut console = console();
    let host = ConsoleHost::default();
    console.run("print('keep')").unwrap();
    finish(&mut console, &host);
    let run = console.snapshot().last_run;
    console
        .check(r#"oblikovati.call("should not execute",{});error("not run")"#)
        .unwrap();
    assert!(console.snapshot().checking);
    assert_eq!(finish(&mut console, &host), ConsoleMode::Check);
    assert!(console.snapshot().last_check.unwrap().result.is_ok());
    assert!(host.calls.borrow().is_empty());
    assert_eq!(console.snapshot().last_run, run);
    assert_eq!(console.snapshot().output, ["keep"]);
    console.check("local =").unwrap();
    finish(&mut console, &host);
    assert!(console.snapshot().last_check.unwrap().result.is_err());
}
#[test]
fn clear_and_snapshot_copy_preserve_running_and_last_result() {
    let mut console = console();
    let host = ConsoleHost::default();
    console.run("print('copy')").unwrap();
    finish(&mut console, &host);
    let mut view = console.snapshot();
    view.output[0] = "tampered".into();
    assert_eq!(console.snapshot().output, ["copy"]);
    let last = console.snapshot().last_run;
    console.run("while true do end").unwrap();
    console.clear_output();
    assert!(console.snapshot().output.is_empty());
    assert!(console.snapshot().running);
    assert_eq!(console.snapshot().last_run, last);
    console.stop();
    finish(&mut console, &host);
}
#[test]
fn source_runtime_errors_record_partial_print_and_failed_start_does_not_mark_running() {
    let mut console = console();
    console
        .run("print('before');error('failure value')")
        .unwrap();
    finish(&mut console, &ConsoleHost::default());
    assert_eq!(console.snapshot().output, ["before"]);
    assert_run_error(&console, "failure value");
    let engine = IsolatedLuaEngine::new(Path::new("/missing/console-worker"), Vec::new());
    let mut missing = ScriptConsoleController::new(engine, ScriptLimits::gui(), Vec::new());
    assert!(missing.run("print('never')").is_err());
    assert!(!missing.snapshot().running);
    assert!(missing.snapshot().last_run.is_none());
}
#[test]
fn gui_defaults_use_the_baseline_interactive_time_and_memory_bounds() {
    let limits = ScriptLimits::gui();
    assert_eq!(limits.wall_millis, 5_000);
    assert_eq!(limits.memory_bytes, 128 << 20);
    assert!(limits.instructions > 0);
}

fn assert_run_error(console: &ScriptConsoleController, expected: &str) {
    let failure = console.snapshot().last_run.unwrap().result.unwrap_err();
    assert!(
        failure.contains(expected),
        "failure {failure}: expected {expected:?}"
    );
}
