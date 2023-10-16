#!/bin/bash

# docker system prune -f

tag="$1"

echo "Deleting old build..."
rm -Rf ./build

echo "Building JSBuild and running on source..."
rm -Rf ./jsbuild/obj
rm -Rf ./jsbuild/bin
docker rm horizonjs-jsbuild
docker build -t horizonjs/jsbuild:latest -f ./Dockerfile-jsbuild .
docker create --name=horizonjs-jsbuild horizonjs/jsbuild:latest
docker cp horizonjs-jsbuild:/build ./
echo '!!!!!!!'
ls -la ./build
echo '!!!!!!!'

echo "Building Extdoc and running on source..."
rm -Rf ./build/docs
docker rm horizonjs-extdoc
docker build -t horizonjs/extdoc:latest -f ./Dockerfile-extdoc .
docker create --name=horizonjs-extdoc horizonjs/extdoc:latest
docker cp horizonjs-extdoc:/docs/ ./build

echo "Copying examples..."
cp -Rf ./extjs-2.0.2/examples ./build/examples

echo "Deleting unused paths..."
rm -Rf ./build/source
rm -Rf ./build/build
rm -Rf ./build/package
rm -Rf ./build/resources/raw-images
find ./build/resources/css -type f -not -name 'ext-all.css' -not -name 'xtheme-gray.css' -delete
rm -Rf ./build/resources/license.txt
rm -Rf ./build/resources/resources.jsb

echo "Copying overlay..."
cp -Rf ./overlay/* ./build/

echo "Compressing..."
pushd ./build
zip -q -r "../build/horizonjs-$tag.zip" .
popd

echo "Done!"