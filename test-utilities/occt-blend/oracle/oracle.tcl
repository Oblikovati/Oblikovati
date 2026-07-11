# SPDX-License-Identifier: GPL-2.0-only
#
# oracle.tcl — the OCCT blend-parity oracle, sourced by the locally-built DRAWEXE.
#
# Runs ONE tests/blend/<grid>/<case> script unmodified and emits, per case:
#   $ORACLE_OUT/$ORACLE_NAME.step  — OCCT's exact INPUT solid (the shape `blend` filleted)
#   $ORACLE_OUT/$ORACLE_NAME.json  — the pick list (radius + geometric edge locator),
#                                    OCCT's reference area, deps, and any TODO marker.
#
# Why a Tcl driver over DRAWEXE instead of a bespoke C++ binary (Task 1 spike outcome):
# every primitive the oracle needs is already a DRAW command in the built toolkits —
# `stepwrite` (TKXSDRAWSTEP) exports the input solid, and an edge's true mid-parameter
# point + tangent come from `mkcurve`+`bounds`+`cvalue`. So we intercept OCCT's own
# `blend`/`checkprops` commands (recording their args, then delegating), let OCCT resolve
# the picked edges exactly as it does in the test, and read each edge's locator back with
# OCCT's own curve evaluation. No re-implementation of OCCT's shape DSL, no C++ build.
#
# Inputs arrive by environment (DRAWEXE batch mode has no argv for a sourced script):
#   ORACLE_CASE  — absolute path to the case .tcl file
#   ORACLE_BEGIN — absolute path to the grid `begin` file (sets SCALE, pload TOPTEST, …)
#   ORACLE_OUT   — output directory (created by the wrapper)
#   ORACLE_GRID  — grid name (e.g. "simple"), for the record
#   ORACLE_NAME  — case name (e.g. "A1"), for the record + file stems

pload MODELING
pload STEP

set casePath  $env(ORACLE_CASE)
set beginPath $env(ORACLE_BEGIN)
set outDir    $env(ORACLE_OUT)
set gridName  $env(ORACLE_GRID)
set caseName  $env(ORACLE_NAME)

# --- recording state (global; DRAW command overrides write here) --------------------
set ORV(verb)    ""
set ORV(input)   ""
set ORV(picks)   {} ;# each: {radiusNumeric edgeName lawOrEmpty}
set ORV(exparea) "0"
set ORV(deps)    "0.01"
set ORV(todo)    ""

# edgeloc: OCCT's own resolution of a picked edge into a geometry-only locator —
# the true curve mid-parameter point and unit tangent. DRAW shapes are Tcl *globals*,
# so both the input edge and the temp curve must be declared global inside the proc
# (the load-bearing DRAW gotcha found in the Task 1 spike).
proc edgeloc {edge} {
    global __oc
    global $edge
    mkcurve __oc $edge
    bounds __oc __lo __hi
    set um [expr {([dval __lo] + [dval __hi]) / 2.0}]
    cvalue __oc $um __px __py __pz __dx __dy __dz
    set L [expr {sqrt([dval __dx]*[dval __dx] + [dval __dy]*[dval __dy] + [dval __dz]*[dval __dz])}]
    if {$L == 0} { set L 1 }
    return [list [dval __px] [dval __py] [dval __pz] \
                 [expr {[dval __dx]/$L}] [expr {[dval __dy]/$L}] [expr {[dval __dz]/$L}]]
}

