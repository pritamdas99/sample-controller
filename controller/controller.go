package controller

import (
	"context"
	"fmt"
	controllerv1 "github.com/PritamDas17021999/sample-controller/pkg/apis/pritamdas.dev/v1alpha1"
	clientset "github.com/PritamDas17021999/sample-controller/pkg/client/clientset/versioned"
	informer "github.com/PritamDas17021999/sample-controller/pkg/client/informers/externalversions/pritamdas.dev/v1alpha1"
	lister "github.com/PritamDas17021999/sample-controller/pkg/client/listers/pritamdas.dev/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"strings"
	"time"
)

type Controller struct {
	// kuberclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// sampleclientset is a clientset for our own API group
	sampleclientset   clientset.Interface
	deploymentLister  appslisters.DeploymentLister
	deploymentsSynced cache.InformerSynced
	pritamLister      lister.PritamLister
	pritamSynced      cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workQueue workqueue.RateLimitingInterface
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	sampleclientset clientset.Interface,
	deploymentInformer appsinformer.DeploymentInformer,
	pritamInformer informer.PritamInformer) *Controller {

	ctrl := &Controller{
		kubeclientset:     kubeclientset,
		sampleclientset:   sampleclientset,
		deploymentLister:  deploymentInformer.Lister(),
		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		pritamLister:      pritamInformer.Lister(),
		pritamSynced:      pritamInformer.Informer().HasSynced,
		workQueue:         workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{}),
	}

	log.Println("Setting up eventhandler")

	// Set up an event handler for when Pritam resources change
	pritamInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.enquePritam,
		UpdateFunc: func(oldObj, newObj interface{}) {
			ctrl.enquePritam(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enquePritam(obj)
		},
	})
	return ctrl
}

// enquePritam takes an Pritam resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Pritam.
func (c *Controller) enquePritam(obj interface{}) {
	log.Println("Enqueing Pritam")
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workQueue.AddRateLimited(key)
}

func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()
	log.Println("Starting Pritam Controller")
	log.Println("waiting for informer cache to sync")

	if ok := cache.WaitForCacheSync(stopCh, c.deploymentsSynced, c.pritamSynced); !ok {
		return fmt.Errorf("failed to wait for cache to sync")
	}

	log.Println("starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	log.Println("worker started")
	<-stopCh
	log.Println("shutting down workers")
	return nil
}

func (c *Controller) runWorker() {
	for c.ProcessNextItem() {

	}
}

func (c *Controller) ProcessNextItem() bool {
	obj, shutdown := c.workQueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func, so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off period.
		defer c.workQueue.Done(obj)
		var key string
		var ok bool

		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the workqueue.
		if key, ok = obj.(string); !ok {
			c.workQueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		log.Println("key here", key, ok)
		// Run the syncHandler, passing it the namespace/name string of the
		// Pritam resource to be synced.
		if err := c.syncHandler(key); err != nil {
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}

		// Finally, if no error occurs we Forget this item, so it does not
		// get queued again until another change happens.
		c.workQueue.Forget(obj)
		log.Printf("successfully synced '%s'\n", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}
	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Pritam resource
// with the current status of the resource.
// implement the business logic here.

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}
	// Get the Pritam resource with this namespace/name

	pritam, err := c.pritamLister.Pritams(namespace).Get(name)
	if err != nil {
		// The Pritam resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			// We choose to absorb the error here as the worker would requeue the
			// resource otherwise. Instead, the next time the resource is updated
			// the resource will be queued again.
			log.Printf("pritam '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}
	//fmt.Println("we are here")

	deploymentName := pritam.Spec.Name
	ss := []string{"service", pritam.Spec.Name}
	serviceName := strings.Join(ss, "-")
	log.Println("got deployment name", deploymentName)
	if pritam.Spec.Name == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.

		//utilruntime.HandleError(fmt.Errorf("%s : deployment name must be specified", key))
		//return nil
		//s := "sha1 this string"
		//h := sha1.New()
		//h.Write([]byte(s))
		//sha1_hash := hex.EncodeToString(h.Sum(nil))

		//fmt.Println(s, sha1_hash)
		ss = []string{pritam.Name, "missing"}
		deploymentName = strings.Join(ss, "-")
		ss = []string{"service", pritam.Name, "missing"}
		serviceName = strings.Join(ss, "-")
	}
	log.Println("we are here", pritam.Spec.Name)

	c.DeploymentHandler(pritam, namespace, name, deploymentName)
	c.ServiceHandler(pritam, serviceName, deploymentName)

	return nil
}

func (c *Controller) DeploymentHandler(pritam *controllerv1.Pritam, namespace string, name string, deploymentName string) error {
	// Get the deployment with the name specified in Pritam.spec
	deployment, err := c.deploymentLister.Deployments(namespace).Get(deploymentName)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment, err = c.kubeclientset.AppsV1().Deployments(pritam.Namespace).Create(context.TODO(), newDeployment(pritam, deploymentName), metav1.CreateOptions{})
		if err != nil {
			return err
		}

		log.Println("deployment name here for pritam das dot dev", deploymentName)
	}
	// If an error occurs during Get/Create, we'll requeue the item, so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return err
	}

	// If this number of the replicas on the Pritam resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if pritam.Spec.Replicas != nil && *pritam.Spec.Replicas != *deployment.Spec.Replicas {
		log.Printf("Pritam %s replicas: %d, deployment replicas: %d\n", name, *pritam.Spec.Replicas, *deployment.Spec.Replicas)
		deployment, err = c.kubeclientset.AppsV1().Deployments(namespace).Update(context.TODO(), newDeployment(pritam, deploymentName), metav1.UpdateOptions{})

		// If an error occurs during Update, we'll requeue the item, so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		if err != nil {
			return err
		}
	}
	// Finally, we update the status block of the Pritam resource to reflect the
	// current state of the world
	err = c.updatePritamStatus(pritam, deployment)
	if err != nil {
		return err
	}
	return nil

}

