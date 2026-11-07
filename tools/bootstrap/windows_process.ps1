# SPDX-License-Identifier: GPL-2.0-only
# Windows native argument quoting follows CommandLineToArgvW/CRT backslash rules, also on PS5.1.
function ConvertTo-NativeArgument {
    [OutputType([string])]
    param([AllowEmptyString()][string] $Argument)
    if ($Argument.IndexOf([char]0) -ge 0) { throw 'GUI argument contains NUL: expected a Windows command-line string' }
    if ($Argument -ne '' -and $Argument -notmatch '[\s"]') { return $Argument }
    $escaped = [regex]::Replace($Argument, '(\\*)"', '$1$1\"')
    $escaped = [regex]::Replace($escaped, '(\\+)$', '$1$1')
    return '"' + $escaped + '"'
}
function New-NativeStartInfo {
    [OutputType([Diagnostics.ProcessStartInfo])]
    param([string] $Executable, [string[]] $Arguments, [bool] $Capture)
    $command = Get-Command $Executable -CommandType Application,ExternalScript -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $command -and $command.CommandType -eq 'ExternalScript') {
        $Arguments = @('-NoLogo', '-NoProfile', '-NonInteractive', '-ExecutionPolicy', 'Bypass', '-File', $command.Source) + $Arguments
        $Executable = [Diagnostics.Process]::GetCurrentProcess().MainModule.FileName
    }
    $start = [Diagnostics.ProcessStartInfo]::new()
    $start.FileName = $Executable
    $start.Arguments = [string]::Join(' ', @($Arguments | ForEach-Object { ConvertTo-NativeArgument $_ }))
    $start.UseShellExecute = $false
    $start.WorkingDirectory = (Get-Location).ProviderPath
    $start.RedirectStandardOutput = $Capture
    $start.RedirectStandardError = $Capture
    if ($Capture) { $start.StandardOutputEncoding = [Text.UTF8Encoding]::new($false); $start.StandardErrorEncoding = [Text.UTF8Encoding]::new($false) }
    return $start
}
class BuildBootstrapHost {
    [bool] Has([string] $Name) { return $null -ne (Get-Command $Name -CommandType Application,ExternalScript -ErrorAction SilentlyContinue) }
    [bool] IsWindows() { return [Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT }
    [string] Architecture() {
        $architecture = $env:PROCESSOR_ARCHITEW6432
        if ([string]::IsNullOrEmpty($architecture)) { $architecture = $env:PROCESSOR_ARCHITECTURE }
        switch ($architecture) {
            'AMD64' { return 'x64' }
            'ARM64' { return 'arm64' }
            default { throw "architecture $architecture`: expected a native x64 or ARM64 Windows terminal" }
        }
        return ''
    }
    [int] Run([string] $Executable, [string[]] $Arguments) {
        $process = [Diagnostics.Process]::new()
        try {
            $process.StartInfo = New-NativeStartInfo $Executable $Arguments $false
            if (-not $process.Start()) { throw "command $Executable`: expected process creation" }
            $process.WaitForExit()
            return $process.ExitCode
        } finally { $process.Dispose() }
    }
    [string] Capture([string] $Executable, [string[]] $Arguments) {
        $process = [Diagnostics.Process]::new()
        try {
            $process.StartInfo = New-NativeStartInfo $Executable $Arguments $true
            if (-not $process.Start()) { throw "command $Executable`: expected process creation" }
            $stdout = $process.StandardOutput.ReadToEndAsync()
            $stderr = $process.StandardError.ReadToEndAsync()
            $process.WaitForExit()
            $captured = $stdout.GetAwaiter().GetResult()
            $diagnostic = $stderr.GetAwaiter().GetResult()
            if ($process.ExitCode -ne 0) { throw "command $Executable $Arguments failed ($($process.ExitCode)): $diagnostic" }
            return $captured
        } finally { $process.Dispose() }
    }
    [void] Download([string] $Url, [string] $Output) {
        if (-not $Url.StartsWith('https://', [StringComparison]::Ordinal)) { throw "download $Url`: expected HTTPS" }
        # Preserve system certificate validation; a redirect cannot replace the pinned hash/signature.
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $Url -OutFile $Output -UseBasicParsing -ErrorAction Stop
    }
    [string] MicrosoftSigner([string] $Path) {
        $signature = Get-AuthenticodeSignature -LiteralPath $Path
        if ($signature.Status -ne 'Valid' -or $null -eq $signature.SignerCertificate) {
            throw "installer $Path signature $($signature.Status): expected a valid Microsoft Authenticode signature"
        }
        return $signature.SignerCertificate.Subject
    }
    [void] RefreshPath() {
        $machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
        $user = [Environment]::GetEnvironmentVariable('Path', 'User')
        $cargoHome = $env:CARGO_HOME
        if ([string]::IsNullOrEmpty($cargoHome)) { $cargoHome = Join-Path $env:USERPROFILE '.cargo' }
        $env:PATH = "$(Join-Path $cargoHome 'bin');$env:PATH;$machine;$user;$env:LOCALAPPDATA\Programs\Git\cmd;$env:ProgramFiles\Git\cmd"
    }
    [string] MsvcInstallation() {
        $vswhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
        if (-not (Test-Path -LiteralPath $vswhere -PathType Leaf)) { return '' }
        return $this.Capture($vswhere, @('-latest', '-products', '*', '-requires', 'Microsoft.VisualStudio.Component.VC.Tools.x86.x64', '-property', 'installationPath', '-utf8')).Trim()
    }
    [void] EnterMsvc([string] $Installation, [string] $Architecture) {
        $entry = Join-Path $Installation 'Common7\Tools\Launch-VsDevShell.ps1'
        if (-not (Test-Path -LiteralPath $entry -PathType Leaf)) { throw "MSVC environment $entry`: expected Launch-VsDevShell.ps1" }
        $native = 'amd64'
        if ($Architecture -eq 'arm64') { $native = 'arm64' }
        & $entry -Arch $native -HostArch $native -SkipAutomaticLocation | Out-Host
    }
}
function Invoke-RequiredCommand {
    param([BuildBootstrapHost] $HostAccess, [string] $Executable, [string[]] $Arguments)
    $status = $HostAccess.Run($Executable, $Arguments)
    if ($status -eq 3010) { throw "installer $Executable requires a reboot; restart Windows and rerun build.ps1" }
    if ($status -ne 0) { throw "command $Executable $Arguments failed ($status): expected success; package installation can require an administrator terminal" }
}
