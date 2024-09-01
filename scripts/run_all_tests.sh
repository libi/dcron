#/bin/bash

function install_gocovmerge {
    echo "install gocovmerge"
    go install github.com/alexfalkowski/gocovmerge@v1.0.4
}

function run_integration_test {
    cd integrationtest
    bash run-integrationtest.sh
    cd ..
}

function run_unit_test {
    mkdir -p cov
    rm -rf cov/*
    go test --coverprofile=cov/coverage.txt -covermode=atomic ./...
}


install_gocovmerge
run_integration_test
run_unit_test

gocovmerge cov/coverage.txt integrationtest/cov/coverage.txt > coverage.txt

