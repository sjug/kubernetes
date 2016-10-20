/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = framework.KubeDescribe("Cluster Loader [Performance] [Slow] [Disruptive]", func() {
	f := framework.NewDefaultFramework("cluster-loader")
	defer GinkgoRecover()

	var c *client.Client
	BeforeEach(func() {
		c = f.Client
	})

	readConfig := func() ([]framework.ClusterLoaderType, int) {
		// Read in configuration settings
		project := framework.TestContext.ClusterLoader.Projects
		if framework.TestContext.ClusterLoader.Delete == false {
			framework.TestContext.DeleteNamespace = false
		}

		return project, len(project)
	}

	project, nsNum := readConfig()
	if nsNum < 1 {
		framework.Failf("invalid config file.\nFile: %v", project)
	}

	It(fmt.Sprintf("running config file: %v, length: %v, type %T", project, nsNum, project), func() {
		// Helper func to make fq path to Kube config file
		mkpath := func(file string) string {
			return filepath.Join(framework.TestContext.RepoRoot, "examples/cluster-loader", file)
		}

		// Get number of namespaces defined in Cluster Loader config
		var namespaces = make([]*api.Namespace, nsNum)

		// Create namespaces as defined in Cluster Loader config
		for i := range namespaces {
			var err error
			nsName := framework.TestContext.ClusterLoader.Projects[i].BaseName
			namespaces[i], err = f.CreateNamespace(nsName, nil)
			Expect(err).NotTo(HaveOccurred())
			framework.Logf("%d/%d : Created new namespace: %v", i+1, nsNum, nsName)
		}

		// Create all pods from YAML defined in Cluster Loader config
		for i, ns := range namespaces {
			var templateFilename, baseName, run string
			var numObjects int

			// Try to see what config object exists
			if len(framework.TestContext.ClusterLoader.Projects[i].Templates) > 0 {
				templateFilename = framework.TestContext.ClusterLoader.Projects[i].Templates[0].File
				run = "template"
			} else if len(framework.TestContext.ClusterLoader.Projects[i].Pods) > 0 {
				templateFilename = framework.TestContext.ClusterLoader.Projects[i].Pods[0].File
				numObjects = framework.TestContext.ClusterLoader.Projects[i].Pods[0].Number
				baseName = framework.TestContext.ClusterLoader.Projects[i].Pods[0].Basename
				run = "pod"
			}

			// Handle an empty filename.
			if templateFilename == "" {
				framework.Failf("No template file defined!")
			}

			// Debugging mostly
			framework.Logf("Template filename is: %v", templateFilename)
			podYAML := mkpath(templateFilename)
			framework.Logf("Full config path is: %v", podYAML)

			// Decide how to create objects
			switch run {
			case "template":
				// Templates have several objects in one file
				framework.RunKubectlOrDie("create", "-f", podYAML, getNsCmdFlag(ns))
				framework.Logf("%d/%d : Created template ", i+1, nsNum)
			case "pod":
				// In order to modify values in config, we load them into struct
				config, err := ioutil.ReadFile(podYAML)
				if err != nil {
					framework.Failf("Cant read config file. Error: %v", err)
				}

				var configJSON api.Pod
				err = json.Unmarshal(config, &configJSON)
				framework.Logf("The loaded config file is: %+v", configJSON.Spec.Containers)
				f.CreatePods(baseName, ns.Name, configJSON.Spec, numObjects)
			}
		}

		// Wait for pods to be running
		for _, ns := range namespaces {
			label := labels.SelectorFromSet(labels.Set(map[string]string{"purpose": "test"}))
			err := framework.WaitForPodsWithLabelRunning(c, ns.Name, label)
			Expect(err).NotTo(HaveOccurred())
			framework.Logf("All pods running in namespace %s.", ns.Name)
		}
	})
})
