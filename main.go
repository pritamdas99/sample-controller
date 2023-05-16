package main

import (
	"fmt"
	"github.com/PritamDas17021999/sample-controller/pkg/apis/pritamdas.dev/v1alpha1"
	_ "k8s.io/code-generator"
)

func main() {
	k := v1alpha1.Kluster{}
	fmt.Println(k)

}
