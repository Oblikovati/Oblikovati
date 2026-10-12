// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn forbidden_standard_libraries_and_loading_doors_are_absent() {
    let source = r#"for _,name in ipairs({"os","io","debug","package","require","load","loadfile","loadstring","dofile","collectgarbage","module","newproxy"}) do assert(_G[name]==nil,name) end
assert(type(math.sqrt)=="function" and type(string.upper)=="function" and type(table.insert)=="function")"#;
    run(source, ScriptLimits::default()).unwrap();
}
#[test]
fn instruction_stack_memory_output_and_host_call_limits_decline() {
    for (source, kind) in QUOTA_CASES {
        let failure = run(source, quota_limits(kind)).unwrap_err();
        assert!(
            failure.contains(kind.expected()),
            "source {source}: expected {} decline, got {failure}",
            kind.expected()
        );
    }
}
#[derive(Clone, Copy)]
enum QuotaKind {
    Instructions,
    Stack,
    Memory,
    Output,
    Calls,
}
const QUOTA_CASES: [(&str, QuotaKind); 5] = [
    ("while true do end", QuotaKind::Instructions),
    ("local function f() return 1+f() end; f()", QuotaKind::Stack),
    ("local x=string.rep('x',4000000)", QuotaKind::Memory),
    ("print(string.rep('x',100))", QuotaKind::Output),
    (
        r#"for i=1,10 do oblikovati.call("echo",{}) end"#,
        QuotaKind::Calls,
    ),
];
impl QuotaKind {
    fn expected(self) -> &'static str {
        match self {
            Self::Instructions => "instructions",
            Self::Stack => "stack",
            Self::Memory => "memory",
            Self::Output => "output",
            Self::Calls => "host calls",
        }
    }
}
fn quota_limits(kind: QuotaKind) -> ScriptLimits {
    let mut limits = ScriptLimits::default();
    match kind {
        QuotaKind::Instructions => limits.instructions = 2_000,
        QuotaKind::Stack => limits.stack_frames = 16,
        QuotaKind::Memory => limits.memory_bytes = 1 << 20,
        QuotaKind::Output => limits.output_bytes = 20,
        QuotaKind::Calls => limits.host_calls = 2,
    }
    limits
}

#[test]
fn wall_timeout_and_stop_terminate_the_child_and_preserve_parent_responsiveness() {
    let limits = ScriptLimits {
        wall_millis: 40,
        instructions: u64::MAX,
        ..Default::default()
    };
    let started = Instant::now();
    assert!(
        run("while true do end", limits)
            .unwrap_err()
            .contains("wall")
    );
    assert!(started.elapsed() < Duration::from_secs(2));
    verify_explicit_stop();
}
fn verify_explicit_stop() {
    let mut execution = engine()
        .start(program("while true do end", ScriptLimits::default()))
        .unwrap();
    execution.stop_handle().cancel();
    assert!(
        execution
            .wait(&EchoScriptHost::default(), &mut Vec::new())
            .unwrap_err()
            .contains("cancelled")
    );
    assert!(
        execution
            .poll(&EchoScriptHost::default(), &mut Vec::new())
            .is_err()
    );
}

#[test]
fn abrupt_worker_failure_does_not_terminate_the_host_process() {
    let crashing = IsolatedLuaEngine::new(
        Path::new(env!("CARGO_BIN_EXE_oblikovati-script-worker")),
        vec!["--abort-fixture".into()],
    );
    let mut execution = crashing
        .start(program("", ScriptLimits::default()))
        .unwrap();
    assert!(
        execution
            .wait(&EchoScriptHost::default(), &mut Vec::new())
            .is_err()
    );
    assert_eq!(
        run("print('alive')", ScriptLimits::default()).unwrap().0,
        "alive\n"
    );
}
struct CancelDuringCall {
    stop: oblikovati_script::ScriptStop,
    committed: RefCell<usize>,
}
impl Caller for CancelDuringCall {
    fn call(&self, _: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        *self.committed.borrow_mut() += 1;
        self.stop.cancel();
        Ok(b"{}".to_vec())
    }
}
#[test]
fn stop_during_host_call_keeps_the_committed_call_and_prevents_the_next() {
    let mut execution = engine()
        .start(program(
            r#"oblikovati.call("first",{}); oblikovati.call("second",{})"#,
            ScriptLimits::default(),
        ))
        .unwrap();
    let host = CancelDuringCall {
        stop: execution.stop_handle(),
        committed: RefCell::new(0),
    };
    assert!(execution.wait(&host, &mut Vec::new()).is_err());
    assert_eq!(*host.committed.borrow(), 1);
}
struct PanickingScriptHost;
impl Caller for PanickingScriptHost {
    fn call(&self, _: &str, _: &[u8]) -> Result<Vec<u8>, Error> {
        panic!("injected host fault")
    }
}
#[test]
fn host_panic_stops_script_calls_and_is_returned_to_the_owner() {
    let mut execution = engine()
        .start(program(
            r#"oblikovati.call("panic",{})"#,
            ScriptLimits::default(),
        ))
        .unwrap();
    assert!(
        execution
            .wait(&PanickingScriptHost, &mut Vec::new())
            .unwrap_err()
            .contains("panicked")
    );
}
