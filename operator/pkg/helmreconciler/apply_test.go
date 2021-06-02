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

package helmreconciler

import (
	"context"
	"io/ioutil"
	"reflect"
	"sync"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha12 "istio.io/api/operator/v1alpha1"
	"istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/operator/pkg/object"
)

func TestHelmReconciler_ApplyObject(t *testing.T) {
	tests := []struct {
		name         string
		currentState string
		input        string
		want         string
		wantErr      bool
	}{
		{
			name:  "creates if not present",
			input: "testdata/configmap.yaml",
			want:  "testdata/configmap.yaml",
		},
		{
			name:         "updates if present",
			currentState: "testdata/configmap.yaml",
			input:        "testdata/configmap-changed.yaml",
			want:         "testdata/configmap-changed.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := loadData(t, tt.input)
			cl := &fakeClientWrapper{fake.NewClientBuilder().WithRuntimeObjects(loadData(t, tt.input).UnstructuredObject()).Build()}
			h := &HelmReconciler{
				client: cl,
				opts:   &Options{},
				iop: &v1alpha1.IstioOperator{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-operator",
						Namespace: "istio-operator-test",
					},
					Spec: &v1alpha12.IstioOperatorSpec{},
				},
				countLock:     &sync.Mutex{},
				prunedKindSet: map[schema.GroupKind]struct{}{},
			}
			if err := h.ApplyObject(obj.UnstructuredObject(), false); (err != nil) != tt.wantErr {
				t.Errorf("HelmReconciler.ApplyObject() error = %v, wantErr %v", err, tt.wantErr)
			}

			manifest := loadData(t, tt.want)
			key := client.ObjectKeyFromObject(manifest.UnstructuredObject())
			got, want := obj.UnstructuredObject(), manifest.UnstructuredObject()

			if err := cl.Get(context.Background(), key, got); err != nil {
				t.Errorf("error validating manifest %v: %v", manifest.Hash(), err)
			}
			// remove resource version and annotations (last applied config) when we compare as we don't care
			unstructured.RemoveNestedField(got.Object, "metadata", "resourceVersion")
			unstructured.RemoveNestedField(got.Object, "metadata", "annotations")

			if !reflect.DeepEqual(want, got) {
				t.Errorf("wanted:\n%v\ngot:\n%v",
					object.NewK8sObject(want, nil, nil).YAMLDebugString(),
					object.NewK8sObject(got, nil, nil).YAMLDebugString(),
				)
			}
		})
	}
}

func TestHelmReconciler_GetObject(t *testing.T) {
	getK8sObj := func(content string) *object.K8sObject {
		obj, err := object.ParseYAMLToK8sObject([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
		return obj
	}

	testInput1 := `---
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
  name: example-istiocontrolplane
spec:
  dummy:
  traffic_management:
    components:
    namespace: istio-traffic-management`
	testInput2 := `---
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-operator-test
  name: test-operator
  resourceVersion: "999"
spec:
  dummy:
  traffic_management:
    components:
    namespace: istio-traffic-management`
	tests := []struct {
		name      string
		input     string
		want      *unstructured.Unstructured
		searchIop *v1alpha1.IstioOperator
		hasErr    bool
	}{
		{
			name:  "object not present",
			input: testInput1,
			searchIop: &v1alpha1.IstioOperator{
				Kind:       "IstioOperator",
				ApiVersion: "install.istio.io/v1alpha1",
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "istio-operator-test",
				},
				Spec: &v1alpha12.IstioOperatorSpec{},
			},
			want:   nil,
			hasErr: false,
		},
		{
			name:  "object presents",
			input: testInput2,
			searchIop: &v1alpha1.IstioOperator{
				Kind:       "IstioOperator",
				ApiVersion: "install.istio.io/v1alpha1",
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "istio-operator-test",
				},
				Spec: &v1alpha12.IstioOperatorSpec{},
			},
			want:   getK8sObj(testInput2).UnstructuredObject(),
			hasErr: false,
		},
		{
			name:  "error object missing required data",
			input: testInput2,
			searchIop: &v1alpha1.IstioOperator{
				Kind:       "IstioOperator",
				ApiVersion: "",
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-operator",
					Namespace: "istio-operator-test",
				},
				Spec: &v1alpha12.IstioOperatorSpec{},
			},
			want:   nil,
			hasErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &fakeClientWrapper{fake.NewClientBuilder().WithRuntimeObjects(getK8sObj(tt.input).UnstructuredObject()).Build()}
			h := &HelmReconciler{
				client:        cl,
				opts:          &Options{},
				iop:           tt.searchIop,
				countLock:     &sync.Mutex{},
				prunedKindSet: map[schema.GroupKind]struct{}{},
			}
			ret, err := h.GetObject(h.iop)
			if !tt.hasErr && err != nil {
				t.Errorf("Expected no error but got error %v", err)
			}

			if tt.hasErr && err == nil {
				t.Error("wanted error returned but err was nil")
			}

			if !tt.hasErr && !reflect.DeepEqual(ret, tt.want) {
				t.Errorf("wanted:\n%v\ngot:\n%v",
					object.NewK8sObject(tt.want, nil, nil).YAMLDebugString(),
					object.NewK8sObject(ret, nil, nil).YAMLDebugString(),
				)
			}
		})
	}
}

type fakeClientWrapper struct {
	client.Client
}

// Patch converts apply patches to merge patches because fakeclient does not support apply patch.
func (c *fakeClientWrapper) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	patch, opts = convertApplyToMergePatch(patch, opts...)
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func convertApplyToMergePatch(patch client.Patch, opts ...client.PatchOption) (client.Patch, []client.PatchOption) {
	if patch.Type() == types.ApplyPatchType {
		patch = client.Merge
		patchOptions := make([]client.PatchOption, 0, len(opts))
		for _, opt := range opts {
			if opt == client.ForceOwnership {
				continue
			}
			patchOptions = append(patchOptions, opt)
		}
		opts = patchOptions
	}
	return patch, opts
}

func loadData(t *testing.T, file string) *object.K8sObject {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := object.ParseYAMLToK8sObject(contents)
	if err != nil {
		t.Fatal(err)
	}
	return obj
}
