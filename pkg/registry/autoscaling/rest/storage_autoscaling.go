/*
Copyright 2016 The Kubernetes Authors.

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

package rest

import (
	"sync"

	autoscalingapiv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/autoscaling"
	autoscalingapiv2 "k8s.io/kubernetes/pkg/apis/autoscaling/v2"
	horizontalpodautoscalerstore "k8s.io/kubernetes/pkg/registry/autoscaling/horizontalpodautoscaler/storage"
)

type RESTStorageProvider struct{}

func (p RESTStorageProvider) NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(autoscaling.GroupName, legacyscheme.Scheme, legacyscheme.ParameterCodec, legacyscheme.Codecs)
	// If you add a version here, be sure to add an entry in `k8s.io/kubernetes/cmd/kube-apiserver/app/aggregator.go with specific priorities.
	// TODO refactor the plumbing to provide the information in the APIGroupInfo

	var hpaOnce sync.Once
	var hpaStorage *horizontalpodautoscalerstore.REST
	var hpaStatusStorage *horizontalpodautoscalerstore.StatusREST
	var hpaStorageErr error
	var storageGetter hpaStorageGetter = func() (*horizontalpodautoscalerstore.REST, *horizontalpodautoscalerstore.StatusREST, error) {
		hpaOnce.Do(func() {
			hpaStorage, hpaStatusStorage, hpaStorageErr = horizontalpodautoscalerstore.NewREST(restOptionsGetter)
		})
		return hpaStorage, hpaStatusStorage, hpaStorageErr
	}

	if storageMap, err := p.v2Storage(apiResourceConfigSource, storageGetter); err != nil {
		return genericapiserver.APIGroupInfo{}, err
	} else if len(storageMap) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap[autoscalingapiv2.SchemeGroupVersion.Version] = storageMap
	}

	if storageMap, err := p.v1Storage(apiResourceConfigSource, storageGetter); err != nil {
		return genericapiserver.APIGroupInfo{}, err
	} else if len(storageMap) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap[autoscalingapiv1.SchemeGroupVersion.Version] = storageMap
	}

	return apiGroupInfo, nil
}

func (p RESTStorageProvider) v1Storage(apiResourceConfigSource serverstorage.APIResourceConfigSource, storageGetter hpaStorageGetter) (map[string]rest.Storage, error) {
	storage := map[string]rest.Storage{}

	// horizontalpodautoscalers
	if resource := "horizontalpodautoscalers"; apiResourceConfigSource.ResourceEnabled(autoscalingapiv1.SchemeGroupVersion.WithResource(resource)) {
		hpaStorage, hpaStatusStorage, err := storageGetter()
		if err != nil {
			return storage, err
		}
		storage[resource] = hpaStorage
		storage[resource+"/status"] = hpaStatusStorage
	}

	return storage, nil
}

func (p RESTStorageProvider) v2Storage(apiResourceConfigSource serverstorage.APIResourceConfigSource, storageGetter hpaStorageGetter) (map[string]rest.Storage, error) {
	storage := map[string]rest.Storage{}

	// horizontalpodautoscalers
	if resource := "horizontalpodautoscalers"; apiResourceConfigSource.ResourceEnabled(autoscalingapiv2.SchemeGroupVersion.WithResource(resource)) {
		hpaStorage, hpaStatusStorage, err := storageGetter()
		if err != nil {
			return storage, err
		}
		storage[resource] = hpaStorage
		storage[resource+"/status"] = hpaStatusStorage
	}

	return storage, nil
}

type hpaStorageGetter func() (*horizontalpodautoscalerstore.REST, *horizontalpodautoscalerstore.StatusREST, error)

func (p RESTStorageProvider) GroupName() string {
	return autoscaling.GroupName
}
