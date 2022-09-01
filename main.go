package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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

}
