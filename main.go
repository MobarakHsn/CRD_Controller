package main

import (
	"context"
	"flag"
	"fmt"
	clientset "github.com/MobarakHsn/CRD_Controller/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	_ "k8s.io/code-generator"
	"log"
	"path/filepath"
)

func main() {
	log.Println("Configure KubeConfig...")

	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	exampleClient, err := clientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	fmt.Println(kubeClient)
	fmt.Println(exampleClient)
	customs, err := exampleClient.CrdV1().Customs("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Listing customs %s\n", err.Error())
	}
	fmt.Printf("Length of customs is %d\n", len(customs.Items))
}
