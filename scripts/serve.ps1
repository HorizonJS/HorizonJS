#!/usr/bin/env pwsh

# Serves ./build at http://localhost:6969/ using only PowerShell and .NET.
# Usage: ./scripts/serve.ps1 [-Port 6969] [-Root ./build]

param(
    [int]$Port = 6969,
    [string]$Root = (Join-Path $PSScriptRoot "..\build")
)

$ErrorActionPreference = "Stop"

function Get-ContentType {
    param([string]$Path)

    switch ([System.IO.Path]::GetExtension($Path).ToLowerInvariant()) {
        ".css" { "text/css; charset=utf-8"; break }
        ".gif" { "image/gif"; break }
        ".htm" { "text/html; charset=utf-8"; break }
        ".html" { "text/html; charset=utf-8"; break }
        ".ico" { "image/x-icon"; break }
        ".jpeg" { "image/jpeg"; break }
        ".jpg" { "image/jpeg"; break }
        ".js" { "text/javascript; charset=utf-8"; break }
        ".json" { "application/json; charset=utf-8"; break }
        ".md" { "text/markdown; charset=utf-8"; break }
        ".png" { "image/png"; break }
        ".svg" { "image/svg+xml"; break }
        ".txt" { "text/plain; charset=utf-8"; break }
        ".xml" { "application/xml; charset=utf-8"; break }
        ".zip" { "application/zip"; break }
        default { "application/octet-stream" }
    }
}

function ConvertTo-HtmlText {
    param([string]$Text)
    [System.Net.WebUtility]::HtmlEncode($Text)
}

function Send-Response {
    param(
        [System.IO.Stream]$Stream,
        [int]$StatusCode,
        [string]$Reason,
        [byte[]]$Body,
        [string]$ContentType = "text/plain; charset=utf-8",
        [bool]$HeadOnly = $false
    )

    $headerText = @(
        "HTTP/1.1 $StatusCode $Reason"
        "Content-Type: $ContentType"
        "Content-Length: $($Body.Length)"
        "Connection: close"
        ""
        ""
    ) -join "`r`n"

    $headerBytes = [System.Text.Encoding]::ASCII.GetBytes($headerText)
    $Stream.Write($headerBytes, 0, $headerBytes.Length)

    if (-not $HeadOnly -and $Body.Length -gt 0) {
        $Stream.Write($Body, 0, $Body.Length)
    }
}

function Send-TextResponse {
    param(
        [System.IO.Stream]$Stream,
        [int]$StatusCode,
        [string]$Reason,
        [string]$Text,
        [string]$ContentType = "text/plain; charset=utf-8",
        [bool]$HeadOnly = $false
    )

    $body = [System.Text.Encoding]::UTF8.GetBytes($Text)
    Send-Response -Stream $Stream -StatusCode $StatusCode -Reason $Reason -Body $body -ContentType $ContentType -HeadOnly $HeadOnly
}

function Send-Redirect {
    param(
        [System.IO.Stream]$Stream,
        [string]$Location,
        [bool]$HeadOnly = $false
    )

    $body = [System.Text.Encoding]::UTF8.GetBytes("Redirecting to $Location")
    $headerText = @(
        "HTTP/1.1 301 Moved Permanently"
        "Location: $Location"
        "Content-Type: text/plain; charset=utf-8"
        "Content-Length: $($body.Length)"
        "Connection: close"
        ""
        ""
    ) -join "`r`n"

    $headerBytes = [System.Text.Encoding]::ASCII.GetBytes($headerText)
    $Stream.Write($headerBytes, 0, $headerBytes.Length)

    if (-not $HeadOnly) {
        $Stream.Write($body, 0, $body.Length)
    }
}

function Get-DirectoryListing {
    param(
        [string]$DirectoryPath,
        [string]$RequestPath
    )

    $safeRequestPath = ConvertTo-HtmlText $RequestPath
    $rows = New-Object System.Collections.Generic.List[string]

    if ($RequestPath -ne "/") {
        [void]$rows.Add('<li><a href="../">../</a></li>')
    }

    Get-ChildItem -LiteralPath $DirectoryPath -Force |
        Sort-Object @{Expression = { -not $_.PSIsContainer }}, Name |
        ForEach-Object {
            $name = $_.Name
            $href = [System.Uri]::EscapeDataString($name)
            $label = ConvertTo-HtmlText $name

            if ($_.PSIsContainer) {
                $href += "/"
                $label += "/"
            }

            [void]$rows.Add("<li><a href=""$href"">$label</a></li>")
        }

    @"
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Directory listing for $safeRequestPath</title>
  <style>
    body { font-family: Segoe UI, Arial, sans-serif; margin: 24px; }
    h1 { font-size: 22px; }
    li { line-height: 1.6; }
  </style>
</head>
<body>
  <h1>Directory listing for $safeRequestPath</h1>
  <ul>
    $($rows -join "`n    ")
  </ul>
</body>
</html>
"@
}