# Intercept `blend result object (rad edge)+`. Record the input object and each
# (radius, edge) pair; radii are DRAW expressions (e.g. "1*SCALE1"). Grid variables like
# SCALE1 are Tcl *globals*, and `dval` resolves names through the current scope — so the
# expression MUST be evaluated at global scope (uplevel #0), else SCALE1 reads as 0 inside
# this proc (the Task 1 spike bug behind the early radius:0 records).
rename blend __real_blend
proc blend {args} {
    global ORV
    set ORV(verb)  "blend"
    set ORV(input) [lindex $args 1]
    foreach {r e} [lrange $args 2 end] {
        lappend ORV(picks) [list [uplevel #0 [list dval $r]] $e ""]
    }
    catch {eval __real_blend $args}
}

# Intercept `checkprops shape -s <area> [-deps <d>]`: capture OCCT's reference area and
# tolerance; do NOT delegate (we assert on our side, not OCCT's).
rename checkprops __real_checkprops
proc checkprops {args} {
    global ORV
    set i [lsearch -exact $args "-s"]
    if {$i >= 0} { set ORV(exparea) [lindex $args [expr {$i + 1}]] }
    set d [lsearch -exact $args "-deps"]
    if {$d >= 0} { set ORV(deps) [lindex $args [expr {$d + 1}]] }
}

# Variable-radius fillet: `mkevol result s` then `updatevol edge p0 r0 p1 r1 …` (repeated)
# then `buildevol`. Record verb+input on mkevol; each updatevol becomes one pick carrying its
# parameter→radius law (constant Radius unused, 0). Parameters and radii may be SCALE
# expressions, so evaluate at global scope (see the blend override).
rename mkevol __real_mkevol
proc mkevol {args} {
    global ORV
    set ORV(verb)  "buildevol"
    set ORV(input) [lindex $args 1]
    catch {eval __real_mkevol $args}
}
rename updatevol __real_updatevol
proc updatevol {args} {
    global ORV
    set edge [lindex $args 0]
    set law  {}
    foreach {p r} [lrange $args 1 end] {
        lappend law [list [uplevel #0 [list dval $p]] [uplevel #0 [list dval $r]]]
    }
    lappend ORV(picks) [list 0 $edge $law]
    catch {eval __real_updatevol $args}
}
# buildevol must be wrapped too: an uncaught build error would abort the sourced case before
# the trailing `checkprops` line runs, losing the reference area (the area:0 bug).
rename buildevol __real_buildevol
proc buildevol {args} {
    catch {eval __real_buildevol $args}
}

# bfuseblend: fuse(shape1,shape2), then blend ALL edges of their boolean section with one
# radius (BRepTest_FilletCommands.cxx). Record only the operands + radius here; the fused
# solid and the section edges are resolved post-source at global scope (see below), so the
# input STEP is the fused solid and one pick is emitted per section edge, all same radius.
rename bfuseblend __real_bfuseblend
proc bfuseblend {args} {
    global ORV
    set ORV(verb)   "bfuseblend"
    set ORV(bf_s)   [lindex $args 1]
    set ORV(bf_b)   [lindex $args 2]
    set ORV(bf_rad) [uplevel #0 [list dval [lindex $args 3]]]
    catch {eval __real_bfuseblend $args}
}

# --- detect OCCT's own TODO/INCOMPLETE marker in the case text ----------------------
# OCCT marks a case it knows fails with `puts "TODO … TEST INCOMPLETE"`; we mirror that
# (never stricter than OCCT). Scan the file text rather than trapping puts at runtime.
proc scanTodo {path} {
    set fh [open $path r]
    set txt [read $fh]
    close $fh
    foreach line [split $txt "\n"] {
        if {[string match {*TODO*} $line] || [string match {*TEST INCOMPLETE*} $line]} {
            return [string trim $line]
        }
    }
    return ""
}

set ORV(todo) [scanTodo $casePath]

# --- run the grid setup + the case (both intercepted) -------------------------------
catch {source $beginPath}
catch {source $casePath}

# bfuseblend post-processing (global scope): build the fused input solid and one pick per
# boolean-section edge — the exact edge set OCCT's bfuseblend blends.
if {$ORV(verb) eq "bfuseblend" && [info exists ORV(bf_s)]} {
    global __bf_fused __bf_sec
    if {![catch {bfuse __bf_fused $ORV(bf_s) $ORV(bf_b)}]} {
        set ORV(input) "__bf_fused"
    }
    if {![catch {bsection __bf_sec $ORV(bf_s) $ORV(bf_b)} secErr]} {
        set secEdges [explode __bf_sec e]
        foreach e $secEdges {
            lappend ORV(picks) [list $ORV(bf_rad) $e ""]
        }
    } else {
        set ORV(todo) "bfuseblend-section-failed: $secErr"
    }
}

# --- JSON emission ------------------------------------------------------------------
proc jnum {x} { return [format "%.10g" $x] }

proc jlocator {loc} {
    return "{\"midpoint\":\[[jnum [lindex $loc 0]],[jnum [lindex $loc 1]],[jnum [lindex $loc 2]]\],\"direction\":\[[jnum [lindex $loc 3]],[jnum [lindex $loc 4]],[jnum [lindex $loc 5]]\]}"
}

proc jesc {s} { return [string map {\\ \\\\ \" \\\"} $s] }

proc jlaw {law} {
    if {$law eq ""} { return "null" }
    set parts {}
    foreach pr $law { lappend parts "\[[jnum [lindex $pr 0]],[jnum [lindex $pr 1]]\]" }
    return "\[[join $parts ,]\]"
}

set pickJson {}
foreach p $ORV(picks) {
    set radius [lindex $p 0]
    set edge   [lindex $p 1]
    set law    [lindex $p 2]
    if {[catch {edgeloc $edge} loc]} {
        # Unresolvable edge — record loudly so the generator never silently drops it.
        set locJson "{\"midpoint\":\[0,0,0\],\"direction\":\[0,0,0\]}"
        set ORV(todo) "unresolved-edge $edge: $loc"
    } else {
        set locJson [jlocator $loc]
    }
    lappend pickJson "{\"radius\":[jnum $radius],\"locator\":$locJson,\"law\":[jlaw $law]}"
}

# Export OCCT's exact input solid as STEP (mode "a" = as-is). Skip when input is unknown
# (case never called blend) — the JSON's empty picks/inputStep flags it downstream.
set stepName ""
if {$ORV(input) ne ""} {
    set stepName "$caseName.step"
    catch {stepwrite a $ORV(input) [file join $outDir $stepName]}
}

set json "{\"grid\":\"[jesc $gridName]\",\"case\":\"[jesc $caseName]\",\"verb\":\"[jesc $ORV(verb)]\",\"expectedArea\":[jnum $ORV(exparea)],\"deps\":[jnum $ORV(deps)],\"todo\":\"[jesc $ORV(todo)]\",\"inputStep\":\"[jesc $stepName]\",\"picks\":\[[join $pickJson ,]\]}"

set jf [open [file join $outDir "$caseName.json"] w]
puts $jf $json
close $jf

puts "ORACLE_DONE $gridName/$caseName picks=[llength $ORV(picks)] area=$ORV(exparea) todo=\"$ORV(todo)\""
