package k8s

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/util/json"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/sosozhuang/gocomponent/errors"
)

type RunFunc func(*kubernetes.Clientset, []byte) (interface{}, error)
type client struct {
	*kubernetes.Clientset
}

func (this *client) Run(data []byte, f RunFunc) (interface{}, error) {
	return f(this.Clientset, data)
}

func CreateNamespace(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var namespace v1.Namespace
	if err := json.Unmarshal(data, &namespace); err != nil {
		return nil, err
	}

	ns, _ := client.Core().Namespaces().Get(namespace.Name)
	if ns == nil {
		return client.Core().Namespaces().Create(&namespace)
	}
	return ns, errors.Errorf("Namespace %s already exists", ns.Name)
}

func DeleteNamespace(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var namespace v1.Namespace
	if err := json.Unmarshal(data, &namespace); err != nil {
		return nil, err
	}
	ns, _ := client.Core().Namespaces().Get(namespace.Name)
	if ns == nil {
		return ns, errors.Errorf("Namespace %s does not exist", ns.Name)
	} else {
		return ns, client.Core().Namespaces().Delete(namespace.Name, &v1.DeleteOptions{})
	}
}

func NewClient(master, configPath string) (*client, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, configPath)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &client{clientset}, nil
}

func CreatePod(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var pod v1.Pod
	if err := json.Unmarshal(data, &pod); err != nil {
		return nil, err
	}
	po, _ := client.Core().Pods(pod.Namespace).Get(pod.Name)
	if po == nil {
		return client.Core().Pods(pod.Namespace).Create(&pod)
	} else {
		return po, errors.Errorf("Pod %s already exist", po.Name)
	}
}

func DeletePod(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var pod v1.Pod
	if err := json.Unmarshal(data, &pod); err != nil {
		return nil, err
	}
	po, _ := client.Core().Pods(pod.Namespace).Get(pod.Name)
	if po == nil {
		return nil, errors.Errorf("Pod %s does not exist", po.Name)
	} else {
		return client.Core().Pods(pod.Namespace).Delete(po.Name, v1.DeleteOptions{})
	}
}

func CreateDaemonsets(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var daemonset v1beta1.DaemonSet
	if err := json.Unmarshal(data, &daemonset); err != nil {
		return nil, err
	}
	ds, _ := client.ExtensionsV1beta1().DaemonSets(daemonset.Namespace).Get(daemonset.Name)
	if ds == nil {
		return client.ExtensionsV1beta1().DaemonSets(daemonset.Namespace).Create(&daemonset)
	} else {
		return ds, errors.Errorf("Daemonsets %s already exist", ds.Name)
	}
}

func DeleteDaemonsets(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var daemonset v1beta1.DaemonSet
	if err := json.Unmarshal(data, &daemonset); err != nil {
		return nil, err
	}
	ds, _ := client.ExtensionsV1beta1().DaemonSets(daemonset.Namespace).Get(daemonset.Name)
	if ds == nil {
		return nil, errors.Errorf("Daemonsets %s do not exist\n", ds.Name)
	} else {
		return ds, client.ExtensionsV1beta1().DaemonSets(daemonset.Namespace).Delete(ds.Name, &v1.DeleteOptions{})
	}
}

func CreateDeployment(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var deployment v1beta1.Deployment
	if err := json.Unmarshal(data, &deployment); err != nil {
		return nil, err
	}
	dm, _ := client.ExtensionsV1beta1().Deployments(deployment.Namespace).Get(deployment.Name)
	if dm == nil {
		return client.ExtensionsV1beta1().Deployments(deployment.Namespace).Create(&deployment)
	} else {
		return dm, errors.Errorf("Deploymnet %s already exist", dm.Name)
	}
}

func DeleteDeployment(client *kubernetes.Clientset, data []byte) (interface{}, error) {
	var deployment v1beta1.Deployment
	if err := json.Unmarshal(data, &deployment); err != nil {
		return nil, err
	}
	dm, _ := client.ExtensionsV1beta1().Deployments(deployment.Namespace).Get(deployment.Namespace)
	if dm == nil {
		return nil, errors.Errorf("", dm.Name)
	} else {
		return dm, client.ExtensionsV1beta1().Deployments(deployment.Namespace).Delete(deployment.Name, &v1.DeleteOptions{})
	}
}
