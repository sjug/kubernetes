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
	"strconv"

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

	project := framework.TestContext.ClusterLoader.Projects
	if len(project) < 1 {
		framework.Failf("invalid config file.\nFile: %v", project)
	}

	It(fmt.Sprintf("running config file: %v, length: %v, type %T", project, len(project), project), func() {
		var namespaces []*api.Namespace
		for _, p := range project {
			for j := 0; j < p.Number; j++ {
				// Create namespaces as defined in Cluster Loader config
				nsName := p.Basename + strconv.Itoa(j)
				ns, err := f.CreateNamespace(nsName, nil)
				Expect(err).NotTo(HaveOccurred())
				framework.Logf("%d/%d : Created new namespace: %v", j+1, p.Number, nsName)
				namespaces = append(namespaces, ns)

				// How about we create some templates
				for _, v := range p.Templates {
					createTemplate(mkPath(v.File), v.Number, v.Basename, ns)
				}
				// This is too familiar, create pods
				for _, v := range p.Pods {
					parsePods(f, mkPath(v.File), v.Number, v.Basename, ns)
				}
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

func mkPath(file string) string {
	// Handle an empty filename.
	if file == "" {
		framework.Failf("No template file defined!")
	}
	return filepath.Join(framework.TestContext.RepoRoot, "examples/", file)
}

// TODO: Can only create one template per namespace, no way to make duplicates?
func createTemplate(podYAML string, numObjects int, baseName string, ns *api.Namespace) {
	framework.RunKubectlOrDie("create", "-f", podYAML, getNsCmdFlag(ns))
	framework.Logf("1/%d : Created template %s", numObjects, baseName)
}

func parsePods(f *framework.Framework, podYAML string, numObjects int, baseName string, ns *api.Namespace) {
	config, err := ioutil.ReadFile(podYAML)
	if err != nil {
		framework.Failf("Cant read config file. Error: %v", err)
	}

	var configJSON api.Pod
	err = json.Unmarshal(config, &configJSON)
	framework.Logf("The loaded config file is: %+v", configJSON.Spec.Containers)
	f.CreatePods(baseName, ns.Name, configJSON.Spec, numObjects)
}
