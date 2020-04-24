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

package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupExternalCert(namespace string) {
	log.Info("# Cleanup ...")
	util.Shell("kubectl delete secret cacerts -n %s", meshNamespace)
	cleanBookinfo(namespace)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"security\":{\"selfSigned\":true}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=citadel", kubeconfig)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=pilot", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

}

func verifyCerts() error {
	pod, err := util.GetPodName(testNamespace, "app=ratings", kubeconfig)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("/bin/cat /etc/certs/root-cert.pem")
	podRootCert, err := util.PodExec(testNamespace, pod, "istio-proxy", cmd, true, kubeconfig)
	if err != nil {
		return err
	}
	ioutil.WriteFile("/tmp/pod-root-cert.pem", []byte(podRootCert), 0644)
	//util.ShellMuteOutput("kubectl exec -n %s -it %s -c istio-proxy -- /bin/cat /etc/certs/root-cert.pem > /tmp/pod-root-cert.pem", testNamespace, pod)

	cmd = fmt.Sprintf("/bin/cat /etc/certs/cert-chain.pem")
	podCertChain, err := util.PodExec(testNamespace, pod, "istio-proxy", cmd, true, kubeconfig)
	if err != nil {
		return err
	}
	ioutil.WriteFile("/tmp/pod-cert-chain.pem", []byte(podCertChain), 0644)
	//util.ShellMuteOutput("kubectl exec -n %s -it %s -c istio-proxy -- /bin/cat /etc/certs/cert-chain.pem > /tmp/pod-cert-chain.pem", testNamespace, pod)

	util.ShellMuteOutput("openssl x509 -in %s -text -noout > /tmp/root-cert.crt.txt", caRootCert)
	util.ShellMuteOutput("openssl x509 -in %s -text -noout > /tmp/pod-root-cert.crt.txt", "/tmp/pod-root-cert.pem")
	err = util.CompareFiles("/tmp/root-cert.crt.txt", "/tmp/pod-root-cert.crt.txt")
	if err != nil {
		return err
	}

	util.ShellMuteOutput("tail -n 22 %s > /tmp/pod-cert-chain-ca.pem", "/tmp/pod-cert-chain.pem")
	util.ShellMuteOutput("openssl x509 -in %s -text -noout > /tmp/ca-cert.crt.txt", caCert)
	util.ShellMuteOutput("openssl x509 -in /tmp/pod-cert-chain-ca.pem -text -noout > /tmp/pod-cert-chain-ca.crt.txt")
	err = util.CompareFiles("/tmp/ca-cert.crt.txt", "/tmp/pod-cert-chain-ca.crt.txt")
	if err != nil {
		return err
	}

	util.ShellMuteOutput("head -n 21 %s > /tmp/pod-cert-chain-workload.pem", "/tmp/pod-cert-chain.pem")
	util.ShellMuteOutput("cat %s %s > /tmp/ca-cert-file.crt.txt", caCert, caRootCert)
	msg, err := util.Shell("openssl verify -CAfile /tmp/ca-cert-file.crt.txt /tmp/pod-cert-chain-workload.pem")
	if err != nil || !strings.Contains(msg, "OK") {
		return fmt.Errorf("Error certs: %s", msg)
	}

	return nil
}

func TestExternalCert(t *testing.T) {
	defer cleanupExternalCert(testNamespace)
	defer recoverPanic(t)

	log.Info("Plugging in External CA Key and Certificate")

	// update mtls to true
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=pilot", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

	t.Run("Security_citadel_plugging_external_cert_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Create secret")
		_, err := util.ShellMuteOutput("kubectl create secret generic %s -n %s --from-file %s --from-file %s --from-file %s --from-file %s",
			"cacerts",
			meshNamespace,
			caCert,
			caCertKey,
			caRootCert,
			caCertChain,
		)
		if err != nil {
			log.Infof("Failed to create secret %s\n", "cacerts")
			t.Errorf("Failed to create secret %s\n", "cacerts")
		}
		log.Infof("Secret %s created\n", "cacerts")
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		log.Info("Redeploy Citadel")
		util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"security\":{\"selfSigned\":false}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*10) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=citadel", kubeconfig)

		log.Info("Delete existing istio.default secret")
		util.Shell("kubectl delete -n %s secret istio.default", testNamespace)
		time.Sleep(time.Duration(waitTime) * time.Second)

		log.Info("Deploy bookinfo")
		deployBookinfo(testNamespace, false)

		log.Info("Verify the new certificates")
		err = verifyCerts()
		if err != nil {
			log.Infof("%v", err)
			t.Errorf("%v", err)
		}
	})
}
