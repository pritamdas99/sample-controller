#!/bin/bash
set -x
vendor/k8s.io/code-generator/generate-groups.sh all \
  github.com/PritamDas17021999/sample-controller/pkg/client \
  github.com/PritamDas17021999/sample-controller/pkg/apis \
  pritamdas.dev:v1alpha1 \
  --go-header-file /home/appscodepc/go/src/github.com/PritamDas17021999/sample-controller/hack/boilerplate.go.txt
#
controller-gen rbac:roleName=controller-perms crd paths=github.com/PritamDas17021999/sample-controller/pkg/apis/pritamdas.dev/v1alpha1 \
 crd:crdVersions=v1 output:crd:dir=/home/appscodepc/go/src/github.com/PritamDas17021999/sample-controller/manifests output:stdout