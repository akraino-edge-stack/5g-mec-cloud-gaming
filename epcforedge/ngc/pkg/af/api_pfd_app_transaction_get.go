// SPDX-License-Identifier: Apache-2.0
// Copyright © 2020 Intel Corporation

package af

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

// Linger please
var (
	_ context.Context
)

// PfdManagementTransactionAppGetAPIService type
type PfdManagementTransactionAppGetAPIService service

func (a *PfdManagementTransactionAppGetAPIService) handlePfdAppGetResponse(
	pfdTs *PfdData, r *http.Response,
	body []byte) error {

	if r.StatusCode == 200 {
		err := json.Unmarshal(body, pfdTs)
		if err != nil {
			log.Errf("Error decoding response body %s: ", err.Error())
			r.StatusCode = 500
		}
		return err
	}
	return handleGetErrorResp(r, body)
}

/*
PfdAppTransactionGet Read an active pfd Data
for the AF, pfd transcation id and an application ID
Read an active pfd transactions for the AF, pfd transaction Id and app ID
 * @param ctx context.Context - for authentication, logging, cancellation,
 * deadlines, tracing, etc. Passed from http.Request or context.Background().
 * @param afID Identifier of the AF
 * @param pfd transaction Identifier of the PFD transaction resource
 * @param appID Application Identifier

@return PfdData
*/
func (a *PfdManagementTransactionAppGetAPIService) PfdAppTransactionGet(
	ctx context.Context, afID string, pfdTransaction string, appID string) (
	PfdData, *http.Response, error) {

	var (
		method  = strings.ToUpper("Get")
		getBody interface{}
		ret     PfdData
	)

	// create path and map variables
	path := a.client.cfg.Protocol + "://" + a.client.cfg.NEFHostname +
		a.client.cfg.NEFPort + a.client.cfg.NEFPFDBasePath + "/" + afID +
		"/transactions/" + pfdTransaction + "/applications/" + appID

	headerParams := make(map[string]string)

	headerParams["Content-Type"] = contentType
	headerParams["Accept"] = contentType

	r, err := a.client.prepareRequest(ctx, path, method,
		getBody, headerParams)
	if err != nil {
		return ret, nil, err
	}

	resp, err := a.client.callAPI(r)
	if err != nil || resp == nil {
		return ret, resp, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Errf("response body was not closed properly")
		}
	}()

	if err != nil {
		log.Errf("http response body could not be read")
		return ret, resp, err
	}

	if err = a.handlePfdAppGetResponse(&ret, resp,
		respBody); err != nil {

		return ret, resp, err
	}

	return ret, resp, nil
}
