# SPDX-License-Identifier: GPL-2.0-only
# Reproduce an OCCT blend on a corpus STEP input and dump per-face AREA, to localize a
# parity area surplus onto specific faces (fillet vs receded plane vs wall).
# Feed via source (NOT `-b <file>`, NOT line-by-line foreach):
#   printf 'source facetypes.tcl\n' | DRAWEXE -b
# Inputs by environment: STEP=<abs path>  RAD=<radius>  EDGEMID="x y z" (filleted edge midpoint).
#
# Surface-type detail: `dumpsurface`/`whatis` are uninformative in this DRAWEXE 8.0.0 build;
# localize by AREA reconciliation against our per-face areas plus the known feature geometry
# (a symmetric family of small cylinder pieces = the split fillet; the two mid-size faces = the
# receded planes). This was conclusive for S1 (fillet Δ+43.2, planes Δ+6.6, Σ=49.8).
pload MODELING
pload STEP
stepread $env(STEP) a *
explode a_1 E
set target ""
foreach e [directory a_1_*] {
    if {[catch { mkcurve __c $e }]} { continue }
    bounds __c __lo __hi
    set um [expr {([dval __lo]+[dval __hi])/2.0}]
    cvalue __c $um px py pz dx dy dz
    set mid $env(EDGEMID)
    set dd [expr {abs([dval px]-[lindex $mid 0])+abs([dval py]-[lindex $mid 1])+abs([dval pz]-[lindex $mid 2])}]
    if {$dd < 0.6} { set target $e }
}
puts "TARGET=$target"
if {$target eq ""} { puts "NO-TARGET"; exit }
blend result a_1 $env(RAD) $target
foreach f [directory result_*] {
    set area "?"
    catch {
        set pr [sprops $f]
        if {[regexp {Mass\s*:\s*([0-9.eE+-]+)} $pr -> m]} { set area $m }
    }
    puts "FACEAREA $f | area=$area"
}
puts "DONE"
exit
