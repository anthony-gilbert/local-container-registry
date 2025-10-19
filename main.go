package main

//go:generate go run build.go

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/go-github/v63/github"
	"github.com/joho/godotenv"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Repositories struct {
	id            int64
	owner         string
	repoName      string
	fullName      string
	commit        string
	prDescription string
}

type DockerImage struct {
	ID        string
	RepoTags  []string
	Size      string
	CreatedAt string
}

type TableData struct {
	CommitSHA     string
	PRDescription string
	ImageID       string
	ImageSize     string
	ImageTag      string
	PushedAt      string
	CreatedAt     string
	// Kubernetes specific fields
	PodName   string
	Namespace string
	Status    string
	Restarts  string
	Age       string
	NodeName  string
}

// This init() function loads in the .env file into environment variables

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or could not load it:", err)
	}
}

func setupLogging() {
	// Redirect logs to a file to avoid interfering with TUI
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// If we can't create log file, just discard logs
		log.SetOutput(io.Discard)
		return
	}
	log.SetOutput(logFile)
}

func disableLogging() {
	// Disable logging output to not interfere with TUI
	log.SetOutput(io.Discard)
}

var db *sql.DB

type RegistryCatalog struct {
	Repositories []string `json:"repositories"`
}

type RegistryTags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type ImageManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
}

type ImageConfig struct {
	Created string `json:"created"`
}

func getImageCreationTime(registryHost, repository, tag string) string {
	// Get the manifest first
	manifestCmd := exec.Command("curl", "-s", "-H", "Accept: application/vnd.docker.distribution.manifest.v2+json",
		fmt.Sprintf("http://%s/v2/%s/manifests/%s", registryHost, repository, tag))
	manifestOutput, err := manifestCmd.Output()
	if err != nil {
		return "Unknown"
	}

	var manifest ImageManifest
	if err := json.Unmarshal(manifestOutput, &manifest); err != nil {
		return "Unknown"
	}

	// Get the config blob to extract creation time
	if manifest.Config.Digest != "" {
		configCmd := exec.Command("curl", "-s",
			fmt.Sprintf("http://%s/v2/%s/blobs/%s", registryHost, repository, manifest.Config.Digest))
		configOutput, err := configCmd.Output()
		if err != nil {
			return "Unknown"
		}

		var config ImageConfig
		if err := json.Unmarshal(configOutput, &config); err != nil {
			return "Unknown"
		}

		if config.Created != "" {
			// Parse the RFC3339 timestamp and format it nicely
			if t, err := time.Parse(time.RFC3339, config.Created); err == nil {
				return t.Format("2006-01-02 15:04:05")
			}
		}
	}

	return "Unknown"
}

func getImageSize(registryHost, repository, tag string) string {
	// Get the manifest first to find config and layer sizes
	manifestCmd := exec.Command("curl", "-s", "-H", "Accept: application/vnd.docker.distribution.manifest.v2+json",
		fmt.Sprintf("http://%s/v2/%s/manifests/%s", registryHost, repository, tag))
	manifestOutput, err := manifestCmd.Output()
	if err != nil {
		return "Unknown"
	}

	var manifest ImageManifest
	if err := json.Unmarshal(manifestOutput, &manifest); err != nil {
		return "Unknown"
	}

	// Calculate total size from config + layers
	totalSize := int64(manifest.Config.Size)

	// Parse manifest to get layer information
	var manifestWithLayers struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
		Config        struct {
			Size int64 `json:"size"`
		} `json:"config"`
		Layers []struct {
			Size int64 `json:"size"`
		} `json:"layers"`
	}

	if err := json.Unmarshal(manifestOutput, &manifestWithLayers); err == nil {
		// Add layer sizes
		for _, layer := range manifestWithLayers.Layers {
			totalSize += layer.Size
		}
	}

	// Format size in human-readable format
	return formatBytes(totalSize)
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	unitIndex := 0

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unitIndex])
	}
	return fmt.Sprintf("%.1f%s", size, units[unitIndex])
}

func getRegistryImages() ([]DockerImage, error) {
	// Use service name when running in Docker Compose, fallback to localhost for local development
	registryHost := os.Getenv("REGISTRY_HOST")
	if registryHost == "" {
		if _, err := os.Stat("/.dockerenv"); err == nil {
			registryHost = "registry:5000"
		} else {
			registryHost = "localhost:5000"
		}
	}

	// First, try to get the list of repositories from the registry
	cmd := exec.Command("curl", "-s", fmt.Sprintf("http://%s/v2/_catalog", registryHost))
	output, err := cmd.Output()
	if err != nil {
		// Fallback to local images
		return getLocalDockerImages()
	}

	// Parse the JSON response
	var catalog RegistryCatalog
	if err := json.Unmarshal(output, &catalog); err != nil {
		return getLocalDockerImages()
	}

	var images []DockerImage

	// For each repository, get its tags
	for _, repo := range catalog.Repositories {
		tagsCmd := exec.Command("curl", "-s", fmt.Sprintf("http://%s/v2/%s/tags/list", registryHost, repo))
		tagsOutput, err := tagsCmd.Output()
		if err != nil {
			continue
		}

		var repoTags RegistryTags
		if err := json.Unmarshal(tagsOutput, &repoTags); err != nil {
			continue
		}

		// Create an image entry for each tag
		for _, tag := range repoTags.Tags {
			imageFullName := fmt.Sprintf("%s/%s:%s", registryHost, repo, tag)

			// Try to get creation timestamp from manifest
			createdAt := getImageCreationTime(registryHost, repo, tag)

			// Try to get image size from manifest
			size := getImageSize(registryHost, repo, tag)

			images = append(images, DockerImage{
				ID:        fmt.Sprintf("registry-%s-%s", repo, tag), // Generate a pseudo-ID
				RepoTags:  []string{imageFullName},
				Size:      size,
				CreatedAt: createdAt,
			})
		}
	}

	if len(images) == 0 {
		return getLocalDockerImages()
	}

	return images, nil
}

