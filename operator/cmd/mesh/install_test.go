// Copyright Istio Authors
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

package mesh

import (
	"strings"
	"testing"

	"istio.io/api/operator/v1alpha1"
	pkgAPI "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestManifestFromString(t *testing.T) {
	tests := []struct {
		desc         string
		given        string
		mustContains []string
		expectErr    bool
	}{
		{
			desc: "with enabled component",
			given: `---
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
spec:
  components:
    ingressGateways:
    - name: istio-ingressgateway
      enabled: true`,
			mustContains: []string{
				"name: istio-ingressgateway-service-account",
				"name: istio-ingressgateway",
			},
			expectErr: false,
		},
		{
			desc: "with no enabled component",
			given: `---
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
spec:
  components:
    pilot:
      k8s:
        podDisruptionBudget:
          maxUnavailable: 2`,
			mustContains: []string{""},
			expectErr:    false,
		},
		{
			desc: "invalid spec",
			given: `---
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: invalid-virtual-service
spec:
  http`,
			mustContains: []string{""},
			expectErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ManifestFromString(tt.given)
			if !tt.expectErr {
				if err != nil {
					t.Errorf("Expect no errors but got one %v", err)
				} else {
					for _, mcs := range tt.mustContains {
						if !strings.Contains(got, mcs) {
							t.Errorf("Results must contain %s, but is not found in the results", mcs)
						}
					}
				}
			} else if err == nil {
				t.Errorf("Expected error but did not get any")
			}
		})
	}
}

func TestShowDiffAgain(t *testing.T) {
	testInput1 := `---
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-operator-test
  name: test-operator
spec:
  components:
    ingressGateways:
    - name: istio-ingressgateway
      k8s:
        service:
          ports:
          - port: 15021
            targetPort: 15021
            name: status-port
      enabled: true`
	tests := []struct {
		name  string
		diff1 string
	}{
		{
			name:  "different inputs",
			diff1: testInput1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liop := &pkgAPI.IstioOperator{}
			err := util.UnmarshalWithJSONPB(tt.diff1, liop, true)
			if err != nil {
				t.Errorf("Expected no error but got error %v", err)
			}
		})
	}
}

func TestShowDiffSimple(t *testing.T) {
	testInput1 := `---
port: 15021
targetPort: 50121
  type: 1
  strVal: whatever
name: status-port`
	tests := []struct {
		name  string
		diff1 string
	}{
		{
			name:  "different inputs",
			diff1: testInput1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liop := &v1.ServicePort{}
			err := util.UnmarshalWithJSONPB(tt.diff1, liop, true)
			if err != nil {
				t.Errorf("Expected no error but got error %v", err)
			}
		})
	}
}

func TestIntOrStr(t *testing.T) {
	testInput1 := `---
targetPort: 50212
`
	tests := []struct {
		name  string
		diff1 string
	}{
		{
			name:  "different inputs",
			diff1: testInput1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liop := &intstr.IntOrString{}
			err := util.UnmarshalWithJSONPB(tt.diff1, liop, true)
			if err != nil {
				t.Errorf("Expected no error but got error %v", err)
			}
		})
	}
}

func TestIntOrStrPB(t *testing.T) {
	testInput1 := `---
50212
`
	tests := []struct {
		name  string
		diff1 string
	}{
		{
			name:  "different inputs",
			diff1: testInput1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			liop := &v1alpha1.IntOrStringForPB{}
			err := util.UnmarshalWithJSONPB(tt.diff1, liop, true)
			if err != nil {
				t.Errorf("Expected no error but got error %v", err)
			}
		})
	}
}