function Test-IsUnderRoot {
    param(
        [string]$RootPath,
        [string]$CandidatePath
    )

    $fullRoot = [System.IO.Path]::GetFullPath($RootPath).TrimEnd(
        [System.IO.Path]::DirectorySeparatorChar,
        [System.IO.Path]::AltDirectorySeparatorChar
    )
    $fullCandidate = [System.IO.Path]::GetFullPath($CandidatePath)
    $rootPrefix = $fullRoot + [System.IO.Path]::DirectorySeparatorChar

    return $fullCandidate.Equals($fullRoot, [System.StringComparison]::OrdinalIgnoreCase) -or
        $fullCandidate.StartsWith($rootPrefix, [System.StringComparison]::OrdinalIgnoreCase)
}

function Test-QuitRequested {
    if (-not [System.Console]::IsInputRedirected -and [System.Console]::KeyAvailable) {
        $key = [System.Console]::ReadKey($true)
        return $key.KeyChar -eq "q" -or $key.KeyChar -eq "Q"
    }

    return $false
}

if (-not (Test-Path -LiteralPath $Root)) {
    throw "Serve root does not exist: $Root"
}

$rootPath = [System.IO.Path]::GetFullPath((Resolve-Path -LiteralPath $Root).Path)
$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)

try {
    $listener.Start()
    Write-Host "Serving $rootPath at http://localhost:$Port/"
    Write-Host "Press q to stop."

    while ($true) {
        if (Test-QuitRequested) {
            Write-Host "Stopping server."
            break
        }

        if (-not $listener.Pending()) {
            Start-Sleep -Milliseconds 100
            continue
        }

        $client = $listener.AcceptTcpClient()

        try {
            $stream = $client.GetStream()
            $reader = [System.IO.StreamReader]::new($stream, [System.Text.Encoding]::ASCII, $false, 1024, $true)
            $requestLine = $reader.ReadLine()

            if ([string]::IsNullOrWhiteSpace($requestLine)) {
                continue
            }

            while ($true) {
                $line = $reader.ReadLine()
                if ($null -eq $line -or $line.Length -eq 0) {
                    break
                }
            }

            $parts = $requestLine.Split(" ")
            if ($parts.Count -lt 2) {
                Send-TextResponse -Stream $stream -StatusCode 400 -Reason "Bad Request" -Text "Bad request"
                continue
            }

            $method = $parts[0].ToUpperInvariant()
            $target = $parts[1]
            $headOnly = $method -eq "HEAD"

            if ($method -ne "GET" -and -not $headOnly) {
                Send-TextResponse -Stream $stream -StatusCode 405 -Reason "Method Not Allowed" -Text "Only GET and HEAD are supported." -HeadOnly $headOnly
                continue
            }

            $pathOnly = $target.Split("?")[0]
            if ([string]::IsNullOrWhiteSpace($pathOnly)) {
                $pathOnly = "/"
            }

            try {
                $decodedPath = [System.Uri]::UnescapeDataString($pathOnly)
            } catch {
                Send-TextResponse -Stream $stream -StatusCode 400 -Reason "Bad Request" -Text "Invalid URL encoding." -HeadOnly $headOnly
                continue
            }

            $relativePath = $decodedPath.TrimStart("/")
            $relativePath = $relativePath.Replace("/", [System.IO.Path]::DirectorySeparatorChar)
            $filePath = Join-Path $rootPath $relativePath
            $fullPath = [System.IO.Path]::GetFullPath($filePath)

            if (-not (Test-IsUnderRoot -RootPath $rootPath -CandidatePath $fullPath)) {
                Send-TextResponse -Stream $stream -StatusCode 403 -Reason "Forbidden" -Text "Forbidden" -HeadOnly $headOnly
                continue
            }

            if (Test-Path -LiteralPath $fullPath -PathType Container) {
                if (-not $decodedPath.EndsWith("/")) {
                    $redirectTarget = $pathOnly + "/"
                    Send-Redirect -Stream $stream -Location $redirectTarget -HeadOnly $headOnly
                    continue
                }

                $indexPath = Join-Path $fullPath "index.html"

                if (Test-Path -LiteralPath $indexPath -PathType Leaf) {
                    $fullPath = $indexPath
                } else {
                    $requestPath = "/" + $relativePath.Replace([System.IO.Path]::DirectorySeparatorChar, "/").Trim("/")
                    if ($requestPath -ne "/") {
                        $requestPath += "/"
                    }

                    $listing = Get-DirectoryListing -DirectoryPath $fullPath -RequestPath $requestPath
                    Send-TextResponse -Stream $stream -StatusCode 200 -Reason "OK" -Text $listing -ContentType "text/html; charset=utf-8" -HeadOnly $headOnly
                    continue
                }
            }

            if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
                Send-TextResponse -Stream $stream -StatusCode 404 -Reason "Not Found" -Text "Not found" -HeadOnly $headOnly
                continue
            }

            $body = [System.IO.File]::ReadAllBytes($fullPath)
            Send-Response -Stream $stream -StatusCode 200 -Reason "OK" -Body $body -ContentType (Get-ContentType $fullPath) -HeadOnly $headOnly
        } catch {
            try {
                $message = "Internal server error: $($_.Exception.Message)"
                Send-TextResponse -Stream $stream -StatusCode 500 -Reason "Internal Server Error" -Text $message
            } catch {
                # Ignore write failures after the client disconnects.
            }
        } finally {
            $client.Close()
        }
    }
} finally {
    $listener.Stop()
}
