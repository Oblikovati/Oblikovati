// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_api::{Error, client::Caller};
use oblikovati_script::{IsolatedLuaEngine, ScriptLimits, ScriptProgram};
use std::{
    cell::RefCell,
    path::Path,
    time::{Duration, Instant},
};
#[path = "engine/containment.rs"]
mod containment;
#[derive(Default, Debug)]
struct EchoScriptHost {
    calls: RefCell<Vec<String>>,
}
impl Caller for EchoScriptHost {
    fn call(&self, method: &str, arguments: &[u8]) -> Result<Vec<u8>, Error> {
        self.calls.borrow_mut().push(method.into());
        match method {
            "error" => Err(Error("named host decline".into())),
            "large" => Ok(br#"{"id":9007199254740993}"#.to_vec()),
            "negative-large" => Ok(br#"{"id":-9007199254740993}"#.to_vec()),
            "scalars" => Ok(br#"{"resistivity":1e16,"negative":-1e16,"fractional":1.25}"#.to_vec()),
            "identifier" => {
                serde_json::from_slice::<oblikovati_api::wire::SaveDocumentArgs>(arguments)
                    .map_err(|cause| Error(cause.to_string()))
                    .map(|_| b"{}".to_vec())
            }
            "empty" => Ok(Vec::new()),
            "null" => Ok(b"null".to_vec()),
            _ => Ok(arguments.to_vec()),
        }
    }
}
fn engine() -> IsolatedLuaEngine {
    IsolatedLuaEngine::new(
        Path::new(env!("CARGO_BIN_EXE_oblikovati-script-worker")),
        Vec::new(),
    )
}
fn program(source: &str, limits: ScriptLimits) -> ScriptProgram {
    ScriptProgram {
        source: source.into(),
        limits,
        methods: vec!["test.echo".into(), "deep.group.echo".into()],
        parse_only: false,
    }
}
fn run(source: &str, limits: ScriptLimits) -> Result<(String, EchoScriptHost), String> {
    let host = EchoScriptHost::default();
    let mut output = Vec::new();
    let mut execution = engine().start(program(source, limits))?;
    execution.wait(&host, &mut output)?;
    Ok((String::from_utf8(output).unwrap(), host))
}
#[test]
fn lua51_language_print_and_structural_groups_preserve_go_script_shapes() {
    let source = r#"assert(_VERSION == "Lua 5.1")
local first = oblikovati.call("echo", {points={{1,2},{3,4}}, active=true, name="shape"})
assert(first.points[2][1] == 3 and first.active and first.name == "shape")
local second = oblikovati.test.echo{key=3.5}
assert(second.key == 3.5)
assert(oblikovati.deep.group.echo{tag="nested"}.tag == "nested")
assert(#oblikovati.methods() == 2)
print("done", true, nil, 42)"#;
    let (output, host) = run(source, ScriptLimits::default()).unwrap();
    assert_eq!(output, "done\ttrue\tnil\t42\n");
    assert_eq!(
        &*host.calls.borrow(),
        &["echo", "test.echo", "deep.group.echo"]
    );
}
#[test]
fn nil_and_empty_reply_behavior_matches_the_go_converter() {
    let source = r#"assert(next(oblikovati.call("empty")) == nil)
assert(oblikovati.call("null") == nil)
assert(oblikovati.call("echo", {}) ~= nil)
assert(oblikovati.call("echo", {[2]="sparse"})["2"] == "sparse")"#;
    run(source, ScriptLimits::default()).unwrap();
}
#[test]
fn host_errors_remain_catchable_with_the_method_and_named_decline() {
    let (output, _) = run(
        r#"local ok, err=pcall(oblikovati.call,"error",{})
assert(not ok); print(tostring(err))"#,
        ScriptLimits::default(),
    )
    .unwrap();
    assert!(output.contains("error"));
    assert!(output.contains("named host decline"));
}
#[test]
fn conversion_refuses_cycles_invalid_numbers_ambiguous_keys_and_unsupported_values() {
    for source in [
        r#"local x={};x.self=x;oblikovati.call("echo",x)"#,
        r#"oblikovati.call("identifier",{document=9007199254740993})"#,
        r#"oblikovati.call("large",{})"#,
        r#"oblikovati.call("negative-large",{})"#,
        r#"oblikovati.call("echo",{value=0/0})"#,
        r#"oblikovati.call("echo",{[1]="a",["1"]="b"})"#,
        r#"oblikovati.call("echo",{value=function() end})"#,
        r#"oblikovati.call("echo",{[{}]="key"})"#,
    ] {
        assert!(
            run(source, ScriptLimits::default()).is_err(),
            "source {source}: expected named conversion decline"
        );
    }
}
#[test]
fn parse_only_accepts_runtime_errors_without_executing_or_calling_the_host() {
    let host = EchoScriptHost::default();
    let mut request = program(
        r#"oblikovati.call("echo",{}); error("must not run")"#,
        ScriptLimits::default(),
    );
    request.parse_only = true;
    let mut execution = engine().start(request).unwrap();
    let report = execution.wait(&host, &mut Vec::new()).unwrap();
    assert_eq!(report.host_calls, 0);
    assert!(host.calls.borrow().is_empty());
    assert!(run("local =", ScriptLimits::default()).is_err());
}
#[test]
fn finite_limits_and_source_size_are_checked_before_spawn() {
    let limits = ScriptLimits {
        source_bytes: 2,
        ..Default::default()
    };
    assert!(engine().start(program("print(1)", limits)).is_err());
    let limits = ScriptLimits {
        wall_millis: 0,
        ..Default::default()
    };
    assert!(engine().start(program("", limits)).is_err());
    let missing = IsolatedLuaEngine::new(Path::new("/missing/lua-worker"), Vec::new());
    assert!(missing.start(program("", ScriptLimits::default())).is_err());
}
#[test]
fn execution_lease_rejects_concurrent_runs_and_releases_after_stop_and_completion() {
    let engine = engine();
    let mut active = engine
        .start(program("while true do end", ScriptLimits::default()))
        .unwrap();
    assert!(engine.start(program("", ScriptLimits::default())).is_err());
    active.stop_handle().cancel();
    assert!(
        active
            .wait(&EchoScriptHost::default(), &mut Vec::new())
            .is_err()
    );
    let mut next = engine
        .start(program("print('next')", ScriptLimits::default()))
        .unwrap();
    assert!(
        next.wait(&EchoScriptHost::default(), &mut Vec::new())
            .is_ok()
    );
    assert!(engine.start(program("", ScriptLimits::default())).is_ok());
}

#[test]
fn integral_api_doubles_remain_scalars_without_weakening_integer_id_checks() {
    let source = r#"local x=oblikovati.call("scalars",{});assert(x.resistivity==1e16);assert(x.negative==-1e16);assert(x.fractional==1.25);print('physical-scalars')"#;
    let (output, _) = run(source, ScriptLimits::default()).unwrap();
    assert_eq!(output, "physical-scalars\n");
}

#[test]
fn physical_scalar_arguments_round_trip_while_typed_integer_fields_refuse_large_values() {
    let source = r#"local x=oblikovati.call('scalars',{});local y=oblikovati.call('echo',x);assert(y.resistivity==x.resistivity and y.negative==x.negative);assert(pcall(oblikovati.call,'identifier',{document=9007199254740991}));for _,n in ipairs({1e16,-1e16,9007199254740993}) do assert(not pcall(oblikovati.call,'identifier',{document=n})) end;print('scalar-write')"#;
    let (output, _) = run(source, ScriptLimits::default()).unwrap();
    assert_eq!(output, "scalar-write\n");
}
