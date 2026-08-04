#!/usr/bin/env bash
# Sources the environment to run the locally-built OCCT DRAWEXE headless (the parity oracle).
# Built with: cmake -S OCCT -B occt-build -DBUILD_MODULE_Draw=ON -DBUILD_MODULE_DataExchange=ON
#   -D3RDPARTY_TCL_INCLUDE_DIR=/usr/include/tcl8.6 -D3RDPARTY_TK_INCLUDE_DIR=/usr/include/tcl8.6
#   -D3RDPARTY_TCL_LIBRARY=/usr/lib/x86_64-linux-gnu/libtcl8.6.so -D3RDPARTY_TK_LIBRARY=.../libtk8.6.so
# Then: cmake --build occt-build --target DRAWEXE TKTopTest TKXSDRAWSTEP -j
# Requires apt: tcl8.6-dev tk8.6-dev.  pload keys: MODELING (box/blend/sprops), STEP (stepwrite).
WS=${WS:-/home/vmiguel/git/oblikovati-workspace}
RES="$WS/occt-install/share/opencascade/resources"
export CASROOT="$WS/occt-install"
export LD_LIBRARY_PATH="$WS/occt-build/lin64/gcc/lib:$WS/occt-install/lib:/usr/lib/x86_64-linux-gnu"
export CSF_OCCTResourcePath="$RES"
export DRAWHOME="$RES/DrawResources"
export CSF_DrawPluginDefaults="$RES/DrawResources"
export CSF_STEPDefaults="$RES/XSTEPResource"
export CSF_XSMessage="$RES/XSMessage"
export CSF_SHMessage="$RES/SHMessage"
export CSF_UnitsLexicon="$RES/UnitsAPI/Lexi_Expr.dat"
export CSF_UnitsDefinition="$RES/UnitsAPI/Units.dat"
export DRAWEXE="$WS/occt-build/lin64/gcc/bin/DRAWEXE"
# Usage: source drawenv.sh ; printf 'pload MODELING\npload STEP\n...\nexit\n' | "$DRAWEXE" -b