func getLocalDockerImages() ([]DockerImage, error) {
	// Get all local Docker images with consistent timestamp format
	cmd := exec.Command("docker", "images", "--format", "{{.ID}},{{.Repository}}:{{.Tag}},{{.Size}},{{.CreatedAt}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker images: %v", err)
	}

	if len(output) == 0 {
		return []DockerImage{{
			ID:        "Not Found",
			RepoTags:  []string{"N/A"},
			Size:      "N/A",
			CreatedAt: "N/A",
		}}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return []DockerImage{{
			ID:        "Not Found",
			RepoTags:  []string{"N/A"},
			Size:      "N/A",
			CreatedAt: "N/A",
		}}, nil
	}

	var images []DockerImage
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 4 {
			// Format the creation timestamp consistently
			createdAt := parts[3]
			// If it's a relative time like "2 hours ago", try to parse it
			if strings.Contains(createdAt, " ago") {
				// For relative times, we'll keep them as-is for now
				// Docker's CreatedAt format is already human-readable
			}

			images = append(images, DockerImage{
				ID:        parts[0],
				RepoTags:  []string{parts[1]},
				Size:      parts[2],
				CreatedAt: createdAt,
			})
		}
	}

	if len(images) == 0 {
		return []DockerImage{{
			ID:        "Parse Error",
			RepoTags:  []string{"N/A"},
			Size:      "N/A",
			CreatedAt: "N/A",
		}}, nil
	}

	return images, nil
}

func ensureImageInMinikube(fullImageName string) error {
	// Check if we're running in Minikube
	if _, err := exec.Command("minikube", "status").Output(); err != nil {
		return nil // Not in Minikube, no action needed
	}

	// Pull the image to local Docker first
	pullCmd := exec.Command("docker", "pull", fullImageName)
	if err := pullCmd.Run(); err != nil {
		return err
	}

	// Load the image into Minikube
	loadCmd := exec.Command("minikube", "image", "load", fullImageName)
	if err := loadCmd.Run(); err != nil {
		return err
	}

	return nil
}

