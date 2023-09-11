package main

import (
	"flag"
	"github.com/MobarakHsn/CRD_Controller/controller"
	customClientset "github.com/MobarakHsn/CRD_Controller/pkg/client/clientset/versioned"
	customInformers "github.com/MobarakHsn/CRD_Controller/pkg/client/informers/externalversions"
	defaultInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	_ "k8s.io/code-generator"
	"log"
	"path/filepath"
	"time"
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

	defaultClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	customClient, err := customClientset.NewForConfig(config)

	if err != nil {
		panic(err)
	}
	defaultInformerFactory := defaultInformers.NewSharedInformerFactory(defaultClient, time.Second*30)
	customInformerFcatory := customInformers.NewSharedInformerFactory(customClient, time.Second*30)

	contrl := controller.NewController(defaultClient, customClient, defaultInformerFactory.Apps().V1().Deployments(),
		customInformerFcatory.Crd().V1().Customs())
	ch := make(chan struct{})
	defaultInformerFactory.Start(ch)
	customInformerFcatory.Start(ch)
	if err = contrl.Run(2, ch); err != nil {
		log.Println("Error running controller")
	}
}
