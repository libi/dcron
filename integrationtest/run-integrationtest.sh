#/bin/bash
echo "run integration test"
mkdir -p cov
rm -rf cov/*
go test --coverprofile=cov/coverage.txt -covermode=atomic -timeout=30m -coverpkg=github.com/libi/dcron,github.com/libi/dcron/consistenthash github.com/libi/dcron/integrationtest
