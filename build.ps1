#!/usr/bin/env pwsh
param(
    [Parameter(Position = 0)]
    [string]$Tag = "main",
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$BuildArguments
)

$ErrorActionPreference = "Stop"
Push-Location -LiteralPath $PSScriptRoot
try {
    & go run ./cmd/horizon build --tag $Tag @BuildArguments
    $BuildExitCode = $LASTEXITCODE
} finally {
    Pop-Location
}
exit $BuildExitCode
