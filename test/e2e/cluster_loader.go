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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

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
	tuningSets := framework.TestContext.ClusterLoader.TuningSets
	if len(project) < 1 {
		framework.Failf("invalid config file.\nFile: %v", project)
	}

	It(fmt.Sprintf("running config file: %v, length: %v, type %T", project, len(project), project), func() {
		var namespaces []*api.Namespace
		//totalPods := 0 // Keep track of how many pods for stepping
		for _, p := range project {
			// Find tuning if we have it
			tuning := getTuningSet(tuningSets, p.Tuning)
			framework.Logf("Our tuning set is: %v", tuning)
			for j := 0; j < p.Number; j++ {
				// Create namespaces as defined in Cluster Loader config
				nsName := p.Basename + strconv.Itoa(j)
				ns, err := f.CreateNamespace(nsName, nil)
				Expect(err).NotTo(HaveOccurred())
				framework.Logf("%d/%d : Created new namespace: %v", j+1, p.Number, nsName)
				namespaces = append(namespaces, ns)

				// How about we create some templates
				for _, v := range p.Templates {
					createTemplate(v.Basename, ns, mkPath(v.File), v.Number, tuning)
				}
				// This is too familiar, create pods
				for _, v := range p.Pods {
					config := parsePods(mkPath(v.File))
					f.CreatePods(v.Basename, ns.Name, config.Spec, v.Number, tuning)
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

func getTuningSet(tuningSets []framework.TuningSetType, podTuning string) (tuning *framework.TuningSetType) {
	if podTuning != "" {
		// Interate through defined tuningSets
		for _, ts := range tuningSets {
			// If we have a matching tuningSet keep it
			if ts.Name == podTuning {
				tuning = &ts
				return
			}
		}
		framework.Failf("No pod tuning found for: %s", podTuning)
	}
	return nil
}

func mkPath(file string) string {
	// Handle an empty filename.
	if file == "" {
		framework.Failf("No template file defined!")
	}
	return filepath.Join(framework.TestContext.RepoRoot, "examples/", file)
}

func createTemplate(baseName string, ns *api.Namespace, yaml string, numObjects int, tuning *framework.TuningSetType) {
	// Try to read the file
	content, err := ioutil.ReadFile(yaml)
	if err != nil {
		framework.Failf("Error %s", err)
	}

	// ${IDENTIFER} is what we're replacing in the file
	regex, err := regexp.Compile("\\${IDENTIFIER}")
	if err != nil {
		framework.Failf("Error %v", err)
	}

	for i := 0; i < numObjects; i++ {
		result := regex.ReplaceAll(content, []byte(strconv.Itoa(i)))

		tmpfile, err := ioutil.TempFile("", "cl")
		if err != nil {
			framework.Failf("Error %v", err)
		}

		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(result); err != nil {
			framework.Failf("Error %v", err)
		}

		if err := tmpfile.Close(); err != nil {
			framework.Failf("Error %v", err)
		}

		framework.RunKubectlOrDie("create", "-f", tmpfile.Name(), getNsCmdFlag(ns))
		framework.Logf("%d/%d : Created template %s", i+1, numObjects, baseName)

		if tuning != nil {
			if tuning.Templates.RateLimit.Delay != 0 {
				framework.Logf("Sleeping %d ms between template creation.", tuning.Templates.RateLimit.Delay)
				time.Sleep(time.Duration(tuning.Templates.RateLimit.Delay) * time.Millisecond)
			}
			if tuning.Templates.Stepping.StepSize != 0 && (i+1)%tuning.Templates.Stepping.StepSize == 0 {
				framework.Logf("We have created %d templates and are now sleeping for %d seconds", i+1, tuning.Templates.Stepping.Pause)
				time.Sleep(time.Duration(tuning.Templates.Stepping.Pause) * time.Second)
			}
		}
	}
}

// parsePods unmarshalls the json file defined in the CL config into a struct
func parsePods(podYAML string (configJSON api.Pod) {
	config, err := ioutil.ReadFile(podYAML)
	if err != nil {
		framework.Failf("Cant read config file. Error: %v", err)
	}

	err = json.Unmarshal(config, &configJSON)
	framework.Logf("The loaded config file is: %+v", configJSON.Spec.Containers)
	return
}
