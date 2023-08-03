#!/usr/bin/env bash


#docker run --rm -v ${PWD}:/local swaggerapi/swagger-codegen-cli -v

docker run --rm -v ${PWD}:/local swaggerapi/swagger-codegen-cli-v3 generate \
    -DapiTests=false \
    -i /local/openapi.yaml \
    -l go \
    -o /local/swagger