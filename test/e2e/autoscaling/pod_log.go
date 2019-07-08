/*
Copyright 2015 The Kubernetes Authors.

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

package autoscaling

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = SIGDescribe("PODLOG", func() {
	f := framework.NewDefaultFramework("podlog")
	var c clientset.Interface

	BeforeEach(func() {
		c = f.ClientSet

	})

	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pause-amd64-",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "write-pod",
					Image: "gcr.io/google_containers/pause-amd64:3.0",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 8080,
							Protocol:      v1.ProtocolTCP,
						},
					},
					ImagePullPolicy: v1.PullIfNotPresent,
				},
			},
			RestartPolicy: v1.RestartPolicyAlways,
		},
	}

	It("Create pod", func() {
		podsCreated, err := c.CoreV1().Pods(f.Namespace.Name).Create(pod)
		Expect(err).NotTo(HaveOccurred())
		framework.Logf("Pod created: %+v", podsCreated)
		f.WaitForPodRunning(podsCreated.Name)
		pod, err := c.CoreV1().Pods(f.Namespace.Name).Get(podsCreated.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		framework.Logf("Pod state: %v", pod.Status.Phase)
	})

})