func pullFromRegistry(imageName string) error {
	// Use service name when running in Docker Compose, fallback to localhost for local development
	registryHost := os.Getenv("REGISTRY_HOST")
	if registryHost == "" {
		if _, err := os.Stat("/.dockerenv"); err == nil {
			registryHost = "registry:5000"
		} else {
			registryHost = "localhost:5000"
		}
	}
	fullImageName := fmt.Sprintf("%s/%s", registryHost, imageName)

	cmd := exec.Command("docker", "pull", fullImageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func getDockerImagesInfo() ([]DockerImage, error) {
	// Try to get images from registry first, then fallback to local
	images, err := getRegistryImages()
	if err != nil {
		return getLocalDockerImages()
	}

	return images, nil
}

func getKubernetesPodsInfo() ([]TableData, error) {
	// Try kubectl first (works in both container and host environments)
	podData, err := getPodsViaKubectl()
	if err == nil && len(podData) > 0 && podData[0].PodName != "kubectl error:" {
		return podData, nil
	}

	// Fallback to direct API calls if kubectl fails
	fmt.Printf("kubectl failed, falling back to direct API calls\n")

	// Build kubeconfig path - check environment variable first, then fallback to home
	var kubeconfig string
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfig = kubeconfigEnv
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return []TableData{{
			PodName:   "No Kubernetes cluster found",
			Namespace: "N/A",
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return []TableData{{
			PodName:   fmt.Sprintf("Config error: %v", err),
			Namespace: "N/A",
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	// Override with environment variables if provided
	if controlPlane := os.Getenv("KUBERNETES_CONTROL_PLANE"); controlPlane != "" {
		if port := os.Getenv("KUBERNETES_CONTROL_PLANE_PORT"); port != "" {
			// Check if controlPlane already has protocol
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = fmt.Sprintf("%s:%s", controlPlane, port)
			} else {
				config.Host = fmt.Sprintf("https://%s:%s", controlPlane, port)
			}
		} else {
			// Check if controlPlane already has protocol
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = controlPlane
			} else {
				config.Host = fmt.Sprintf("https://%s", controlPlane)
			}
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return []TableData{{
			PodName:   fmt.Sprintf("Client error: %v", err),
			Namespace: "N/A",
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	// Get namespace from environment or use default
	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// List pods
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []TableData{{
			PodName:   fmt.Sprintf("List error: %v", err),
			Namespace: namespace,
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	var tableData []TableData
	for _, pod := range pods.Items {
		// Calculate age
		age := time.Since(pod.CreationTimestamp.Time).Truncate(time.Second).String()

		// Calculate total restarts
		restarts := int32(0)
		for _, containerStatus := range pod.Status.ContainerStatuses {
			restarts += containerStatus.RestartCount
		}

		// Get node name
		nodeName := pod.Spec.NodeName
		if nodeName == "" {
			nodeName = "N/A"
		}

		tableData = append(tableData, TableData{
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Restarts:  fmt.Sprintf("%d", restarts),
			Age:       age,
			NodeName:  nodeName,
		})
	}

	if len(tableData) == 0 {
		return []TableData{{
			PodName:   "No pods found",
			Namespace: namespace,
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	return tableData, nil
}

func getKubernetesPodDetails(podName, namespace string) (map[string]string, error) {
	// Try kubectl first
	podDetails, err := getPodDetailsViaKubectl(podName, namespace)
	if err == nil && len(podDetails) > 0 {
		return podDetails, nil
	}

	// Fallback to direct API calls
	fmt.Printf("kubectl pod details failed, falling back to direct API calls\n")

	// Build kubeconfig path
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig not found")
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("error building config: %v", err)
	}

	// Override with environment variables if provided
	if controlPlane := os.Getenv("KUBERNETES_CONTROL_PLANE"); controlPlane != "" {
		if port := os.Getenv("KUBERNETES_CONTROL_PLANE_PORT"); port != "" {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = fmt.Sprintf("%s:%s", controlPlane, port)
			} else {
				config.Host = fmt.Sprintf("https://%s:%s", controlPlane, port)
			}
		} else {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = controlPlane
			} else {
				config.Host = fmt.Sprintf("https://%s", controlPlane)
			}
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	// Get the specific pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting pod: %v", err)
	}

	// Build detailed pod information
	details := make(map[string]string)

	// Basic information
	details["Name"] = pod.Name
	details["Namespace"] = pod.Namespace
	details["Status"] = string(pod.Status.Phase)
	details["Node"] = pod.Spec.NodeName
	details["Start Time"] = pod.Status.StartTime.Format("2006-01-02 15:04:05")
	details["Created"] = pod.CreationTimestamp.Format("2006-01-02 15:04:05")

	// Pod IP and Host IP
	details["Pod IP"] = pod.Status.PodIP
	details["Host IP"] = pod.Status.HostIP

	// Service Account
	details["Service Account"] = pod.Spec.ServiceAccountName

	// Restart Policy
	details["Restart Policy"] = string(pod.Spec.RestartPolicy)

	// DNS Policy
	details["DNS Policy"] = string(pod.Spec.DNSPolicy)

	// Labels
	if len(pod.Labels) > 0 {
		labelsStr := ""
		for k, v := range pod.Labels {
			if labelsStr != "" {
				labelsStr += ", "
			}
			labelsStr += fmt.Sprintf("%s=%s", k, v)
		}
		details["Labels"] = labelsStr
	} else {
		details["Labels"] = "None"
	}

	// Annotations count
	details["Annotations"] = fmt.Sprintf("%d annotations", len(pod.Annotations))

	// Container information
	if len(pod.Spec.Containers) > 0 {
		container := pod.Spec.Containers[0] // Show first container
		details["Container Name"] = container.Name
		details["Container Image"] = container.Image
		details["Image Pull Policy"] = string(container.ImagePullPolicy)

		// Ports
		if len(container.Ports) > 0 {
			portsStr := ""
			for _, port := range container.Ports {
				if portsStr != "" {
					portsStr += ", "
				}
				portsStr += fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol)
			}
			details["Container Ports"] = portsStr
		} else {
			details["Container Ports"] = "None"
		}

		// Resource requests and limits
		if container.Resources.Requests != nil {
			if cpu := container.Resources.Requests["cpu"]; !cpu.IsZero() {
				details["CPU Request"] = cpu.String()
			}
			if memory := container.Resources.Requests["memory"]; !memory.IsZero() {
				details["Memory Request"] = memory.String()
			}
		}
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits["cpu"]; !cpu.IsZero() {
				details["CPU Limit"] = cpu.String()
			}
			if memory := container.Resources.Limits["memory"]; !memory.IsZero() {
				details["Memory Limit"] = memory.String()
			}
		}
	}

	// Container status information
	if len(pod.Status.ContainerStatuses) > 0 {
		containerStatus := pod.Status.ContainerStatuses[0]
		details["Container Ready"] = fmt.Sprintf("%t", containerStatus.Ready)
		details["Restart Count"] = fmt.Sprintf("%d", containerStatus.RestartCount)
		details["Container ID"] = containerStatus.ContainerID

		// Last state
		if containerStatus.LastTerminationState.Terminated != nil {
			term := containerStatus.LastTerminationState.Terminated
			details["Last Exit Code"] = fmt.Sprintf("%d", term.ExitCode)
			details["Last Exit Reason"] = term.Reason
		}
	}

	// Conditions
	readyCondition := "Unknown"
	scheduledCondition := "Unknown"
	initializedCondition := "Unknown"

	for _, condition := range pod.Status.Conditions {
		switch condition.Type {
		case "Ready":
			readyCondition = string(condition.Status)
		case "PodScheduled":
			scheduledCondition = string(condition.Status)
		case "Initialized":
			initializedCondition = string(condition.Status)
		}
	}

	details["Ready Condition"] = readyCondition
	details["Scheduled Condition"] = scheduledCondition
	details["Initialized Condition"] = initializedCondition

	return details, nil
}

func getKubernetesDeployments() ([]TableData, error) {
	// Build kubeconfig path
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return []TableData{{
			PodName:   "No Kubernetes cluster found",
			Namespace: "N/A",
		}}, nil
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return []TableData{{
			PodName:   "Error connecting to cluster",
			Namespace: "N/A",
		}}, nil
	}

	// Override with environment variables if provided
	if controlPlane := os.Getenv("KUBERNETES_CONTROL_PLANE"); controlPlane != "" {
		if port := os.Getenv("KUBERNETES_CONTROL_PLANE_PORT"); port != "" {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = fmt.Sprintf("%s:%s", controlPlane, port)
			} else {
				config.Host = fmt.Sprintf("https://%s:%s", controlPlane, port)
			}
		} else {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = controlPlane
			} else {
				config.Host = fmt.Sprintf("https://%s", controlPlane)
			}
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return []TableData{{
			PodName:   "Error creating Kubernetes client",
			Namespace: "N/A",
		}}, nil
	}

	// Get namespace from environment or use default
	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// List deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		// Fall back to listing pods if deployments fail
		pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return []TableData{{
				PodName:   "Error listing deployments/pods",
				Namespace: namespace,
			}}, nil
		}

		var tableData []TableData
		for _, pod := range pods.Items {
			tableData = append(tableData, TableData{
				PodName:   pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
			})
		}
		return tableData, nil
	}

	var tableData []TableData
	for _, deployment := range deployments.Items {
		// Get deployment status
		status := "Unknown"
		if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
			status = "Ready"
		} else if deployment.Status.ReadyReplicas > 0 {
			status = "Partial"
		} else {
			status = "NotReady"
		}

		tableData = append(tableData, TableData{
			PodName:   deployment.Name, // Using PodName field for deployment name
			Namespace: deployment.Namespace,
			Status:    status,
			Restarts:  fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, *deployment.Spec.Replicas),
		})
	}

	if len(tableData) == 0 {
		return []TableData{{
			PodName:   "No deployments found",
			Namespace: namespace,
			Status:    "N/A",
		}}, nil
	}

	return tableData, nil
}

func getPodsForDeployment(deploymentName, namespace string) ([]TableData, error) {
	// Build kubeconfig path
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return []TableData{{
			PodName:   "No Kubernetes cluster found",
			Namespace: "N/A",
		}}, nil
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return []TableData{{
			PodName:   "Error connecting to cluster",
			Namespace: "N/A",
		}}, nil
	}

	// Override with environment variables if provided
	if controlPlane := os.Getenv("KUBERNETES_CONTROL_PLANE"); controlPlane != "" {
		if port := os.Getenv("KUBERNETES_CONTROL_PLANE_PORT"); port != "" {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = fmt.Sprintf("%s:%s", controlPlane, port)
			} else {
				config.Host = fmt.Sprintf("https://%s:%s", controlPlane, port)
			}
		} else {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = controlPlane
			} else {
				config.Host = fmt.Sprintf("https://%s", controlPlane)
			}
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return []TableData{{
			PodName:   "Error creating Kubernetes client",
			Namespace: "N/A",
		}}, nil
	}

	// Get the deployment first to get label selectors
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return []TableData{{
			PodName:   fmt.Sprintf("Error getting deployment: %v", err),
			Namespace: namespace,
		}}, nil
	}

	// Build label selector from deployment's match labels
	var labelSelector string
	if deployment.Spec.Selector != nil && deployment.Spec.Selector.MatchLabels != nil {
		var selectors []string
		for key, value := range deployment.Spec.Selector.MatchLabels {
			selectors = append(selectors, fmt.Sprintf("%s=%s", key, value))
		}
		labelSelector = strings.Join(selectors, ",")
	}

	// List pods with the label selector
	listOptions := metav1.ListOptions{}
	if labelSelector != "" {
		listOptions.LabelSelector = labelSelector
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
	if err != nil {
		return []TableData{{
			PodName:   fmt.Sprintf("Error listing pods: %v", err),
			Namespace: namespace,
		}}, nil
	}

	var tableData []TableData
	for _, pod := range pods.Items {
		// Calculate age
		age := time.Since(pod.CreationTimestamp.Time).Truncate(time.Second).String()

		// Calculate total restarts
		restarts := int32(0)
		for _, containerStatus := range pod.Status.ContainerStatuses {
			restarts += containerStatus.RestartCount
		}

		// Get node name
		nodeName := pod.Spec.NodeName
		if nodeName == "" {
			nodeName = "N/A"
		}

		tableData = append(tableData, TableData{
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			Restarts:  fmt.Sprintf("%d", restarts),
			Age:       age,
			NodeName:  nodeName,
		})
	}

	if len(tableData) == 0 {
		return []TableData{{
			PodName:   "No pods found for this deployment",
			Namespace: namespace,
			Status:    "N/A",
		}}, nil
	}

	return tableData, nil
}

func deployImageToPod(imageName, deploymentName, namespace string) error {
	// When running in Docker container, use kubectl through Docker socket
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return deployViaKubectl(imageName, deploymentName, namespace)
	}

	// Build kubeconfig path - check environment variable first, then fallback to home
	var kubeconfig string
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfig = kubeconfigEnv
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig not found")
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("error building config: %v", err)
	}

	// Override with environment variables if provided
	if controlPlane := os.Getenv("KUBERNETES_CONTROL_PLANE"); controlPlane != "" {
		if port := os.Getenv("KUBERNETES_CONTROL_PLANE_PORT"); port != "" {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = fmt.Sprintf("%s:%s", controlPlane, port)
			} else {
				config.Host = fmt.Sprintf("https://%s:%s", controlPlane, port)
			}
		} else {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = controlPlane
			} else {
				config.Host = fmt.Sprintf("https://%s", controlPlane)
			}
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Get the deployment
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting deployment %s: %v", deploymentName, err)
	}

	// Update the first container's image
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("deployment %s has no containers", deploymentName)
	}

	// Ensure image name includes registry if it's from our local registry
	fullImageName := imageName

	// Always ensure we have the correct registry prefix for local images
	if !strings.Contains(imageName, "localhost:5000") && !strings.Contains(imageName, "host.minikube.internal:5000") {
		// This is likely a local image that needs the registry prefix
		registryHost := "localhost:5000"
		if os.Getenv("KUBERNETES_REGISTRY_HOST") != "" {
			registryHost = os.Getenv("KUBERNETES_REGISTRY_HOST")
		} else {
			// Try to detect if we're running in Minikube
			if _, err := exec.Command("minikube", "status").Output(); err == nil {
				registryHost = "host.minikube.internal:5000"
			}
		}

		// Extract just the image name and tag from the full image name
		imageParts := strings.Split(imageName, "/")
		imageNameAndTag := imageParts[len(imageParts)-1] // Get the last part (name:tag)

		fullImageName = fmt.Sprintf("%s/%s", registryHost, imageNameAndTag)
	}

	// Ensure the image is available in Minikube if needed
	ensureImageInMinikube(fullImageName)

	// Create a copy of the deployment with updated image
	deploymentCopy := deployment.DeepCopy()
	deploymentCopy.Spec.Template.Spec.Containers[0].Image = fullImageName

	// Set image pull policy for local registry images
	// For local development, always use "Never" to avoid pulling from remote registries
	deploymentCopy.Spec.Template.Spec.Containers[0].ImagePullPolicy = "Never"

	// Update the deployment
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deploymentCopy, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error updating deployment %s: %v", deploymentName, err)
	}

	return nil
}

