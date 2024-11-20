package http

import (
	"context"
	"github.com/koupleless/virtual-kubelet/common/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

var _ = Describe("Module Lifecycle Test", func() {

	ctx := context.Background()

	nodeID := "test-http-base"
	clusterName := "test-cluster-name"

	mockBase := NewMockHttpBase(nodeID, clusterName, "1.0.0", env, 1238)

	mockModulePod := prepareModulePod("test-module", "default", utils.FormatNodeName(nodeID, env))

	Context("pod install", func() {
		It("base should become a ready vnode eventually", func() {
			go mockBase.Start(ctx, clientID)
			Eventually(func() bool {
				vnode := &v1.Node{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: utils.FormatNodeName(nodeID, env),
				}, vnode)
				vnodeReady := false
				for _, cond := range vnode.Status.Conditions {
					if cond.Type == v1.NodeReady {
						vnodeReady = cond.Status == v1.ConditionTrue
						break
					}
				}
				return err == nil && vnodeReady
			}, time.Second*20, time.Second).Should(BeTrue())
		})

		It("publish a module pod and it should be running", func() {
			err := k8sClient.Create(ctx, &mockModulePod)
			Expect(err).To(BeNil())
			Eventually(func() bool {
				podFromKubernetes := &v1.Pod{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: mockModulePod.Namespace,
					Name:      mockModulePod.Name,
				}, podFromKubernetes)
				return err == nil && podFromKubernetes.Status.Phase == v1.PodRunning
			}, time.Second*20, time.Second).Should(BeTrue())
			Eventually(func() bool {
				return len(mockBase.BizInfos) == 1
			}, time.Second*20, time.Second).Should(BeTrue())
		})

		It("delete pod, all modules should deactivated, pod should finally deleted from k8s", func() {
			err := k8sClient.Delete(ctx, &mockModulePod)
			Expect(err).To(BeNil())
			Eventually(func() bool {
				podFromKubernetes := &v1.Pod{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: mockModulePod.Namespace,
					Name:      mockModulePod.Name,
				}, podFromKubernetes)
				return errors.IsNotFound(err)
			}, time.Second*40, time.Second).Should(BeTrue())
		})

		It("base offline with deactive message and finally exit", func() {
			mockBase.Exit()
			Eventually(func() bool {
				vnode := &v1.Node{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: utils.FormatNodeName(nodeID, env),
				}, vnode)
				return errors.IsNotFound(err)
			}, time.Second*30, time.Second).Should(BeTrue())
		})
	})

})
