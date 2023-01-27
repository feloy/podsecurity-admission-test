package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"k8s.io/pod-security-admission/admission"
	"k8s.io/pod-security-admission/api"
	"k8s.io/pod-security-admission/metrics"
	"k8s.io/pod-security-admission/policy"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	var (
		namespace = "ns1"
		//podPassFile = "pod-pass.yaml"
		podFailFile = "pod-fail.yaml"
	)

	config, err := getConfig()
	if err != nil {
		panic(err)
	}

	// # Examining the requests
	klog.InitFlags(nil)
	flag.Parse()

	// # Getting a clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	labels := ns.GetLabels()
	fmt.Printf("Namespace labels: %+v\n", labels)

	ns1Policy, errList := api.PolicyToEvaluate(labels, api.Policy{})
	fmt.Printf("error #: %d\nEnforce policy: (%s, %s)\n", len(errList), ns1Policy.Enforce.Level, ns1Policy.Enforce.Version)

	podFail, err := getPodFromFile(podFailFile)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v\n", podFail)

	evaluator, err := policy.NewEvaluator(policy.DefaultChecks()) // TODO
	if err != nil {
		panic(err)
	}
	metrics := metrics.NewPrometheusRecorder(api.GetAPIVersion())

	informerFactory := kubeinformers.NewSharedInformerFactory(clientset, 0 /* no resync */)
	namespaceInformer := informerFactory.Core().V1().Namespaces()
	namespaceLister := namespaceInformer.Lister()
	adm := admission.Admission{
		Evaluator:        evaluator,
		Metrics:          metrics,
		PodSpecExtractor: admission.DefaultPodSpecExtractor{},
		PodLister:        admission.PodListerFromClient(clientset),
		NamespaceGetter:  admission.NamespaceGetterFromListerAndClient(namespaceLister, clientset),
	}
	attrs := api.AttributesRecord{
		Name:      podFail.GetName(),
		Namespace: podFail.GetNamespace(),
		Kind:      podFail.GetObjectKind().GroupVersionKind(),
		Operation: v1.Create,
		Object:    podFail,
	}
	response := adm.EvaluatePod(ctx, ns1Policy, nil, &podFail.ObjectMeta, &podFail.Spec, &attrs, true)
	if response == nil {
		fmt.Printf("no response")
		os.Exit(1)
	}
	fmt.Printf("allowed: %v\n", response.Allowed)
}

func getConfig() (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		nil,
	).ClientConfig()
}

func getPodFromFile(file string) (*corev1.Pod, error) {
	var result corev1.Pod
	contents, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(contents, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