func deployViaKubectl(imageName, deploymentName, namespace string) error {
	// Find kubectl binary
	kubectlPath := findKubectl()

	// Prepare the full image name
	fullImageName := imageName
	if !strings.Contains(imageName, "localhost:5000") && !strings.Contains(imageName, "host.minikube.internal:5000") {
		registryHost := "localhost:5000"
		if os.Getenv("KUBERNETES_REGISTRY_HOST") != "" {
			registryHost = os.Getenv("KUBERNETES_REGISTRY_HOST")
		}
		imageParts := strings.Split(imageName, "/")
		imageNameAndTag := imageParts[len(imageParts)-1]
		fullImageName = fmt.Sprintf("%s/%s", registryHost, imageNameAndTag)
	}

	// Execute kubectl command to patch the deployment
	kubectlCmd := exec.Command(kubectlPath, "set", "image",
		fmt.Sprintf("deployment/%s", deploymentName),
		fmt.Sprintf("app=%s", fullImageName),
		"--namespace", namespace)

	// If running in container, use the fixed kubeconfig
	if _, err := os.Stat("/.dockerenv"); err == nil {
		fixKubeconfigPaths()
		kubectlCmd = exec.Command(kubectlPath, "--kubeconfig=/tmp/kubeconfig", "set", "image",
			fmt.Sprintf("deployment/%s", deploymentName),
			fmt.Sprintf("app=%s", fullImageName),
			"--namespace", namespace)
	}

	output, err := kubectlCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl command failed: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("✅ Successfully updated deployment %s with image %s\n", deploymentName, fullImageName)
	return nil
}

