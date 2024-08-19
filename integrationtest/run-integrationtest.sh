#/bin/bash

mkdir -p cov
rm -rf cov/*
go test -v --coverprofile=cov/cover-dcron-integration.out -covermode=atomic -timeout=30m -coverpkg=github.com/libi/dcron github.com/libi/dcron/integrationtest
