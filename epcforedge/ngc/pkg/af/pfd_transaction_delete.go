// SPDX-License-Identifier: Apache-2.0
// Copyright © 2019 Intel Corporation

package af

import (
	"context"
	"net/http"
)

func deletePfdTransaction(cliCtx context.Context, afCtx *Context,
	pfdID string) (*http.Response, error) {

	cliCfg := NewConfiguration(afCtx)
	cli := NewClient(cliCfg)

	resp, err := cli.PfdManagementDeleteAPI.PfdTransactionDelete(cliCtx,
		afCtx.cfg.AfID, pfdID)

	if err != nil {
		return resp, err
	}
	return resp, nil

}

// DeletePfdTransaction function
func DeletePfdTransaction(w http.ResponseWriter, r *http.Request) {
	var (
		err      error
		resp     *http.Response
		pfdTrans string
	)

	afCtx := r.Context().Value(keyType("af-ctx")).(*Context)
	if afCtx == nil {
		errRspHeader(&w, "DELETE", "af-ctx retrieved from request is nil",
			http.StatusInternalServerError)
		return
	}

	cliCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	pfdTrans = getPfdTransIDFromURL(r)

	resp, err = deletePfdTransaction(cliCtx, afCtx, pfdTrans)
	if err != nil {
		if resp != nil {
			errRspHeader(&w, "DELETE", err.Error(), resp.StatusCode)
		} else {
			errRspHeader(&w, "DELETE", err.Error(),
				http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(resp.StatusCode)
}
