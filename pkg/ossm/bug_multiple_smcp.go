package ossm

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupMultipleSMCP() {
	util.Log.Info("Delete the Multiple CP")
	// util.KubeDeleteContents(meshNamespace, smmr)
	// util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV23_template, smcp))
	time.Sleep(time.Duration(40) * time.Second)
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV23_template_meta, smcp))
	time.Sleep(time.Duration(40) * time.Second)
}

// TestSMCPMutiple tests If multiple SMCPs exist in a namespace, the controller reconciles them all.
func TestSMCPMutiple(t *testing.T) {
	defer cleanupMultipleSMCP()
	defer util.RecoverPanic(t)
	util.Log.Info("Delete Validation Webhook ")
	util.Shell(`oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io`)

	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
	// util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template, smcp))
	// util.KubeApplyContents(meshNamespace, smmr)
	// time.Sleep(time.Duration(20) * time.Second)
	util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template_meta, smcp))
	time.Sleep(time.Duration(20) * time.Second)

	util.Log.Info("Verify SMCP status and pods")
	msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
	if !strings.Contains(msg, "ComponentsReady") {
		util.Log.Error("SMCP not Ready")
		t.Error("SMCP not Ready")
	}

	util.Log.Info("Verify meta control plane and status")
	text, _ := util.Shell(`oc get -n %s smcp/meta -o wide`, meshNamespace)
	if !strings.Contains(text, "ErrMultipleSMCPs") {
		util.Log.Error("SMCP not Ready")
		t.Error("SMCP not Ready")
	}
	util.Shell(`oc get -n %s pods`, meshNamespace)
	util.Shell(`oc wait --for=condition=Ready pods --all -n %s`, meshNamespace)

}
