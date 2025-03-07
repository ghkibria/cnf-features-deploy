package bond

import (
	"context"
	"fmt"
	"strings"
	"time"

	netattdefv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	sriovtestclient "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/client"
	client "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/execute"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/networks"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/pods"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apitypes "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var sriovclient *sriovtestclient.ClientSet

func init() {
	sriovclient = sriovtestclient.New("")
}

var _ = Describe("[sriov] Bond CNI integration", func() {
	apiclient := client.New("")

	execute.BeforeAll(func() {
		err := namespaces.Create(namespaces.BondTestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := namespaces.CleanPods(namespaces.BondTestNamespace, apiclient)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("bond cni over sriov", func() {
		execute.BeforeAll(func() {
			if discovery.Enabled() {
				Skip("Tuned sriov tests disabled for discovery mode")
			}
			networks.CleanSriov(sriovclient, namespaces.BondTestNamespace)
			networks.CreateSriovPolicyAndNetwork(sriovclient, namespaces.SRIOVOperator, "test-network", "testresource", "")

			By("Checking the network-attachment-definition is ready")
			Eventually(func() error {
				nad := netattdefv1.NetworkAttachmentDefinition{}
				objKey := apitypes.NamespacedName{
					Namespace: namespaces.SRIOVOperator,
					Name:      "test-network",
				}
				err := client.Client.Get(context.Background(), objKey, &nad)
				return err
			}, 2*time.Minute, 1*time.Second).Should(BeNil())
		})

		It("pod with sysctl's on bond over sriov interfaces should start", func() {
			bondLinkName := "bond0"
			bondNetworkAttachmentDefinition, err := networks.NewNetworkAttachmentDefinitionBuilder(namespaces.BondTestNamespace, "bondifc").WithBond(bondLinkName, "net1", "net2", 1300).WithHostLocalIpam("1.1.1.0").Build()
			Expect(err).ToNot(HaveOccurred())
			err = client.Client.Create(context.Background(), bondNetworkAttachmentDefinition)
			Expect(err).ToNot(HaveOccurred())

			podDefinition := pods.DefineWithNetworks(namespaces.BondTestNamespace, []string{
				fmt.Sprintf("%s/%s", namespaces.SRIOVOperator, "test-network"),
				fmt.Sprintf("%s/%s", namespaces.SRIOVOperator, "test-network"),
				fmt.Sprintf("%s/%s@%s", namespaces.BondTestNamespace, "bond", bondLinkName),
			})
			pod, err := client.Client.Pods(namespaces.BondTestNamespace).Create(context.Background(), podDefinition, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = pods.WaitForCondition(client.Client, pod, corev1.ContainersReady, corev1.ConditionTrue, 1*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			stdout, err := pods.ExecCommand(client.Client, *pod, []string{"ip", "addr", "show", "bondifc"})
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Index(stdout.String(), "inet 1.1.1.0"))

			stdout, err = pods.ExecCommand(client.Client, *pod, []string{"ip", "link", "show", "net1"})
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Index(stdout.String(), "master bondifc"))

			stdout, err = pods.ExecCommand(client.Client, *pod, []string{"ip", "link", "show", "net2"})
			Expect(err).ToNot(HaveOccurred())
			Expect(strings.Index(stdout.String(), "master bondifc"))
		})
	})
})