func createKubernetesDeployment(imageName, deploymentName, namespace string) error {
	// When running in Docker container, use kubectl through Docker socket
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return createDeploymentViaKubectl(imageName, deploymentName, namespace)
	}

	// Build kubeconfig path - check environment variable first, then fallback to home
	var kubeconfig string
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfig = kubeconfigEnv
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig not found")
	}

	// Build config from kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("error building config: %v", err)
	}

	// Override with environment variables if provided
	if controlPlane := os.Getenv("KUBERNETES_CONTROL_PLANE"); controlPlane != "" {
		if port := os.Getenv("KUBERNETES_CONTROL_PLANE_PORT"); port != "" {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = fmt.Sprintf("%s:%s", controlPlane, port)
			} else {
				config.Host = fmt.Sprintf("https://%s:%s", controlPlane, port)
			}
		} else {
			if strings.HasPrefix(controlPlane, "http://") || strings.HasPrefix(controlPlane, "https://") {
				config.Host = controlPlane
			} else {
				config.Host = fmt.Sprintf("https://%s", controlPlane)
			}
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Prepare the full image name
	fullImageName := imageName

	// Always ensure we have the correct registry prefix for local images
	if !strings.Contains(imageName, "localhost:5000") && !strings.Contains(imageName, "host.minikube.internal:5000") {
		// This is likely a local image that needs the registry prefix
		registryHost := "localhost:5000"
		if os.Getenv("KUBERNETES_REGISTRY_HOST") != "" {
			registryHost = os.Getenv("KUBERNETES_REGISTRY_HOST")
		} else {
			// Try to detect if we're running in Minikube
			if _, err := exec.Command("minikube", "status").Output(); err == nil {
				registryHost = "host.minikube.internal:5000"
			}
		}

		// Extract just the image name and tag from the full image name
		imageParts := strings.Split(imageName, "/")
		imageNameAndTag := imageParts[len(imageParts)-1] // Get the last part (name:tag)

		fullImageName = fmt.Sprintf("%s/%s", registryHost, imageNameAndTag)
	}

	// Ensure the image is available in Minikube if needed
	ensureImageInMinikube(fullImageName)

	// Create deployment specification
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": deploymentName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: fullImageName,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set image pull policy for local registry images
	// For local development, always use "Never" to avoid pulling from remote registries
	deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = "Never"

	// Create the deployment
	_, err = clientset.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		// Provide helpful error message
		errorMsg := fmt.Sprintf("error creating deployment %s: %v", deploymentName, err)

		// Add troubleshooting hints
		if strings.Contains(err.Error(), "already exists") {
			errorMsg += "\n\nTip: A deployment with this name already exists. Try using a different image name or delete the existing deployment."
		} else {
			errorMsg += fmt.Sprintf("\n\nTroubleshooting:\n1. Make sure the image exists: docker images | grep %s\n2. For Minikube, load the image: minikube image load %s\n3. Check if registry is running: curl -k https://localhost:443/v2/_catalog", imageName, fullImageName)
		}

		return fmt.Errorf(errorMsg)
	}

	return nil
}

