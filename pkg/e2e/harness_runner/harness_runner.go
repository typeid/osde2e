package harness_runner

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	viper "github.com/openshift/osde2e/pkg/common/concurrentviper"
	"github.com/openshift/osde2e/pkg/common/config"
	"github.com/openshift/osde2e/pkg/common/helper"
	"github.com/openshift/osde2e/pkg/common/label"
	"github.com/openshift/osde2e/pkg/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	serviceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"
	TimeoutInSeconds  = viper.GetFloat64(config.Tests.PollingTimeout)
	harnesses         = strings.Split(viper.GetString(config.Tests.TestHarnesses), ",")
	h                 *helper.H
	HarnessEntries    []ginkgo.TableEntry
)

var _ = ginkgo.Describe("Test Harness", ginkgo.Ordered, label.TestHarness, func() {
	for _, harness := range harnesses {
		HarnessEntries = append(HarnessEntries, ginkgo.Entry("should run "+harness+" successfully", harness))
	}
	ginkgo.DescribeTable("Executing Harness",
		func(ctx context.Context, harness string) {
			ginkgo.By("======= RUNNING HARNESS: " + harness + " =======")
			log.Printf("======= RUNNING HARNESS: %s =======", harness)
			viper.Set(config.Project, "")
			// Run harness in new project
			h = helper.New()
			h.SetServiceAccount(ctx, "system:serviceaccount:%s:cluster-admin")
			harnessImageIndex := strings.LastIndex(harness, "/")
			harnessImage := harness[harnessImageIndex+1:]
			suffix := util.RandomStr(5)
			jobName := fmt.Sprintf("%s-%s", harnessImage, suffix)
			r := h.RunnerWithTemplateCommand(TimeoutInSeconds, harness, suffix, jobName, serviceAccountDir)

			// run tests
			stopCh := make(chan struct{})
			err := r.Run(int(TimeoutInSeconds), stopCh)
			Expect(err).NotTo(HaveOccurred(), "Could not run pod")

			// get results
			results, err := r.RetrieveTestResults()
			Expect(err).NotTo(HaveOccurred(), "Could not read results")

			// write results
			h.WriteResults(results)

			// ensure job has not failed
			_, err = h.Kube().BatchV1().Jobs(r.Namespace).Get(ctx, jobName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred(), "Harness job pods failed")

			h.Cleanup(ctx)
			ginkgo.By("======= FINISHED HARNESS: " + harness + " =======")
		},
		HarnessEntries)
})
