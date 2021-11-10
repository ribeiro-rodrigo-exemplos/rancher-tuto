package main

import (
	"context"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	api "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var clientset *kubernetes.Clientset
var store cache.Store
var imageCapacity map[string]int64

func getClient(pathToCfg string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if pathToCfg == "" {
		logrus.Info("Using in cluster config")
		config, err = rest.InClusterConfig()
	} else {
		logrus.Info("Using out of cluster config")
		config, err = clientcmd.BuildConfigFromFlags("", pathToCfg)
	}

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func pollNodes() error {

	for {

		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), v1.ListOptions{FieldSelector: "metadata.name=minikube"})

		if err != nil {
			logrus.Warnf("Failed to poll the nodes: %v", err)
			continue
		}

		if len(nodes.Items) > 0 {
			node := nodes.Items[0]
			node.Annotations["checked"] = "true"
			updateNode, err := clientset.CoreV1().Nodes().Update(context.TODO(), &node, v1.UpdateOptions{})

			if err != nil {
				logrus.Warnf("Failed to update the node: %v", err)
				continue
			}

			logrus.Infof("node updated %v", updateNode)

			//gracePeriod := int64(10)
			//err = clientset.CoreV1().Nodes().Delete(context.TODO(), updateNode.Name, v1.DeleteOptions{GracePeriodSeconds: &gracePeriod})

		}

		for _, node := range nodes.Items {
			checkImageStore(&node)
		}

		time.Sleep(10 * time.Second)
	}

}

func checkImageStore(node *api.Node) {
	var storage int64
	for _, image := range node.Status.Images {
		storage = storage + image.SizeBytes
	}

	changed := true

	if _, ok := imageCapacity[node.Name]; ok {
		if imageCapacity[node.Name] == storage {
			changed = false
		}
	}

	if changed {
		logrus.Infof("Node [%s] storage occupied by images changed. Old value: [%v], new value: [%v]", node.Name,
			humanize.Bytes(uint64(imageCapacity[node.Name])), humanize.Bytes(uint64(storage)))
		imageCapacity[node.Name] = storage
	} else {
		logrus.Infof("No changes in node [%s] storage occupied by images", node.Name)
	}
}

func main() {
	var err error
	clientset, err = getClient("/Users/ribeiro.santos/.kube/config")

	imageCapacity = make(map[string]int64)

	if err != nil {
		logrus.Error(err)
		return
	}

	go pollNodes()

	for {
		time.Sleep(5 * time.Second)
	}
}
