#/bin/bash

function install_gocovmerge {
    echo "install gocovmerge"
    go install github.com/alexfalkowski/gocovmerge@master
}

function run_integration_test {
    cd integrationtest
    ./run-integrationtest.sh
    cd ..
}

function run_driver_test {
    cd driver/$1
    
    mkdir -p cov
    rm -rf cov/*
    go test --coverprofile=cov/coverage.txt -covermode=atomic ./...
    
    cd ../..
}

function run_unit_test {
    mkdir -p cov
    rm -rf cov/*
    go test --coverprofile=cov/coverage.txt -covermode=atomic ./...
}


install_gocovmerge
run_integration_test
run_driver_test etcddriver
run_driver_test redisdriver
run_driver_test rediszsetdriver
run_unit_test

gocovmerge cov/coverage.txt driver/etcddriver/cov/coverage.txt driver/redisdriver/cov/coverage.txt driver/rediszsetdriver/cov/coverage.txt integrationtest/cov/coverage.txt > coverage.txt

