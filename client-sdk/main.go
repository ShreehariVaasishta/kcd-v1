package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource" // Updated import path
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Rest of the code...

var (
	kubeconfig     string
	buildName      string
	buildNamespace string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", getKubeconfigPath(), "Path to the kubeconfig file")
	flag.StringVar(&buildName, "build", "", "Name of the build")
	flag.StringVar(&buildNamespace, "namespace", "scicd", "Namespace of the build")
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

	// Create the namespace if it doesn't exist
	err = createNamespace(clientset, buildNamespace)
	if err != nil {
		log.Fatal(err)
	}

	// Create the ConfigMap
	// err = createConfigMap(clientset, buildName, buildNamespace)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Create go configmap
	// err = createGoConfigMap(clientset, buildNamespace)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Create the pod
	err = createBuildPod(clientset, buildName, buildNamespace)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Build pod created successfully")
}

func createBuildPod(clientset *kubernetes.Clientset, buildName, buildNamespace string) error {
	// Create pod specification
	pod := &corev1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      buildName,
			Namespace: buildNamespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "build-container",
					Image: "shreeharivl/kcd:0.1",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "phases-volume",
							MountPath: "/phases",
						},
					},
					Command: []string{"./main"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
			Volumes: []corev1.Volume{
				{
					Name: "phases-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: buildName,
							},
						},
					},
				},
				{
					Name: "phases-volume2",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "goconfigmap",
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
	// if err != nil {
	// return err
	// }

	// Create the pod
	_, err = clientset.CoreV1().Pods(buildNamespace).Create(context.TODO(), pod, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func createConfigMap(clientset *kubernetes.Clientset, buildName, buildNamespace string) error {
	// Create ConfigMap data
	data := make(map[string]string)
	data["run-phases.sh"] = `#!/bin/bash

set -e

# Download phase
echo ">>> Inside the pod"

echo ">>> Creating a directory"
mkdir random-folder

echo ">>> Going into created folder"
cd random-folder

echo ">>> Copying files/folders"
# Get the source directory
source_directory="/phases"

# Get the destination directory
destination_directory="."

# Copy the source directory to the destination directory
cp -r $source_directory $destination_directory

echo ">>> Files/Folders Current directory"
ls .

echo ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>"
echo ">>> Copying runner"
echo $(ls)
echo "Hello"


echo ">>> Files/Folders Current directory"
# Get the current directory
current_directory=$(pwd)

# Print all files in the current directory
for file in $current_directory/*; do
  echo $file
done

# Print all folders in the current directory
for folder in $current_directory/*/; do
  echo $folder
done

sleep 180
`

	// Create the ConfigMap object
	configMap := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      buildName,
			Namespace: buildNamespace,
		},
		Data: data,
	}
	// check if ConfigMap exists
	_, err := clientset.CoreV1().ConfigMaps(buildNamespace).Get(context.TODO(), configMap.Name, metaV1.GetOptions{})
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

func createGoConfigMap(clientset *kubernetes.Clientset, buildNamespace string) error {
	buildName := "goconfigmap"
	// Check if the ConfigMap already exists
	_, err := clientset.CoreV1().ConfigMaps(buildNamespace).Get(context.TODO(), buildName, metaV1.GetOptions{})
	if err == nil {
		// ConfigMap already exists, update it with new data

		// Read the Go program file
		// goProgramPath := filepath.Join("phases", "phases") // Update with the path to the Go program relative to the root of your project
		goProgramContent, err := ioutil.ReadFile("runner.go")

		if err != nil {
			return err
		}

		// Create ConfigMap data
		data := make(map[string]string)
		data["main.go"] = string(goProgramContent)

		// Update the ConfigMap with new data
		configMap := &corev1.ConfigMap{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      buildName,
				Namespace: buildNamespace,
			},
			Data: data,
		}

		_, err = clientset.CoreV1().ConfigMaps(buildNamespace).Update(context.TODO(), configMap, metaV1.UpdateOptions{})
		if err != nil {
			return err
		}

		fmt.Printf("ConfigMap %s updated successfully\n", buildName)
		return nil
	} else if !k8serrors.IsNotFound(err) {
		// An error occurred while retrieving the ConfigMap
		return err
	}

	// ConfigMap doesn't exist, create it

	// Read the Go program file
	goProgramPath := filepath.Join("phases", "phases.go") // Update with the path to the Go program relative to the root of your project
	goProgramContent, err := ioutil.ReadFile(goProgramPath)
	if err != nil {
		return err
	}

	// Create ConfigMap data
	data := make(map[string]string)
	data["main.go"] = string(goProgramContent)

	// Create the ConfigMap object
	configMap := &corev1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      buildName,
			Namespace: buildNamespace,
		},
		Data: data,
	}

	// Create the ConfigMap
	_, err = clientset.CoreV1().ConfigMaps(buildNamespace).Create(context.TODO(), configMap, metaV1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("ConfigMap %s created successfully\n", buildName)
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
