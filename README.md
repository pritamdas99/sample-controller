# sample-controller #

Group Name: pritamdas.dev

Version Name: v1alpha1

Resource Name: Kluster

## Code Gen ##
Code gen is needed for generating:
- DeepCopyObject
- Clientset
- Informer
- Lister

### Procedure ###
- cd ~/$Your_Go_Path
- import `"k8s.io/code-generator"` into `main.go`
- run `go mod tidy;go mod vendor`
- run `chmod +x vendor/k8s.io/code-generator/generate-groups.sh`
- run `chmod +x ./hack/codegen.sh`
- run `hack/codegen.sh`
- again run `go mod tidy;go mod vendor`

## Deploy custom resource ##

Just create a yaml file like `manifests/kluster.yaml` and apply.

Run `kubectl get Kluster`

## Resource ##
https://www.linkedin.com/pulse/kubernetes-custom-controllers-part-1-kritik-sachdeva/

https://www.linkedin.com/pulse/kubernetes-custom-controller-part-2-kritik-sachdeva/

https://github.com/ishtiaqhimel/crd-controller/tree/master
