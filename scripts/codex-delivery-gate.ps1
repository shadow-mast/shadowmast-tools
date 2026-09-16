<#
.SYNOPSIS
Runs the local Codex delivery gate.

.DESCRIPTION
Builds missing local CLI binaries, runs repo-audit and codex-status, and optionally runs artifact-verify or release-checker.

.PARAMETER ArtifactPath
Artifact path passed to artifact-verify. Repeat -ArtifactPath for multiple paths.

.PARAMETER ReleaseRepo
GitHub repository passed to release-checker.

.PARAMETER ReleaseTag
Release tag passed to release-checker.

.PARAMETER RequireAsset
Required release asset passed to release-checker. Repeat -RequireAsset for multiple assets.

.PARAMETER ReleaseLocalDir
Local directory passed to release-checker for SHA256SUMS.txt verification.

.PARAMETER SkipBuild
Do not build missing binaries.
#>
[CmdletBinding()]
param(
    [string[]]$ArtifactPath = @(),
    [string]$ReleaseRepo = "",
    [string]$ReleaseTag = "",
    [string[]]$RequireAsset = @(),
    [string]$ReleaseLocalDir = "",
    [switch]$SkipBuild
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = (Resolve-Path (Join-Path $scriptDir "..")).Path
$binDir = Join-Path $repoRoot ".codex-tools\bin"
$finalExit = 0
$lastToolOutput = ""

function New-ToolSpec {
    param(
        [string]$EnvName,
        [string]$DefaultPath,
        [string]$Package
    )

    $override = [Environment]::GetEnvironmentVariable($EnvName)
    if ($override) {
        return @{ Path = $override; Package = $Package; Override = $true }
    }
    return @{ Path = $DefaultPath; Package = $Package; Override = $false }
}

$tools = @{
    "repo-audit"      = New-ToolSpec -EnvName "CODEX_GATE_REPO_AUDIT" -DefaultPath (Join-Path $binDir "repo-audit.exe") -Package ".\cmd\repo-audit"
    "codex-status"    = New-ToolSpec -EnvName "CODEX_GATE_CODEX_STATUS" -DefaultPath (Join-Path $binDir "codex-status.exe") -Package ".\cmd\codex-status"
    "artifact-verify" = New-ToolSpec -EnvName "CODEX_GATE_ARTIFACT_VERIFY" -DefaultPath (Join-Path $binDir "artifact-verify.exe") -Package ".\cmd\artifact-verify"
    "release-checker" = New-ToolSpec -EnvName "CODEX_GATE_RELEASE_CHECKER" -DefaultPath (Join-Path $binDir "release-checker.exe") -Package ".\cmd\release-checker"
}

function Set-GateExitCode {
    param([int]$Code)

    if ($Code -eq 0) {
        return
    }
    if ($Code -eq 1 -and $script:finalExit -eq 0) {
        $script:finalExit = 1
        return
    }
    if ($Code -ge 2) {
        $script:finalExit = 2
    }
}

function Ensure-Tool {
    param(
        [string]$Name,
        [string]$Path,
        [string]$Package,
        [bool]$Override
    )

    if (Test-Path $Path) {
        return
    }
    if ($Override) {
        throw "$Name override path does not exist: $Path"
    }
    if ($SkipBuild) {
        throw "$Name binary is missing and -SkipBuild was set"
    }
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "$Name binary is missing and Go is not available in PATH"
    }

    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Path) | Out-Null
    Push-Location $repoRoot
    try {
        & go build -o $Path $Package
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed for $Name with exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }
}

function Invoke-GateTool {
    param(
        [string]$Name,
        [string]$Path,
        [string[]]$Arguments
    )

    Write-Output ""
    Write-Output "## $Name"
    Write-Output ""
    Write-Output '```text'
    $output = & $Path @Arguments 2>&1
    $code = $LASTEXITCODE
    $script:lastToolOutput = ($output | ForEach-Object { [string]$_ }) -join [Environment]::NewLine
    foreach ($line in $output) {
        Write-Output ([string]$line)
    }
    Write-Output '```'
    Write-Output ""
    Write-Output "- exit code: $code"
    Set-GateExitCode -Code $code
}

function Test-CodexStatusBlocked {
    param([string]$Output)

    return $Output -match "PR not found or gh unavailable|CI checks unavailable through gh|CI checks JSON could not be parsed"
}

Write-Output "# Codex Delivery Gate"
Write-Output ""
Write-Output "- repo: $repoRoot"
Write-Output "- platform: Windows/PowerShell"
Write-Output "- exit codes: 0 pass, 1 check failure, 2 blocked gate/tool error"

$requiredTools = @("repo-audit", "codex-status")
if ($ArtifactPath.Count -gt 0) {
    $requiredTools += "artifact-verify"
}
if ($ReleaseRepo -ne "" -or $ReleaseTag -ne "") {
    $requiredTools += "release-checker"
}

try {
    foreach ($name in $requiredTools) {
        Ensure-Tool -Name $name -Path $tools[$name].Path -Package $tools[$name].Package -Override $tools[$name].Override
    }
}
catch {
    Write-Output ""
    Write-Output "## Build"
    Write-Output ""
    Write-Output '```text'
    Write-Output $_.Exception.Message
    Write-Output '```'
    Write-Output ""
    Write-Output "## Summary"
    Write-Output ""
    Write-Output "- Local autonomous delivery gate: partially blocked"
    Write-Output "- Reason: $($_.Exception.Message)"
    exit 2
}

Push-Location $repoRoot
try {
    Invoke-GateTool -Name "repo-audit" -Path $tools["repo-audit"].Path -Arguments @("--root", $repoRoot)
    Invoke-GateTool -Name "codex-status" -Path $tools["codex-status"].Path -Arguments @()
    if (Test-CodexStatusBlocked -Output $lastToolOutput) {
        Write-Output "- gate note: codex-status could not fully verify PR/CI"
        Set-GateExitCode -Code 2
    }

    if ($ArtifactPath.Count -gt 0) {
        Invoke-GateTool -Name "artifact-verify" -Path $tools["artifact-verify"].Path -Arguments $ArtifactPath
    }

    if ($ReleaseRepo -ne "" -or $ReleaseTag -ne "") {
        if ($ReleaseRepo -eq "" -or $ReleaseTag -eq "") {
            Write-Output ""
            Write-Output "## release-checker"
            Write-Output ""
            Write-Output '```text'
            Write-Output "Release checks require both -ReleaseRepo and -ReleaseTag."
            Write-Output '```'
            Write-Output ""
            Write-Output "- exit code: 2"
            Set-GateExitCode -Code 2
        }
        else {
            $releaseArgs = @("--repo", $ReleaseRepo, "--tag", $ReleaseTag)
            foreach ($asset in $RequireAsset) {
                $releaseArgs += @("--require-asset", $asset)
            }
            if ($ReleaseLocalDir -ne "") {
                $releaseArgs += @("--local-dir", $ReleaseLocalDir)
            }
            Invoke-GateTool -Name "release-checker" -Path $tools["release-checker"].Path -Arguments $releaseArgs
        }
    }
}
finally {
    Pop-Location
}

Write-Output ""
Write-Output "## Summary"
Write-Output ""
if ($finalExit -eq 0) {
    Write-Output "- Local autonomous delivery gate: PASS"
}
elseif ($finalExit -eq 1) {
    Write-Output "- Local autonomous delivery gate: FAIL"
}
else {
    Write-Output "- Local autonomous delivery gate: partially blocked"
}
Write-Output "- Final exit code: $finalExit"

exit $finalExit
