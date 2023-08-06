#!/usr/bin/env bash

rm -rf swagger
wget https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/6.6.0/openapi-generator-cli-6.6.0.jar -O openapi-generator-cli.jar
# on windows: Invoke-WebRequest -OutFile openapi-generator-cli.jar https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/6.6.0/openapi-generator-cli-6.6.0.jar
java -jar openapi-generator-cli.jar generate -i openapi.yaml -g go -o swagger/

rm openapi-generator-cli.jar
