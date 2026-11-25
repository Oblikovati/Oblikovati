# Oblikovati

Oblikovati is an open-source, parametric mechanical CAD application. It is
designed to build 3D parts from sketches, dimensions, and features. Parameters
and feature history let you change a design and recompute its geometry.

The desktop application provides a model browser, a ribbon, and a 3D viewport.
It uses the [Martensite](https://github.com/Oblikovati/Martensite) geometry
kernel and Vulkan rendering. A command-line application is also available.

**Status:** The Rust application is in development. Modeling tools and platform
support are not yet complete. Follow the
[project milestones](https://github.com/Oblikovati/Oblikovati/milestones) for
planned work.

## Install and run

The Rust application is not yet published as an installable release. To try the
current development version, build it from source. The existing GitHub releases
are for the earlier Go application.

Clone the repository:

```sh
git clone --branch rust-work https://github.com/Oblikovati/Oblikovati.git
cd Oblikovati
```

On Linux or macOS, build and start the desktop application:

```sh
./build.sh run
```

On Windows, use PowerShell:

```powershell
.\build.ps1 run
```

The scripts check for build dependencies and install missing dependencies when
possible. Installation can require administrator permission. A Vulkan-capable
GPU and driver are required to run the desktop application. On Linux, the GUI
also needs an X11 or XWayland display.

The desktop executable is `oblikovati` (`oblikovati.exe` on Windows). The
command-line executable is `oblikovati-cli`. See [Build and run](BUILDING.md)
for prerequisites, platform details, and other build commands.

On Linux, set `OBLIKOVATI_VSYNC=0` before launching the desktop to prefer
immediate Vulkan presentation on X11. Set `OBLIKOVATI_VULKAN_VALIDATION=1`
to collect Vulkan diagnostics in a release build.

## Desktop appearance

Choose Dark, Light, or High Contrast under **View → Appearance → Theme**.
Use **Colour scheme** to control the viewport backdrop independently. Default
uses the dark studio gradient, Presentation uses the light studio gradient,
and High Contrast uses a solid black background.

## Learn more

- [Project website](https://oblikovati.org)
- [Build and run](BUILDING.md)
- [Application architecture](architecture/rust-port/plan.md)
- [Public API](https://github.com/Oblikovati/Oblikovati.API)

## License

Oblikovati is licensed under [GPL-2.0-only](LICENSE). The separate public API
is licensed under Apache-2.0.
