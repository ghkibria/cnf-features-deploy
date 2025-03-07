//go:build !unittests
// +build !unittests

package test_test

import (
	"context"
	"flag"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/security"
	"log"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/bond"       // this is needed otherwise the bond test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/dpdk"       // this is needed otherwise the dpdk test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/fec"        // this is needed otherwise the fec test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/gatekeeper" // this is needed otherwise the gatekeeper test won't be executed'
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/ovs_qos"    // this is needed otherwise the ovs_qos test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/ptp"        // this is needed otherwise the ptp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/s2i"        // this is needed otherwise the dpdk test won't be executed
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sctp"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sctp"     // this is needed otherwise the sctp test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/security" // this is needed otherwise the security test won't be executed
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/sro"      // this is needed otherwise the sro test won't be executed
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/vrf"
	_ "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/e2esuite/xt_u32"                     // this is needed otherwise the xt_u32 test won't be executed
	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/1_performance" // this is needed otherwise the performance test won't be executed
	_ "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/4_latency"     // this is needed otherwise the performance test won't be executed

	_ "github.com/k8snetworkplumbingwg/sriov-network-operator/test/conformance/tests"
	sriovNamespaces "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/namespaces"
	_ "github.com/metallb/metallb-operator/test/e2e/functional/tests"
	_ "github.com/openshift/ptp-operator/test/conformance/ptp"

	perfUtils "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils"

	sriovClean "github.com/k8snetworkplumbingwg/sriov-network-operator/test/util/clean"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/clean"
	testclient "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/client"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/discovery"
	"github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/namespaces"
	testutils "github.com/openshift-kni/cnf-features-deploy/cnf-tests/testsuites/pkg/utils"
	perfClean "github.com/openshift/cluster-node-tuning-operator/test/e2e/performanceprofile/functests/utils/clean"
	ptpClean "github.com/openshift/ptp-operator/test/utils/clean"

	numaserialconf "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	_ "github.com/openshift-kni/numaresources-operator/test/e2e/serial/tests"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ginkgo_reporters "kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

// TODO: we should refactor tests to use client from controller-runtime package
// see - https://github.com/openshift/cluster-api-actuator-pkg/blob/master/pkg/e2e/framework/framework.go

var (
	junitPath          *string
	reportPath         *string
	skipTestNSCreation bool
)

func init() {
	junitPath = flag.String("junit", "junit.xml", "the path for the junit format report")
	reportPath = flag.String("report", "", "the path of the report file containing details for failed tests")

	skipTestNSCreation = false
	if os.Getenv("SKIP_TEST_NAMESPACES_CREATION") == "true" {
		skipTestNSCreation = true
	}
}

func TestTest(t *testing.T) {
	RegisterFailHandler(Fail)

	rr := []Reporter{}
	if ginkgo_reporters.Polarion.Run {
		rr = append(rr, &ginkgo_reporters.Polarion)
	}
	if *junitPath != "" {
		junitFile := path.Join(*junitPath, "cnftests-junit.xml")
		rr = append(rr, reporters.NewJUnitReporter(junitFile))
	}
	if *reportPath != "" {
		reportFile := path.Join(*reportPath, "cnftests_failure_report.log")
		reporter, err := testutils.NewReporter(reportFile)
		if err != nil {
			log.Fatalf("Failed to create log reporter %s", err)
		}
		rr = append(rr, reporter)
	}

	RunSpecsWithDefaultAndCustomReporters(t, "CNF Features e2e integration tests", rr)
}

var _ = BeforeSuite(func() {
	if !skipTestNSCreation {
		Expect(testclient.Client).NotTo(BeNil())
		// create test namespace
		err := namespaces.Create(testutils.NamespaceTesting, testclient.Client)
		Expect(err).ToNot(HaveOccurred())

		err = namespaces.Create(perfUtils.NamespaceTesting, testclient.Client)
		Expect(err).ToNot(HaveOccurred())

		err = namespaces.Create(namespaces.DpdkTest, testclient.Client)
		Expect(err).ToNot(HaveOccurred())

		err = namespaces.Create(testutils.GatekeeperTestingNamespace, testclient.Client)
		Expect(err).ToNot(HaveOccurred())

		err = namespaces.Create(namespaces.SroTestNamespace, testclient.Client)
		Expect(err).ToNot(HaveOccurred())
	}

	// note this intentionally does NOT set the infra we depends on the configsuite for this
	_ = numaserialconf.SetupFixture()
	// note we do NOT CHECK for error to have occurred - intentionally.
	// Among other things, this function gets few NUMA resources-specific objects.
	// In case we do NOT have the NUMA resources CRDs deployed, the setup will fail.
	// But we cannot know until we run the tests, so we handle this in the tests themselves.
	// This will be improved in future releases of the numaresources operator.
})

// We do the cleanup in AfterSuite because the failure reporter is triggered
// after a test fails. If we did it as part of the test body, the reporter would not
// find the items we want to inspect.
var _ = AfterSuite(func() {
	numaserialconf.Teardown()

	clean.All()
	ptpClean.All()
	sriovClean.All()
	if !discovery.Enabled() {
		perfClean.All()
	}

	if !skipTestNSCreation {
		nn := []string{testutils.NamespaceTesting,
			perfUtils.NamespaceTesting,
			namespaces.DpdkTest,
			sctp.TestNamespace,
			vrf.TestNamespace,
			sriovNamespaces.Test,
			namespaces.XTU32Test,
			testutils.GatekeeperTestingNamespace,
			namespaces.OVSQOSTest,
			namespaces.SroTestNamespace,
			security.TestNamespace,
			security.SriovTestNamespace,
			namespaces.BondTestNamespace,
		}

		for _, n := range nn {
			err := testclient.Client.Namespaces().Delete(context.Background(), n, metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				continue
			}
			Expect(err).ToNot(HaveOccurred())
			err = namespaces.WaitForDeletion(testclient.Client, n, 5*time.Minute)
			Expect(err).ToNot(HaveOccurred())
		}
	}
})
