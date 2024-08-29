/*
Copyright 2024 Klutch Authors. All rights reserved.

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

package v2

import (
	"fmt"
	"net/http"

	"k8s.io/klog/v2"
)

func (c *client) GetCatalog() (*CatalogResponse, error) {
	catalog := c.getCatalogFromCache()
	if catalog == nil {
		return c.getCatalogFromBroker()
	} else {
		return catalog, nil
	}
}

func (c *client) getCatalogFromCache() *CatalogResponse {
	if c.cache == nil {
		return nil
	} else {
		catalog, exists, err := c.cache.GetByKey("catalog")
		if err != nil {
			klog.Info(fmt.Sprintf("There was an error retrieving the catalog from the cache\n%v", err))
			return nil
		}
		if !exists {
			if c.Verbose {
				klog.Info("Catalog was not found in cache - cache will be refreshed")
			}
			return nil
		}
		convertedCatalog, ok := catalog.(cachedCatalog)
		if !ok {
			return nil
		}

		if c.Verbose {
			klog.Info("Fetching Catalog from cache")
		}

		return convertedCatalog.value
	}
}

func (c *client) setCatalogInCache(catalog *CatalogResponse) {
	if c.cache != nil {
		err := c.cache.Add(cachedCatalog{
			key:   "catalog",
			value: catalog,
		})

		if err != nil {
			klog.Info(fmt.Sprintf("Unexpected error occurred while saving the catalog in the cache\n%v", err))
		}
	}
}

func (c *client) getCatalogFromBroker() (*CatalogResponse, error) {
	fullURL := fmt.Sprintf(catalogURL, c.URL)

	response, err := c.prepareAndDo(http.MethodGet, fullURL, nil /* params */, nil /* request body */, nil /* originating identity */)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = drainReader(response.Body)
		response.Body.Close()
	}()

	switch response.StatusCode {
	case http.StatusOK:
		catalogResponse := &CatalogResponse{}
		if err := c.unmarshalResponse(response, catalogResponse); err != nil {
			return nil, HTTPStatusCodeError{StatusCode: response.StatusCode, ResponseError: err}
		}

		if c.APIVersion.IsLessThan(Version2_13()) || !c.EnableAlphaFeatures {
			c.pruneCatalogResponse(catalogResponse)
		}

		c.setCatalogInCache(catalogResponse)

		return catalogResponse, nil
	default:
		return nil, c.handleFailureResponse(response)
	}
}

func (c *client) pruneCatalogResponse(catalogResponse *CatalogResponse) {
	for ii := range catalogResponse.Services {
		for jj := range catalogResponse.Services[ii].Plans {
			if c.APIVersion.IsLessThan(Version2_13()) {
				catalogResponse.Services[ii].Plans[jj].Schemas = nil
			}
			if !c.EnableAlphaFeatures {
				catalogResponse.Services[ii].Plans[jj].MaintenanceInfo = nil
				catalogResponse.Services[ii].Plans[jj].MaximumPollingDuration = nil
				catalogResponse.Services[ii].Plans[jj].PlanUpdateable = nil
			}
		}
	}
}
