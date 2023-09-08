#!/bin/bash

set -x

controller-gen rbac:roleName=controller-perms crd paths=github.com/MobarakHsn/CRD_Controller/pkg/apis/crd.com/v1 crd:crdVersions=v1 output:crd:dir=/home/user/go/src/github.com/MobarakHsn/CRD_Controller/manifests output:stdout