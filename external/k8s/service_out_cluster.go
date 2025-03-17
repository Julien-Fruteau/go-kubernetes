package k8s

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/julien-fruteau/go-kubernetes/internal/env"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type K8SOutCli struct {
	ctx       context.Context
	clientset *kubernetes.Clientset
}

func getKubeConfig() (string, error) {
	// get env KUBE_CONFIG or default homedir kube config

	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	//📢 voluntarily named KUBE_CONFIG and not KUBECONFIG
	kubeconfig = env.GetEnvOrDefault("KUBE_CONFIG", kubeconfig)

	if kubeconfig == "" {
		return "", errors.New("kubeconfig path not found, nor provided by environment variable KUBE_CONFIG")
	}

	return kubeconfig, nil
}

// Instanciate a new kubernetes out of cluster client
// using kubeconfig from env var KUBE_CONFIG (expect file path)
// or trying to find $HOME/.kube/config
// returns a new service with a clientset to interact with k8s
func NewK8SOutCli(ctx context.Context) (*K8SOutCli, error) {
	k := &K8SOutCli{}

	if ctx != nil {
		k.ctx = ctx
	} else {
		k.ctx = context.Background()
	}

	// k.ctx = context.Background()
	// k.ctx, cancel = context.WithTimeout(context.Background(), time.Second*30)

	kubeconfig, err := getKubeConfig()
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	k.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// retrieve cluster images in order to not delete those images
func (k *K8SOutCli) GetClusterImagesV1() ([]string, error) {
	var images []string

	ctx, cancel := context.WithTimeout(k.ctx, DEFAULT_TIMEOUT*time.Second)
	defer cancel() // releases resources if slowOperation completes before timeout elapses

	pods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return images, err
	}
	return getImagesV1(pods), nil
}

func getImagesV1(pods *v1.PodList) []string {
	imageSet := make(map[string]struct{})
	var images []string

	for _, pod := range pods.Items {
		for _, c := range pod.Spec.Containers {
			if c.Image != "" {
				if _, exists := imageSet[c.Image]; !exists {
					imageSet[c.Image] = struct{}{}
					images = append(images, c.Image)
				}
			}
		}
	}
	return images
}

// 📢 READ https://go.dev/blog/pipelines 🛫

func (k *K8SOutCli) GetClusterImagesV2() ([]string, error) {
	ctx, cancel := context.WithTimeout(k.ctx, DEFAULT_TIMEOUT*time.Second)
	defer cancel()

	// Create pod channel
	podChan := make(chan v1.Pod)

	// Start pod streaming in a goroutine
	go func() {
		defer close(podChan)

		// List pods in chunks using continue token
		var continueToken string
		for {
			pods, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
				Limit:    100, // Process pods in smaller batches
				Continue: continueToken,
			})
			if err != nil {
				return
			}

			// Stream each pod
			for i := range pods.Items {
				select {
				case <-ctx.Done():
					return
				case podChan <- pods.Items[i]:
				}
			}

			// Check if we've processed all pods
			continueToken = pods.Continue
			if continueToken == "" {
				return
			}
		}
	}()

	return getImagesV2(podChan), nil
}

func getImagesV2(podChan <-chan v1.Pod) []string {
	imageSet := make(map[string]struct{})

	// Process each pod as it arrives
	for pod := range podChan {
		for _, container := range pod.Spec.Containers {
			if container.Image != "" {
				imageSet[container.Image] = struct{}{}
			}
		}
	}

	// Convert map to slice
	images := make([]string, 0, len(imageSet))
	for image := range imageSet {
		images = append(images, image)
	}

	return images
}

func (k *K8SOutCli) GetImagesV3(pods *v1.PodList) []string {
	// Use a map for O(1) lookups to avoid duplicates
	imageSet := make(map[string]struct{})

	// Process containers from all pods concurrently
	ch := make(chan string)
	var wg sync.WaitGroup

	// Launch goroutine for each pod
	for i := range pods.Items {
		wg.Add(1)
		go func(pod *v1.Pod) {
			defer wg.Done()
			for _, container := range pod.Spec.Containers {
				if container.Image != "" {
					ch <- container.Image
				}
			}
		}(&pods.Items[i])
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Collect unique images
	for image := range ch {
		imageSet[image] = struct{}{}
	}

	// Convert map keys to slice
	images := make([]string, 0, len(imageSet))
	for image := range imageSet {
		images = append(images, image)
	}

	return images
}
