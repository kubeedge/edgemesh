package traffic

import (
	"flag"
	"math/rand"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"

	"github.com/kubeedge/edgemesh/tests/e2e/k8s"
	"github.com/kubeedge/kubeedge/tests/e2e/utils"
)

var (
	lan2NodeNames          map[string][]string
	ctx                    *utils.TestContext
	busyboxToolContainerID string
	busyboxToolName        = "busybox-edge-tools-" + utils.GetRandomString(5)
)

func TestMain(m *testing.M) {
	utils.CopyFlags(utils.Flags, flag.CommandLine)
	utils.RegisterFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	os.Exit(m.Run())
}

func TestEdgeMeshTraffic(t *testing.T) {
	rand.Seed(time.Now().Unix())
	RegisterFailHandler(Fail)
	BeforeSuite(func() {
		utils.Infof("Before Suite Execution")
		ctx = utils.NewTestContext(utils.LoadConfig())
		lan2NodeNames = make(map[string][]string)
		lan2NodeNames["edge-lan-01"] = []string{"edge-node"}

		// start a busybox tool pod for test
		nodeSelector := map[string]string{"lan": "edge-lan-01"}
		labels := map[string]string{"app": "busybox"}
		busyboxPod, err := k8s.CreateBusyboxTool(busyboxToolName, labels, nodeSelector, ctx)
		Expect(err).To(BeNil())
		// delete "docker://" in docker://8f2e9eb669d42c09dae3901286e1e09709059090fed49e411042662d0666735d
		busyboxToolContainerID = busyboxPod.Status.ContainerStatuses[0].ContainerID[9:]
	})
	AfterSuite(func() {
		By("After Suite Execution....!")
		err := k8s.CleanBusyBoxTool(busyboxToolName, ctx)
		Expect(err).To(BeNil())
	})
	RunSpecs(t, "EdgeMesh App Traffic Suite")
}
