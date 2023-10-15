#!/bin/bash

rm -Rf ./build

dotnet run --project ./jsbuild/JSBuildConsole.csproj --jsb "./extjs-2.0.2/source/ext.jsb" --out "./build"

# dotnet run --project ./JSBuildCore/JSBuildConsole.csproj --jsb "./extjs-2.0.2/resources/resources.jsb" --out "./build/resources"

# dotnet run --project ./JSBuildCore/JSBuildConsole.csproj --jsb "./extjs-2.0.2/source/ext.jsb" --jsdocdockerfile "./JSDocDocker/_Dockerfile" --out "./build"
# dotnet run --project ./JSBuildCore/JSBuildConsole.csproj --jsb "./extjs-2.0.2/resources/resources.jsb" --out "./build/resources"

