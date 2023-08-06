#!/usr/bin/env bash

rm -rf swagger
# if openapi-generator-cli.jar exists in the current directory, use it; otherwise, download it
if [ ! -f openapi-generator-cli.jar ]; then
  wget https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/6.6.0/openapi-generator-cli-6.6.0.jar -O openapi-generator-cli.jar
fi
mkdir generator
cd generator
jar -xf ../openapi-generator-cli.jar
cp ../patches/model_anyof.mustache ./go/
jar -cf ../openapi-generator-cli-patched.jar *
jar -cmf META-INF/MANIFEST.MF ../openapi-generator-cli-patched-fixed.jar *
cd ..
rm -rf generator
# on windows: Invoke-WebRequest -OutFile openapi-generator-cli.jar https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/6.6.0/openapi-generator-cli-6.6.0.jar
java -jar openapi-generator-cli-patched-fixed.jar generate -i openapi.yaml -g go -o swagger/

rm swagger/go.mod
rm swagger/go.sum
rm swagger/.gitignore
rm swagger/.openapi-generator-ignore
rm swagger/.travis.yml
rm swagger/git_push.sh
rm swagger/README.md
rm -rf swagger/test/
rm -rf swagger/docs/
rm -rf swagger/api/
rm -rf swagger/.openapi-generator/

