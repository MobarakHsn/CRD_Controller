#!/bin/bash

set -x

vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/MobarakHsn/CRD_Controller/pkg/client \
  github.com/MobarakHsn/CRD_Controller/pkg/apis \
  crd.com:v1 \
  --go-header-file /home/user/go/src/github.com/MobarakHsn/CRD_Controller/hack/boilerplate.go.txt