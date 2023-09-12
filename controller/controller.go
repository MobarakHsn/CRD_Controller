package controller

import (
	"context"
	"fmt"
	crdv1 "github.com/MobarakHsn/CRD_Controller/pkg/apis/crd.com/v1"
	clientset "github.com/MobarakHsn/CRD_Controller/pkg/client/clientset/versioned"
	customInformer "github.com/MobarakHsn/CRD_Controller/pkg/client/informers/externalversions/crd.com/v1"
	customLister "github.com/MobarakHsn/CRD_Controller/pkg/client/listers/crd.com/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsInformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appsListers "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"time"
)

type Controller struct {
	Clientset        kubernetes.Interface
	CustomClientset  clientset.Interface
	DeploymentLister appsListers.DeploymentLister
	DeploymentSynced cache.InformerSynced
	CustomLister     customLister.CustomLister
	CustomSynced     cache.InformerSynced
	WorkQueue        workqueue.RateLimitingInterface
}

func NewController(Clientset kubernetes.Interface, CustomClientset clientset.Interface,
	DeploymentInformer appsInformer.DeploymentInformer, CustomInformer customInformer.CustomInformer) *Controller {
	contrl := &Controller{
		Clientset:        Clientset,
		CustomClientset:  CustomClientset,
		DeploymentLister: DeploymentInformer.Lister(),
		DeploymentSynced: DeploymentInformer.Informer().HasSynced,
		CustomLister:     CustomInformer.Lister(),
		CustomSynced:     CustomInformer.Informer().HasSynced,
		WorkQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "custom"),
	}
	log.Println("Setting up event handlers")
	CustomInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: contrl.enqueueCustom,
		UpdateFunc: func(oldObj, newObj interface{}) {
			contrl.enqueueCustom(newObj)
		},
		DeleteFunc: contrl.enqueueCustom,
	})
	return contrl
}

func (c *Controller) enqueueCustom(obj interface{}) {
	log.Println("Enqueuing Custom")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilRuntime.HandleError(err)
		return
	}
	c.WorkQueue.AddRateLimited(key)
}

func (c *Controller) Run(workers int, ch <-chan struct{}) error {
	defer utilRuntime.HandleCrash()
	defer c.WorkQueue.ShutDown()
	log.Println("Starting custom controller")
	log.Println("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(ch, c.DeploymentSynced, c.CustomSynced); !ok {
		return fmt.Errorf("\nFailed to wait for cache to sync")
	}
	log.Println("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, ch)
	}
	log.Println("Workers started")
	<-ch
	log.Println("Shutting down workers")
	return nil
}

func (c *Controller) runWorker() {
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	//defer func() { fmt.Println("\n\n\n") }()
	obj, shutdown := c.WorkQueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.WorkQueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.WorkQueue.Forget(obj)
			utilRuntime.HandleError(fmt.Errorf("\nExpected string in workqueue but got %v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			c.WorkQueue.AddRateLimited(key)
			return fmt.Errorf("\nError syncing '%s': %s, requeuing", key, err.Error())
		}
		c.WorkQueue.Forget(obj)
		log.Printf("Successfully synced '%s'\n", key)
		return nil
	}(obj)
	if err != nil {
		utilRuntime.HandleError(err)
		return true
	}
	return true
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilRuntime.HandleError(fmt.Errorf("\nInvalid resource key: %s\n", key))
		return nil
	}
	custom, err := c.CustomLister.Customs(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			utilRuntime.HandleError(fmt.Errorf("\nCustome %s in work queue no longer exists", key))
			return nil
		}
		return err
	}
	//deploymentName := ""
	//if custom.Spec.Name == "" {
	//	rand.Seed(time.Now().UnixNano())
	//	randomNumber := rand.Intn(100000)
	//	deploymentName = custom.Name + "-" + string(randomNumber)
	//	fmt.Println(deploymentName)
	//} else {
	//	deploymentName = custom.Name + "-" + custom.Spec.Name
	//}
	//fmt.Println(deploymentName)
	//return nil
	deploymentName := custom.Spec.Name
	if deploymentName == "" {
		utilRuntime.HandleError(fmt.Errorf("\nName must be specified in the spec for %s", key))
		return nil
	}
	deployment, err := c.DeploymentLister.Deployments(namespace).Get(deploymentName)
	if errors.IsNotFound(err) {
		log.Printf("Deployment %s created .....\n", deploymentName)
		deployment, err = c.Clientset.AppsV1().Deployments(custom.Namespace).Create(context.TODO(), newDeployment(custom), metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}
	if custom.Spec.Replicas != nil && deployment.Spec.Replicas != nil && *custom.Spec.Replicas != *deployment.Spec.Replicas {
		log.Println("Custom %s replicas: d, deployment replicas: %d", name, *custom.Spec.Replicas, *deployment.Spec.Replicas)
		deployment, err = c.Clientset.AppsV1().Deployments(namespace).Update(context.TODO(), newDeployment(custom), metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	//if custom.Spec.Replicas == nil {
	//	custom.Spec.Replicas = 1
	//	deployment, err = c.Clientset.AppsV1().Deployments(namespace).Update(context.TODO(), newDeployment(custom), metav1.UpdateOptions{})
	//	if err != nil {
	//		return err
	//	}
	//}

	err = c.updateCustomStatus(custom, deployment)
	if err != nil {
		return err
	}
	serviceName := deploymentName + "-service"
	service, err := c.Clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		service, err = c.Clientset.CoreV1().Services(namespace).Create(context.TODO(), newService(custom), metav1.CreateOptions{})
		if err != nil {
			log.Println(err)
			return err
		}
		log.Printf("service %s created .....\n", service.Name)
	} else if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func newDeployment(custom *crdv1.Custom) *appsv1.Deployment {

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      custom.Spec.Name,
			Namespace: custom.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(custom, crdv1.SchemeGroupVersion.WithKind("Custom")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: custom.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "api-server",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "api-server",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "api-server",
							Image: custom.Spec.Container.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: custom.Spec.Container.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}

func newService(custom *crdv1.Custom) *corev1.Service {
	labels := map[string]string{
		"app": "api-server",
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: custom.Spec.Name + "-service",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(custom, crdv1.SchemeGroupVersion.WithKind("Custom")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       custom.Spec.Container.Port,
					TargetPort: intstr.FromInt(int(custom.Spec.Container.Port)),
					Protocol:   "TCP",
				},
			},
		},
	}

}

func (c *Controller) updateCustomStatus(custom *crdv1.Custom, deployment *appsv1.Deployment) error {
	customCopy := custom.DeepCopy()
	customCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	_, err := c.CustomClientset.CrdV1().Customs(custom.Namespace).Update(context.TODO(), customCopy, metav1.UpdateOptions{})
	return err
}
