/*
Copyright 2021 Juicedata Inc

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

package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/transport"
	"k8s.io/klog"
)

const (
	timeout = 10 * time.Second
)

type kubeletClient struct {
	host   string
	port   int
	client *http.Client
}

func makeRoundTripper() (http.RoundTripper, error) {
	const (
		tokenFile  = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		rootCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	)
	token, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	cfg := &transport.Config{
		TLS: transport.TLSConfig{
			Insecure: true,
		},
		BearerToken: string(token),
	}

	tlsConfig, err := transport.TLSConfigFor(cfg)
	if err != nil {
		return nil, err
	}
	rt := utilnet.SetOldTransportDefaults(&http.Transport{
		TLSClientConfig: tlsConfig,
	})

	// cfg.TLS.Insecure = false
	// cfg.TLS.CAFile = rootCAFile
	return transport.HTTPWrappersForConfig(cfg, rt)
}

func newKubeletClient(host string, port int) (*kubeletClient, error) {
	rr, err := makeRoundTripper()
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: rr,
		Timeout:   timeout,
	}
	return &kubeletClient{
		host:   host,
		port:   port,
		client: client,
	}, nil
}

func (kc *kubeletClient) GetNodeRunningPods() (*corev1.PodList, error) {
	resp, err := kc.client.Get(fmt.Sprintf("https://%v:%d/pods/", kc.host, kc.port))
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	podLists := &corev1.PodList{}
	if err = json.Unmarshal(body, &podLists); err != nil {
		klog.V(5).Infof("GetNodeRunningPods err: %s", body)
		return nil, err
	}
	return podLists, err
}
