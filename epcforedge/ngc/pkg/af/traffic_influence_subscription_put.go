// SPDX-License-Identifier: Apache-2.0
// Copyright © 2019 Intel Corporation

package af

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

func modifySubscriptionByPut(cliCtx context.Context, ts TrafficInfluSub,
	afCtx *Context, sID string) (TrafficInfluSub,
	*http.Response, error) {

	cliCfg := NewConfiguration(afCtx)
	cli := NewClient(cliCfg)

	tsResp, resp, err := cli.TrafficInfluSubPutAPI.SubscriptionPut(cliCtx,
		afCtx.cfg.AfID, sID, ts)

	if err != nil {
		return TrafficInfluSub{}, resp, err
	}
	return tsResp, resp, nil
}

// ModifySubscriptionPut function
func ModifySubscriptionPut(w http.ResponseWriter, r *http.Request) {

	var (
		err     error
		ts      TrafficInfluSub
		tsResp  TrafficInfluSub
		resp    *http.Response
		sID     string
		transID int
	)

	afCtx := r.Context().Value(keyType("af-ctx")).(*Context)
	if afCtx == nil {
		log.Errf("Traffic Influance Subscription put: " +
			"af-ctx retrieved from request is nil")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if afCtx.subscriptions == nil {
		log.Errf("AF context subscriptions map has not been initialized")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if afCtx.transactions == nil {
		log.Errf("AF context transactions map has not been initialized")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	cliCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err = json.NewDecoder(r.Body).Decode(&ts); err != nil {
		log.Errf("Traffic Influance Subscription modify: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	transID, err = genTransactionID(afCtx)
	if err != nil {

		log.Errf("Traffic Influance Subscription modify %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ts.AFTransID = strconv.Itoa(transID)
	sID, err = getSubsIDFromURL(r.URL)
	if err != nil {
		log.Errf("Traffic Influence Subscription modify: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	afCtx.transactions[transID] = TrafficInfluSub{}
	tsResp, resp, err = modifySubscriptionByPut(cliCtx, ts, afCtx,
		sID)

	if err != nil {
		log.Errf("Traffic Influence Subscription modify : %s", err.Error())
		w.WriteHeader(getStatusCode(resp))
		return
	}

	if ts.SubscribedEvents == nil {
		delete(afCtx.transactions, transID)
		log.Infof("Deleted transaction: %v", transID)
	} else {
		afCtx.transactions[transID] = tsResp
	}

	if _, ok := afCtx.subscriptions[sID][strconv.Itoa(transID)]; ok {
		delete(afCtx.transactions, transID)
		log.Infof("Deleted transaction: %v", transID)
	}

	if afCtx.subscriptions[sID] == nil {
		afCtx.subscriptions[sID] =
			make(map[string]TrafficInfluSub)
	}

	afCtx.subscriptions[sID] =
		map[string]TrafficInfluSub{ts.AFTransID: tsResp}

	if resp != nil {
		w.WriteHeader(resp.StatusCode)
	}
}
