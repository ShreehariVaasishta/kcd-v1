package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource" // Updated import path
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Phases configs
type PhasesConfig struct {
	BuildPhase
	ArtifactsPhase
	FinalizePhase
}

type BuildPhase struct {
	Build []string `json:"build"`
}

type ArtifacsNestedPhase struct {
	LocalTargetDir  string `json:"local_target_dir"`
	RemoteTargetDir string `json:"remote_target_dir"`
}
type ArtifactsPhase struct {
	Artifacts ArtifacsNestedPhase `json:"artifacts"`
}

type FinalizePhase struct {
	Finalize []string `json:"finalize"`
}

// Pod Specific Configs
type PodConfig struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type JsonConfigStruct struct {
	PodCfg PodConfig    `json:"pod"`
	Phases PhasesConfig `json:"phases"`
}

// Rest of the code...

var (
	// Kubernetes
	kubeconfig     string
	buildName      string
	buildNamespace string

	// Json
	jsonConfigPath string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", getKubeconfigPath(), "Path to the kubeconfig file")
	flag.StringVar(&buildName, "build", "", "Name of the build")
	flag.StringVar(&buildNamespace, "namespace", "scicd", "Namespace of the build")

	// Add a new flag for jsonConfigPath
	flag.StringVar(&jsonConfigPath, "jsonConfig", "", "Path to the JSON config file")

	flag.Parse()
}

func main() {
	// Build clientset
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Check if jsonConfigPath is provided
	if jsonConfigPath == "" {
		log.Fatal("jsonConfigPath not provided")
		return
	}
	jsonContents, err := readJsonFile(jsonConfigPath)
	log.Println(jsonContents)
	if err != nil {
		log.Fatal(err)
	}

	// Create the namespace if it doesn't exist
	err = createNamespace(clientset, buildNamespace)
	if err != nil {
		log.Fatal(err)
	}
	// Create the Json ConfigMap
	err = createConfigMap(clientset, buildName, buildNamespace, jsonContents.Phases, "config.json")

	if err != nil {
		log.Fatal(err)
	}
	// Create the pod
	// err = createBuildPod(clientset, jsonContents, buildName, buildNamespace)
	err = createBuildPod(clientset, jsonContents.PodCfg, buildName, buildNamespace)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Build pod created successfully")
}

func readJsonFile(filePath string) (JsonConfigStruct, error) {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return JsonConfigStruct{}, err
	}
	var jsonconfig JsonConfigStruct
	err = json.Unmarshal(data, &jsonconfig)

	if err != nil {
		fmt.Println(err)
		return JsonConfigStruct{}, err
	}
	return jsonconfig, nil
}

func createBuildPod(clientset *kubernetes.Clientset, jsonPod PodConfig, buildName, buildNamespace string) error {
	// Create pod specification
	fmt.Println(">> PodCPU", jsonPod.CPU)
	fmt.Println(">> PodMemory", jsonPod.Memory)
	pod := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      buildName,
			Namespace: buildNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "build-container",
					Image:           "shreeharivl/kcd:0.2",
					ImagePullPolicy: "Always",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(jsonPod.CPU),
							corev1.ResourceMemory: resource.MustParse(jsonPod.Memory),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "user-config-volume",
							MountPath: "/config",
						},
					},
					Command: []string{"./main"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: "user-config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: buildName,
							},
						},
					},
				},
			},
		},
	}

	// Create the pod
	err := clientset.CoreV1().Pods(buildNamespace).Delete(context.TODO(), pod.Name, metaV1.DeleteOptions{})
	time.Sleep(120)
	if err != nil {
		return err
	}

	// Create the pod
	_, err = clientset.CoreV1().Pods(buildNamespace).Create(context.TODO(), pod, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// func createConfigMap(clientset *kubernetes.Clientset, buildName, buildNamespace string, configMapFileContents string, configMapFileName string) error {
func createConfigMap(clientset *kubernetes.Clientset, buildName, buildNamespace string, configMapFileContents PhasesConfig, configMapFileName string) error {

	// Create ConfigMap data
	config_data, err := json.Marshal(configMapFileContents)
	if err != nil {
		fmt.Println("Unable to marshal json")
		log.Fatal(err)
	}
	data := make(map[string]string)
	data[configMapFileName] = string(config_data)

	// Create the ConfigMap object
	configMap := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      buildName,
			Namespace: buildNamespace,
		},
		Data: data,
	}
	// check if ConfigMap exists
	_, err = clientset.CoreV1().ConfigMaps(buildNamespace).Get(context.TODO(), configMap.Name, metaV1.GetOptions{})
	if err == nil {
		fmt.Println("ConfigMap Exists")
		clientset.CoreV1().ConfigMaps(buildNamespace).Delete(context.TODO(), configMap.Name, metaV1.DeleteOptions{})
	}
	// Create the ConfigMap
	_, err = clientset.CoreV1().ConfigMaps(buildNamespace).Create(context.TODO(), configMap, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func createNamespace(clientset *kubernetes.Clientset, namespace string) error {
	// Check if the namespace already exists
	_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), namespace, metaV1.GetOptions{})
	if err == nil {
		// Namespace already exists, no need to create
		return nil
	}

	// Create the namespace
	ns := &corev1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Namespace %s created successfully\n", namespace)
	return nil
}

func getKubeconfigPath() string {
	home := homedir.HomeDir()
	return fmt.Sprintf("%s/.kube/config", home)
}
