#!/bin/bash

# docker system prune -f

tag="$1"

echo "Deleting old build..."
rm -Rf ./build

mkdir -p ./build/build/
mkdir -p ./build/docs/
mkdir -p ./build/examples/

echo "Building JSBuild and running on source..."
rm -Rf ./jsbuild/obj
rm -Rf ./jsbuild/bin
docker rm horizonjs-jsbuild
docker build -t horizonjs/jsbuild:latest -f ./Dockerfile-jsbuild .
docker create --name=horizonjs-jsbuild horizonjs/jsbuild:latest
docker cp horizonjs-jsbuild:/build/. ./build/build/
# echo '!!!!!!!'
# ls -la ./build
# echo '!!!!!!!'

echo "Building Extdoc and running on source..."
docker rm horizonjs-extdoc
docker build -t horizonjs/extdoc:latest -f ./Dockerfile-extdoc .
docker create --name=horizonjs-extdoc horizonjs/extdoc:latest
docker cp horizonjs-extdoc:/docs/. ./build/docs/

echo "Copying examples..."
cp -Rf ./extjs-2.0.2/examples/. ./build/examples/

# echo "Deleting unused paths..."
rm -Rf ./build/build/source
rm -Rf ./build/build/build/
rm -Rf ./build/build/package
rm -Rf ./build/build/resources/raw-images
find ./build/build/resources/css -type f -not -name 'ext-all.css' -not -name 'xtheme-gray.css' -delete
rm -Rf ./build/build/resources/license.txt
rm -Rf ./build/build/resources/resources.jsb

echo "Copying overlay..."
cp -Rf ./overlay/build/. ./build/build/
cp -Rf ./overlay/docs/. ./build/docs/
cp -Rf ./overlay/examples/. ./build/examples/

echo "Compressing..."
pushd ./build/build/
zip -q -r "../horizonjs-$tag.zip" *
popd
pushd ./build/docs/
zip -q -r "../horizonjs-docs-$tag.zip" *
popd
pushd ./build/examples/
zip -q -r "../horizonjs-examples-$tag.zip" *
popd

echo "Done!"