func createDeploymentViaKubectl(imageName, deploymentName, namespace string) error {
	// Find kubectl binary
	kubectlPath := findKubectl()

	// Prepare the full image name
	fullImageName := imageName
	if !strings.Contains(imageName, "localhost:5000") && !strings.Contains(imageName, "host.minikube.internal:5000") {
		registryHost := "localhost:5000"
		if os.Getenv("KUBERNETES_REGISTRY_HOST") != "" {
			registryHost = os.Getenv("KUBERNETES_REGISTRY_HOST")
		}
		imageParts := strings.Split(imageName, "/")
		imageNameAndTag := imageParts[len(imageParts)-1]
		fullImageName = fmt.Sprintf("%s/%s", registryHost, imageNameAndTag)
	}

	// Create a temporary YAML file for the deployment
	yamlContent := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: app
        image: %s
        imagePullPolicy: Never
        ports:
        - containerPort: 80
`, deploymentName, namespace, deploymentName, deploymentName, deploymentName, fullImageName)

	// Write to temporary file
	tmpFile := "/tmp/deployment.yaml"
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create deployment YAML: %v", err)
	}

	// Execute kubectl apply
	kubectlCmd := exec.Command(kubectlPath, "apply", "-f", tmpFile)

	// If running in container, use the fixed kubeconfig
	if _, err := os.Stat("/.dockerenv"); err == nil {
		fixKubeconfigPaths()
		kubectlCmd = exec.Command(kubectlPath, "--kubeconfig=/tmp/kubeconfig", "apply", "-f", tmpFile)
	}

	output, err := kubectlCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %v\nOutput: %s", err, string(output))
	}

	fmt.Printf("✅ Successfully created deployment %s with image %s\n", deploymentName, fullImageName)
	return nil
}

func fixKubeconfigPaths() {
	// When running in Docker, the kubeconfig paths need to be adjusted
	// since we mount ~/.minikube to /root/.minikube
	if _, err := os.Stat("/.dockerenv"); err == nil {
		kubeconfigPath := "/root/.kube/config"
		if _, err := os.Stat(kubeconfigPath); err == nil {
			// Read the kubeconfig
			content, err := os.ReadFile(kubeconfigPath)
			if err == nil {
				// Replace paths from /home/nova to /root
				newContent := strings.ReplaceAll(string(content), "/home/nova/.minikube", "/root/.minikube")

				// Write back the modified kubeconfig
				tempKubeconfig := "/tmp/kubeconfig"
				err = os.WriteFile(tempKubeconfig, []byte(newContent), 0644)
				if err == nil {
					os.Setenv("KUBECONFIG", tempKubeconfig)
					fmt.Printf("DEBUG: KUBECONFIG set to %s\n", tempKubeconfig)
				} else {
					fmt.Printf("DEBUG: Failed to write kubeconfig: %v\n", err)
				}
			} else {
				fmt.Printf("DEBUG: Failed to read kubeconfig: %v\n", err)
			}
		} else {
			fmt.Printf("DEBUG: Kubeconfig not found at %s\n", kubeconfigPath)
		}
	}
}

func testConnections() {
	fmt.Println("Testing database connection...")

	// Fix kubeconfig paths for container environment
	fixKubeconfigPaths()

	// Test database connection
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	if cfg.User == "" {
		cfg.User = "mysql"
	}
	cfg.Passwd = os.Getenv("MYSQL_ROOT_PASSWORD")
	if cfg.Passwd == "" {
		cfg.Passwd = "mysql_password"
	}
	cfg.Net = "tcp"

	// Use service name when running in Docker Compose
	dbHost := os.Getenv("MYSQL_HOST")
	if dbHost == "" {
		if _, err := os.Stat("/.dockerenv"); err == nil {
			dbHost = "db:3306"
		} else {
			dbHost = "127.0.0.1:3307"
		}
	}
	cfg.Addr = dbHost

	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	if cfg.DBName == "" {
		cfg.DBName = "images"
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Database ping failed: %v\n", err)
		return
	}

	fmt.Println("✅ Database connection successful!")

	// Test GitHub connection
	fmt.Println("Testing GitHub connection...")
	client := github.NewClient(nil).WithAuthToken(os.Getenv("gitHubAuth"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")

	if owner == "" || repo == "" {
		fmt.Println("⚠️  GitHub credentials not configured (GITHUB_OWNER or GITHUB_REPO missing)")
	} else {
		_, _, err := client.Repositories.ListCommits(context.Background(), owner, repo, &github.CommitsListOptions{
			SHA:         "master",
			ListOptions: github.ListOptions{Page: 1, PerPage: 1},
		})
		if err != nil {
			fmt.Printf("❌ GitHub connection failed: %v\n", err)
		} else {
			fmt.Println("✅ GitHub connection successful!")
		}
	}

	// Test Docker registry connection
	fmt.Println("Testing Docker registry connection...")
	registryHost := os.Getenv("REGISTRY_HOST")
	if registryHost == "" {
		if _, err := os.Stat("/.dockerenv"); err == nil {
			registryHost = "registry:5000"
		} else {
			registryHost = "localhost:5000"
		}
	}

	cmd := exec.Command("curl", "-s", fmt.Sprintf("http://%s/v2/_catalog", registryHost))
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ Registry connection failed: %v\n", err)
	} else {
		fmt.Println("✅ Registry connection successful!")
		fmt.Printf("Registry catalog: %s\n", string(output))
	}

	// Test Kubernetes connection
	fmt.Println("Testing Kubernetes connection...")
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// In container - test kubectl access
		fixKubeconfigPaths()
		kubectlCmd := exec.Command("kubectl", "--kubeconfig=/tmp/kubeconfig", "get", "pods", "--all-namespaces")
		output, err := kubectlCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("kubectl output: %s\n", string(output))
			if strings.Contains(string(output), "dial tcp") && strings.Contains(string(output), "i/o timeout") {
				fmt.Println("⚠️  Kubernetes API not accessible from container (networking limitation)")
				fmt.Println("💡 kubectl fallback will be used in the TUI")
			} else {
				fmt.Printf("❌ kubectl configuration error\n")
			}
		} else {
			fmt.Println("✅ kubectl connection successful!")
			// Show first few lines of output
			lines := strings.Split(string(output), "\n")
			if len(lines) > 3 {
				fmt.Printf("Found pods: %s\n", strings.Join(lines[1:4], ", "))
			}
		}
	} else {
		// On host - test direct API access
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}

		if kubeconfig == "" {
			fmt.Println("⚠️  KUBECONFIG environment variable not set")
		} else if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
			fmt.Printf("❌ Kubeconfig not found at: %s\n", kubeconfig)
		} else {
			// Try to list pods
			config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				fmt.Printf("❌ Kubernetes config error: %v\n", err)
			} else {
				clientset, err := kubernetes.NewForConfig(config)
				if err != nil {
					fmt.Printf("❌ Kubernetes client error: %v\n", err)
				} else {
					pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{Limit: 1})
					if err != nil {
						fmt.Printf("❌ Kubernetes API error: %v\n", err)
					} else {
						fmt.Printf("✅ Kubernetes connection successful! Found %d pods in default namespace\n", len(pods.Items))
					}
				}
			}
		}
	}

	fmt.Println("🎉 All connection tests completed!")
}

func getPodsViaKubectl() ([]TableData, error) {
	// Find kubectl binary
	kubectlPath := findKubectl()

	// Use kubectl to get pod information
	kubectlCmd := exec.Command(kubectlPath, "get", "pods", "--all-namespaces",
		"-o", "jsonpath={range .items[*]}{.metadata.name},{.metadata.namespace},{.status.phase},{.status.containerStatuses[0].restartCount},{.metadata.creationTimestamp}{'\\n'}{end}")

	// If running in container, use the fixed kubeconfig
	if _, err := os.Stat("/.dockerenv"); err == nil {
		fixKubeconfigPaths()
		kubectlCmd = exec.Command(kubectlPath, "--kubeconfig=/tmp/kubeconfig", "get", "pods", "--all-namespaces",
			"-o", "jsonpath={range .items[*]}{.metadata.name},{.metadata.namespace},{.status.phase},{.status.containerStatuses[0].restartCount},{.metadata.creationTimestamp}{'\\n'}{end}")
	}

	output, err := kubectlCmd.CombinedOutput()
	if err != nil {
		// Provide helpful error message for networking issues
		errorMsg := "Kubernetes connection error"
		if strings.Contains(string(output), "dial tcp") && strings.Contains(string(output), "i/o timeout") {
			errorMsg = "Cannot reach Kubernetes cluster. Run on host or check Minikube status."
		} else if strings.Contains(string(output), "Unable to connect to the server") {
			errorMsg = "Kubernetes cluster not accessible. Check Minikube status."
		}
		return []TableData{{
			PodName:   errorMsg,
			Namespace: "N/A",
			Status:    "Error",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	// Parse the output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return []TableData{{
			PodName:   "No pods found",
			Namespace: "N/A",
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	var tableData []TableData
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			// Calculate age from creation timestamp
			age := "N/A"
			if len(parts) >= 5 {
				if t, err := time.Parse(time.RFC3339, parts[4]); err == nil {
					age = time.Since(t).Truncate(time.Second).String()
				}
			}

			restartCount := "0"
			if len(parts) >= 4 && parts[3] != "" {
				restartCount = parts[3]
			}

			tableData = append(tableData, TableData{
				PodName:   parts[0],
				Namespace: parts[1],
				Status:    parts[2],
				Restarts:  restartCount,
				Age:       age,
			})
		}
	}

	if len(tableData) == 0 {
		return []TableData{{
			PodName:   "No pods found",
			Namespace: "N/A",
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}, nil
	}

	return tableData, nil
}

func getPodDetailsViaKubectl(podName, namespace string) (map[string]string, error) {
	// Find kubectl binary
	kubectlPath := findKubectl()

	// Use kubectl to get detailed pod information
	kubectlCmd := exec.Command(kubectlPath, "get", "pod", podName, "-n", namespace, "-o", "yaml")

	// If running in container, use the fixed kubeconfig
	if _, err := os.Stat("/.dockerenv"); err == nil {
		fixKubeconfigPaths()
		kubectlCmd = exec.Command(kubectlPath, "--kubeconfig=/tmp/kubeconfig", "get", "pod", podName, "-n", namespace, "-o", "yaml")
	}

	output, err := kubectlCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl get pod failed: %v\nOutput: %s", err, string(output))
	}

	// Parse the YAML output to extract key information
	details := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	details["Name"] = podName
	details["Namespace"] = namespace

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "phase:") {
			details["Status"] = strings.TrimSpace(strings.TrimPrefix(line, "phase:"))
		}
		if strings.Contains(line, "nodeName:") {
			details["Node"] = strings.TrimSpace(strings.TrimPrefix(line, "nodeName:"))
		}
		if strings.Contains(line, "restartCount:") {
			details["Restarts"] = strings.TrimSpace(strings.TrimPrefix(line, "restartCount:"))
		}
		if strings.Contains(line, "image:") {
			if _, exists := details["Image"]; !exists {
				details["Image"] = strings.TrimSpace(strings.TrimPrefix(line, "image:"))
			}
		}
	}

	return details, nil
}

func findKubectl() string {
	// Try multiple possible kubectl locations
	possiblePaths := []string{
		os.Getenv("HOME") + "/bin/kubectl", // User location (highest priority)
		"/usr/local/bin/kubectl",           // System location
		"/usr/bin/kubectl",                 // Alternative system location
		"./kubectl",                        // Current directory
		"kubectl",                          // Rely on PATH
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// Test if it's executable and works
			cmd := exec.Command(path, "version", "--client")
			if err := cmd.Run(); err == nil {
				return path
			}
		}
	}

	// Fallback to PATH
	return "kubectl"
}

func isTTYAvailable() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func main() {
	// Check if TEST_MODE environment variable is set (for non-interactive testing)
	if os.Getenv("TEST_MODE") == "true" {
		testConnections()
		return
	}

	// Check if DOCKER_BUILD environment variable is set
	if os.Getenv("DOCKER_BUILD") == "true" {
		fmt.Println("🐳 Building Docker image...")

		cmd := exec.Command("docker", "build", "-t", "local-container-registry", ".")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			log.Fatalf("❌ Docker build failed: %v", err)
		}

		fmt.Println("✅ Docker image built successfully!")
		fmt.Println("🚀 You can now run: docker run --rm -it local-container-registry")
		return
	}

	// Fix kubeconfig paths for container environment (do this early)
	fixKubeconfigPaths()

	// Capture connection properties for the MySQL database
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	if cfg.User == "" {
		cfg.User = "mysql"
	}
	cfg.Passwd = os.Getenv("MYSQL_ROOT_PASSWORD")
	if cfg.Passwd == "" {
		cfg.Passwd = "mysql_password"
	}
	cfg.Net = "tcp"

	// Use service name when running in Docker Compose, fallback to localhost for local development
	dbHost := os.Getenv("MYSQL_HOST")
	if dbHost == "" {
		// Check if we're running in Docker by looking for the db service
		if _, err := os.Stat("/.dockerenv"); err == nil {
			dbHost = "db:3306"
		} else {
			dbHost = "127.0.0.1:3307"
		}
	}
	cfg.Addr = dbHost

	cfg.DBName = os.Getenv("MYSQL_DATABASE")
	if cfg.DBName == "" {
		cfg.DBName = "images"
	}

	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	var (
		Green  = "\033[32m"
		Reset  = "\033[0m"
		Yellow = "\033[33m"
	)

	fmt.Println("------------------------------------------------------------------------------------------------")
	println(Yellow + "Logging into Github..." + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")

	client := github.NewClient(nil).WithAuthToken(os.Getenv("GITHUB_AUTH_TOKEN"))
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")

	branch := "master"
	// Get multiple commits instead of just one
	commits, _, err := client.Repositories.ListCommits(context.Background(), owner, repo, &github.CommitsListOptions{
		SHA: branch,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 10, // Get last 10 commits
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	println(Green + "Logged into Github" + Reset)
	fmt.Println("------------------------------------------------------------------------------------------------")
	fmt.Printf("Found %d commits on master branch\n", len(commits))
	if len(commits) > 0 {
		fmt.Printf("Latest commit: %s\n", commits[0].GetCommit().GetMessage())
		fmt.Printf("SHA: %s\n", commits[0].GetSHA())
	}

	// fmt.Printf("Owner: %v\n", repoData.GetOwner())
	// fmt.Printf("repo: %+v\n", repoData.GetFullName())
	// fmt.Printf("Date: %s\n", commit.GetCommit().GetAuthor().GetDate())
	// fmt.Printf("UpdatedAt: %v\n", repoData.GetUpdatedAt())
	// fmt.Printf("Author: %s\n", commit.GetCommit().GetAuthor().GetName())
	// fmt.Printf("ID: %d\n", repoData.GetID())
	// 	fmt.Printf("PushedAt: %v\n", repoData.GetPushedAt())
	// 	Create a code break
	// 	fmt.Println("------------------------------------------------------------------------------------------------")
	// 	fmt.Printf("Size: %d\n", repoData.GetSize())
	// 	fmt.Printf("CommitsURL: %s\n", repoData.GetCommitsURL())
	// 	fmt.Printf("FullName: %s\n", repoData.GetFullName())
	// 	fmt.Printf("Name: %s\n", repoData.GetName())
	// 	fmt.Printf("Description: %s\n", repoData.GetDescription())
	// 	fmt.Printf("BranchesURL: %s\n", repoData.GetBranchesURL())
	// 	fmt.Printf("CreatedAt: %v\n", repoData.GetCreatedAt())
	// 	fmt.Printf("URL: %s\n", repoData.GetURL())
	// 	fmt.Println("Logged into Github")

	// Process each commit for database insertion
	for _, commit := range commits {
		commitMessage := commit.GetCommit().GetMessage()
		fmt.Printf("Processing commit: %s\n", commitMessage)

		// Insert into MySQL database
		_, err = db.Exec("INSERT INTO images (PR_Description) VALUES (?)", commitMessage)
		if err != nil {
			// Silently continue on database errors during TUI operation
		}
	}

	// Get Docker images information
	dockerImages, err := getDockerImagesInfo()
	if err != nil {
		dockerImages = []DockerImage{{
			ID:        "Error",
			RepoTags:  []string{"N/A"},
			Size:      "N/A",
			CreatedAt: "N/A",
		}}
	}

	// Start TUI with collected data from all commits
	var gitTableData []TableData
	for _, commit := range commits {
		commitMessage := commit.GetCommit().GetMessage()

		// Get PushedAt from individual commit date
		pushedAt := "N/A"
		if commit.GetCommit() != nil && commit.GetCommit().GetAuthor() != nil {
			pushedAt = commit.GetCommit().GetAuthor().GetDate().Format("2006-01-02 15:04:05")
		}

		gitTableData = append(gitTableData, TableData{
			CommitSHA:     commit.GetSHA(),
			PRDescription: commitMessage,
			PushedAt:      pushedAt,
		})
	}

	// Create Docker table data from actual Docker images
	var dockerTableData []TableData
	for _, dockerImg := range dockerImages {
		imageID := dockerImg.ID
		if len(imageID) > 20 {
			imageID = imageID[:20] // Show more of the ID to match column width
		}

		imageTag := "N/A"
		if len(dockerImg.RepoTags) > 0 && dockerImg.RepoTags[0] != "<none>:<none>" {
			imageTag = dockerImg.RepoTags[0]
		}

		imageSize := dockerImg.Size
		if dockerImg.Size == "" || dockerImg.Size == "N/A" {
			imageSize = "N/A"
		}

		dockerTableData = append(dockerTableData, TableData{
			ImageID:   imageID,
			ImageSize: imageSize,
			ImageTag:  imageTag,
			CreatedAt: dockerImg.CreatedAt,
		})
	}

	// Get Kubernetes pods information
	kubernetesData, err := getKubernetesPodsInfo()
	if err != nil {
		kubernetesData = []TableData{{
			PodName:   "Error",
			Namespace: "N/A",
			Status:    "N/A",
			Restarts:  "N/A",
			Age:       "N/A",
		}}
	}

	// Disable logging before starting TUI to prevent interference
	disableLogging()

	startTUI(gitTableData, dockerTableData, kubernetesData)
}

// I need to insert git commits into the mysql database
