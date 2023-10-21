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
mkdir -p ./build/zips
pushd ./build/build/
zip -q -r "../zips/horizonjs-$tag.zip" *
popd
pushd ./build/docs/
zip -q -r "../zips/horizonjs-docs-$tag.zip" *
popd
pushd ./build/examples/
zip -q -r "../zips/horizonjs-examples-$tag.zip" *
popd

if [ "$tag" == "main" ]; then
    echo "Building website..."
    mkdir -p ./build/website
    mkdir -p ./build/website/docs
    mkdir -p ./build/website/examples
    cp -Rf ./build/build/. ./build/website
    cp -Rf ./build/docs/. ./build/website/docs
    cp -Rf ./build/examples/. ./build/website/examples
    cp -Rf ./build/zips/. ./build/website/release
    # this should already be done since copying from the build docs
    # cp -Rf ./overlay/docs/. ./build/website/docs
    cp -Rf ./overlay/website/. ./build/website/
    # cleanup unwanted from source files
    rm -f ./build/website/docs/welcome.html
    rm -f ./build/website/docs/README.md
    rm -f ./build/website/examples/README.md
    rm -f ./build/website/README.md
fi

echo "Done!"