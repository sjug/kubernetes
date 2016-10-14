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
	"fmt"
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

	var c *client.Client
	BeforeEach(func() {
		c = f.Client
	})

	readConfig := func() []framework.ClusterLoaderType {
		// Read in configuration settings
		project := framework.TestContext.ClusterLoader.Projects
		framework.Logf("Loaded project config: %v, length: %v, type %T) {", project, len(project), project)
		return project
	}

	project := readConfig()
	It(fmt.Sprintf("running config file: %v", project), func() {
		defer GinkgoRecover()

		// Helper func to make fq path to Kube config file
		mkpath := func(file string) string {
			return filepath.Join(framework.TestContext.RepoRoot, "examples/cluster-loader", file)
		}

		// Get number of namespaces defined in Cluster Loader config
		nsNum := len(project)
		var namespaces = make([]*api.Namespace, nsNum)

		// Create namespaces as defined in Cluster Loader config
		for i := range namespaces {
			var err error
			nsName := framework.TestContext.ClusterLoader.Projects[i].BaseName
			namespaces[i], err = f.CreateNamespace(nsName, nil)
			Expect(err).NotTo(HaveOccurred())
			framework.Logf("Created new namespace: %v, NS: %d/%d", nsName, i+1, nsNum)
		}

		// Create all pods from YAML defined in Cluster Loader config
		for i, ns := range namespaces {
			templateFilename := framework.TestContext.ClusterLoader.Projects[i].Templates[0].File
			framework.Logf("Template filename is: %v", templateFilename)
			podYAML := mkpath(templateFilename)
			framework.Logf("Full config path is: %v", podYAML)
			framework.RunKubectlOrDie("create", "-f", podYAML, getNsCmdFlag(ns))
		}

		// Wait for pods to be running
		for _, ns := range namespaces {
			label := labels.SelectorFromSet(labels.Set(map[string]string{"purpose": "test"}))
			err := framework.WaitForPodsWithLabelRunning(c, ns.Name, label)
			Expect(err).NotTo(HaveOccurred())
			framework.Logf("All pods running.")
		}
	})
})
