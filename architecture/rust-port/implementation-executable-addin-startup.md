# Executable flat-directory add-in startup — partial R4

The Linux head now discovers flat-directory native add-ins using Go's
`OBK_ADDINS_DIR` override and executable-adjacent `addins` default. An empty override
uses the default; an unavailable executable location uses relative `addins`.
A missing directory is empty. Other directory errors become visible startup
feedback. Discovery uses a project-owned reader, excludes directories, preserves
Go's case-insensitive shared-library suffixes, and sorts the selected paths.

The selected directory is a native-code trust boundary, as in the Go application.
The loader retains its ABI/version admission checks and worker affinity. The
application does not claim isolation from native constructors or foreign faults.
Only local Linux is tested; suffix recognition is not other-platform acceptance.

A single startup plan loads and registers each selected library through
`NativeAddInDesktop`, then activates startup-policy entries. Load or activation
failure creates named command feedback and continues to the next library. It
never silently substitutes another library. Persisted user policy and installed
bundle discovery are not yet connected. The executable selects the existing
native service adapter only when discovery has libraries, so ordinary empty
startup does not consume the process-wide callback identity.

The frame owner keeps serving events, native callbacks, scripts, and model changes
during startup. A bounded `--frames N` run counts N presentation attempts after
startup is ready; initial startup frames remain included in the returned total
summary. Without add-ins the existing bounded behavior is unchanged. Close waits
for startup readiness and the existing document guard. Explicit shutdown still
retires add-ins before document ownership and checks native reply ownership.

## Local evidence

A real executable test builds API-only Go and Rust document add-ins, copies them
to a selected directory with spaces, and places a broken library before them in
sorted order. It starts `oblikovati-head --frames 3` with the directory override.
The process succeeds and both add-ins produce their native saved/reopened files.
This uses the production startup coordinator and frame loop, not the previous
injected fixture controller.

Named directory-reader tests cover ordering, suffixes, directory exclusion,
missing/denied directories, and override/default/fallback resolution. A real
reader test checks filesystem names and types. A named startup owner proves
bounded frame accounting starts after readiness. Existing real native-window
acceptance still verifies typed document operations and shutdown.

The acceptance profile is local Linux with software Vulkan and validation enabled.
The known AMD native presentation issue remains open. Installed per-user bundles,
policy persistence, hot reload/supervisor behavior, full lifecycle wire routing,
event delivery, directory diagnostics in the application log, and foreground
MCPBridge acceptance remain required. This work does not close R4 or the port epic.

Full local `make gate` passed across all thirteen crates with software Vulkan
and validation enabled. Head production line coverage was 94.73%; duplication
remained below 3%. No remote push, PR, or deferred-platform test was performed.
