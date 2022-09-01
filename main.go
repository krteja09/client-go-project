package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

func main() {

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	config, err := kubeconfig.ClientConfig()

	if err != nil {
		log.Debugf("Error while getiing configs-- %v ", err)
	}

	// Kubernetes client - package kubernetes
	clientset := kubernetes.NewForConfigOrDie(config)

	// 2. Listing out all namespaces on the cluster
	nsList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Debugf("Error while getiing namespaces-- %v ", err)
	}

	fmt.Println("*************************")
	fmt.Println("Namespaces on the cluster")
	fmt.Println("*************************")
	for _, n := range nsList.Items {
		fmt.Printf("Namepace: %v \n", n.Name)
	}
	fmt.Println("*************************")

	//3. Creating New Namespace
	namespace := "sample-namespace"
	newNamespaceMeta := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), newNamespaceMeta, metav1.CreateOptions{})
	if err != nil {
		log.WithError(err).Error("Error while creating namespace: %v", namespace)
	}

	//4. Creating new pod and that runs a simple hello-world container
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "helloworld", Image: "helloworld:latest", Command: []string{"sleep", "100000"}},
			},
		},
	}

	_, err = clientset.CoreV1().Pods(namespace).Create(context.Background(), newPod, metav1.CreateOptions{})
	if err != nil {
		log.WithError(err).Error("Error while creating pod: %v in the namespace: %v ", newPod.Name, namespace)
	}

	// 5. print out pod names and the namespace they are in for any pods that have a label of ‘k8s-app=kube-dns’
	pods, err := clientset.CoreV1().Pods("kube-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "k8s-app=kube-dns",
	})
	if err != nil {
		log.WithError(err).Error("Error while creating pod: %v in the namespace: %v ", newPod.Name, namespace)
	}
	fmt.Printf("There are %d pods in the cluster with label k8s-app=kube-dns ", len(pods.Items))
	for _, pod := range pods.Items {
		fmt.Printf("\n Pod Name: %v present in Namespace : %v with label k8s-app=kube-dns", pod.Name, pod.Namespace)

	}

	// 6. Deleting created Hello World Pod
	err = clientset.CoreV1().Pods(namespace).Delete(context.Background(), newPod.Name, metav1.DeleteOptions{})
	if err != nil {
		log.WithError(err).Error("Error while deleting pod: %v in the namespace: %v ", newPod.Name, namespace)
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Hour*12)
	controller := NewPodLoggingController(factory)
	stop := make(chan struct{})
	defer close(stop)
	err = controller.Run(stop)
	if err != nil {
		log.WithError(err).Error("Error while running the controller")
	}
	select {}
}

type PodLoggingController struct {
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
}

func (c *PodLoggingController) Run(stopCh chan struct{}) error {
	c.informerFactory.Start(stopCh)
	// wait for the initial synchronization of the local cache.
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}
	return nil
}

func (c *PodLoggingController) podAdd(obj interface{}) {
	pod := obj.(*corev1.Pod)
	log.Infof("POD CREATED: %s/%s", pod.Namespace, pod.Name)
}

func (c *PodLoggingController) podUpdate(old, new interface{}) {
	oldPod := old.(*corev1.Pod)
	newPod := new.(*corev1.Pod)
	log.Infof("POD UPDATED. %s/%s %s", oldPod.Namespace, oldPod.Name, newPod.Status.Phase)
}

func (c *PodLoggingController) podDelete(obj interface{}) {
	pod := obj.(*corev1.Pod)
	log.Infof("POD DELETED: %s/%s", pod.Namespace, pod.Name)
}

// NewPodLoggingController creates a PodLoggingController
func NewPodLoggingController(informerFactory informers.SharedInformerFactory) *PodLoggingController {
	podInformer := informerFactory.Core().V1().Pods()

	c := &PodLoggingController{
		informerFactory: informerFactory,
		podInformer:     podInformer,
	}
	podInformer.Informer().AddEventHandler(
		// Your custom resource event handlers.
		cache.ResourceEventHandlerFuncs{
			// Called on creation
			AddFunc: c.podAdd,
			// Called on resource update and every resyncPeriod on existing resources.
			UpdateFunc: c.podUpdate,
			// Called on resource deletion.
			DeleteFunc: c.podDelete,
		},
	)
	return c
}
