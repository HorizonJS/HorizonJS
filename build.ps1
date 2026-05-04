#!/usr/bin/env pwsh

param(
    [string]$Tag = "$1"
)

# docker system prune -f

Write-Host "Deleting old build..."
if (Test-Path ./build) {
    Remove-Item -Recurse -Force ./build
}

New-Item -ItemType Directory -Force -Path ./build/build/ | Out-Null
New-Item -ItemType Directory -Force -Path ./build/docs/ | Out-Null
New-Item -ItemType Directory -Force -Path ./build/examples/ | Out-Null

Write-Host "Building JSBuild and running on source..."
if (Test-Path ./jsbuild/obj) {
    Remove-Item -Recurse -Force ./jsbuild/obj
}
if (Test-Path ./jsbuild/bin) {
    Remove-Item -Recurse -Force ./jsbuild/bin
}
docker rm horizonjs-jsbuild 2>$null
docker build -t horizonjs/jsbuild:latest -f ./Dockerfile-jsbuild .
docker create --name=horizonjs-jsbuild horizonjs/jsbuild:latest
docker cp horizonjs-jsbuild:/build/. ./build/build/
# Write-Host '!!!!!!!'
# Get-ChildItem ./build
# Write-Host '!!!!!!!'

Write-Host "Building Extdoc and running on source..."
docker rm horizonjs-extdoc 2>$null
docker build -t horizonjs/extdoc:latest -f ./Dockerfile-extdoc .
docker create --name=horizonjs-extdoc horizonjs/extdoc:latest
docker cp horizonjs-extdoc:/docs/. ./build/docs/

Write-Host "Copying examples..."
Copy-Item -Recurse -Force ./extjs-2.0.2/examples/* ./build/examples/

# Write-Host "Deleting unused paths..."
if (Test-Path ./build/build/source) {
    Remove-Item -Recurse -Force ./build/build/source
}
if (Test-Path ./build/build/build/) {
    Remove-Item -Recurse -Force ./build/build/build/
}
if (Test-Path ./build/build/package) {
    Remove-Item -Recurse -Force ./build/build/package
}
if (Test-Path ./build/build/resources/raw-images) {
    Remove-Item -Recurse -Force ./build/build/resources/raw-images
}
# Delete all CSS files except ext-all.css and xtheme-gray.css
Get-ChildItem -Path ./build/build/resources/css -File |
    Where-Object { $_.Name -notin @('ext-all.css', 'xtheme-gray.css') } |
    Remove-Item -Force
if (Test-Path ./build/build/resources/license.txt) {
    Remove-Item -Force ./build/build/resources/license.txt
}
if (Test-Path ./build/build/resources/resources.jsb) {
    Remove-Item -Force ./build/build/resources/resources.jsb
}

Write-Host "Copying overlay..."
if (Test-Path ./overlay/build/) {
    Copy-Item -Recurse -Force ./overlay/build/* ./build/build/
}
if (Test-Path ./overlay/docs/) {
    Copy-Item -Recurse -Force ./overlay/docs/* ./build/docs/
}
if (Test-Path ./overlay/examples/) {
    Copy-Item -Recurse -Force ./overlay/examples/* ./build/examples/
}

Write-Host "Compressing..."
New-Item -ItemType Directory -Force -Path ./build/zips | Out-Null

Push-Location ./build/build/
Compress-Archive -Path * -DestinationPath ../zips/horizonjs-$Tag.zip -Force
Pop-Location

Push-Location ./build/docs/
Compress-Archive -Path * -DestinationPath ../zips/horizonjs-docs-$Tag.zip -Force
Pop-Location

Push-Location ./build/examples/
Compress-Archive -Path * -DestinationPath ../zips/horizonjs-examples-$Tag.zip -Force
Pop-Location

if ($Tag -eq "main") {
    Write-Host "Building website..."
    New-Item -ItemType Directory -Force -Path ./build/website | Out-Null
    New-Item -ItemType Directory -Force -Path ./build/website/docs | Out-Null
    New-Item -ItemType Directory -Force -Path ./build/website/examples | Out-Null
    New-Item -ItemType Directory -Force -Path ./build/website/release | Out-Null
    
    Copy-Item -Recurse -Force ./build/build/* ./build/website/
    Copy-Item -Recurse -Force ./build/docs/* ./build/website/docs/
    Copy-Item -Recurse -Force ./build/examples/* ./build/website/examples/
    Copy-Item -Recurse -Force ./build/zips/* ./build/website/release/
    
    # this should already be done since copying from the build docs
    # Copy-Item -Recurse -Force ./overlay/docs/* ./build/website/docs/
    Copy-Item -Recurse -Force ./overlay/website/* ./build/website/
    
    # cleanup unwanted from source files
    Remove-Item -Force ./build/website/docs/README.md -ErrorAction SilentlyContinue
    Remove-Item -Force ./build/website/examples/README.md -ErrorAction SilentlyContinue
    Remove-Item -Force ./build/website/README.md -ErrorAction SilentlyContinue
}

Write-Host "Done!"