func (c *Controller) ServiceHandler(pritam *controllerv1.Pritam, serviceName string, deploymentName string) error {
	// check if service exists or not
	service, err := c.kubeclientset.CoreV1().Services(pritam.Namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		service, err = c.kubeclientset.CoreV1().Services(pritam.Namespace).Create(context.TODO(), newService(pritam, serviceName, deploymentName), metav1.CreateOptions{})
		if err != nil {
			log.Println(err)
			return err
		}
		log.Printf("\nservice %s created .....\n", service.Name)
	} else if err != nil {
		log.Println(err)
		return err
	}

	_, err = c.kubeclientset.CoreV1().Services(pritam.Namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil

}

func (c *Controller) updatePritamStatus(pritam *controllerv1.Pritam, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	pritamCopy := pritam.DeepCopy()
	pritamCopy.Status.AvailableReplicas = deployment.Status.AvailableReplicas

	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Pritam resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	_, err := c.sampleclientset.PritamdasV1alpha1().Pritams(pritam.Namespace).Update(context.TODO(), pritamCopy, metav1.UpdateOptions{})
	return err

}

// newDeployment creates a new Deployment for a Pritam resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Pritam resource that 'owns' it.

func Arrr(arr ...string) []string {
	var ss []string

	for _, v := range arr {
		ss = append(ss, v)
	}
	return ss
}

func newDeployment(pritam *controllerv1.Pritam, deploymentName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: pritam.Namespace,
			OwnerReferences: []metav1.OwnerReference{

				*metav1.NewControllerRef(pritam, controllerv1.SchemeGroupVersion.WithKind("Pritam")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pritam.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					controllerv1.App: strings.Join(Arrr("my-app", deploymentName), "-"),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						controllerv1.App: strings.Join(Arrr("my-app", deploymentName), "-"),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "my-app",
							Image: pritam.Spec.Container.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: pritam.Spec.Container.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}

func newService(pritam *controllerv1.Pritam, name string, dep_name string) *corev1.Service {
	labels := map[string]string{
		controllerv1.App: strings.Join(Arrr("my-app", dep_name), "-"),
	}
	return &corev1.Service{

		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pritam, controllerv1.SchemeGroupVersion.WithKind(controllerv1.ResourceDefinition)),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeNodePort,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port:       pritam.Spec.Container.Port,
					TargetPort: intstr.FromInt(int(pritam.Spec.Container.Port)),
				},
			},
		},
	}
}
