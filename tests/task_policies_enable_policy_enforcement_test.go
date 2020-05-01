// Copyright 2020 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func TestEnablePolicyEnforcement(t *testing.T) {
	defer recoverPanic(t)

	t.Run("Policies_enable_policy_enforcement", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Enabling Policy Enforcement")
		util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"disablePolicyChecks\":false}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*4) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

		log.Info("Validate the policy enforcement")
		msg, _ := util.Shell("kubectl -n %s get cm istio -o jsonpath=\"{@.data.mesh}\" | grep disablePolicyChecks", meshNamespace)
		if strings.Contains(msg, "false") {
			log.Info("Success.")
		} else {
			log.Errorf("Failed. Got: %s", msg)
			t.Errorf("Failed. Got: %s", msg)
		}
	})
}
