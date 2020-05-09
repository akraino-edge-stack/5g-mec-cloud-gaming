// SPDX-License-Identifier: Apache-2.0
// Copyright © 2019 Intel Corporation

package af

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

func verifyAFTransID(afCtx *Context, transID string, p *ProblemDetails) (int,
	error) {

	var (
		transIDInt int
		err        error
	)

	const ProblemTitle = "AF transaction ID verification"

	if transID == "" {
		log.Errf("Traffic Influence Subscription notification - empty " +
			"afTransID")
		p.Status = http.StatusInternalServerError
		p.Title = ProblemTitle
		p.Detail = "Traffic Influance Subscription notification" +
			" - empty transactionID"
		p.InvalidParams = []InvalidParam{{
			Param:  "AfTransId",
			Reason: "Empty"},
		}

		err = errors.New("empty AfTransID")
		return http.StatusInternalServerError, err
	}

	if transIDInt, err = strconv.Atoi(transID); err != nil {
		log.Errf("Error while converting transaction ID to int: %s.", err)
		p.Status = http.StatusInternalServerError
		p.Title = ProblemTitle
		p.Detail = "Traffic Influance Subscription notification - " +
			"error while converting transaction ID to int: " +
			err.Error()
		p.InvalidParams = []InvalidParam{{
			Param: "AfTransID",
			Reason: "AFTransID = " + transID + ". Unable to convert to " +
				" integer."},
		}
		err = errors.New("error converting AfTransID string to integer ")
		return http.StatusInternalServerError, err
	}

	if _, ok := afCtx.transactions[transIDInt]; !ok {
		log.Errf("Transaction ID %s corresponding to notification does "+
			"not exist", transID)
		p.Status = http.StatusInternalServerError
		p.Title = ProblemTitle
		p.Detail = "Traffic Influance Subscription notification - " +
			"Transaction ID " + transID + " corresponding to " +
			"notification was not found"

		err = errors.New("AfTransID not found")
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil

}

// NotificationPost function
func NotificationPost(w http.ResponseWriter, r *http.Request) {

	var (
		err        error
		en         EventNotification
		prJSON     []byte
		statusCode int
		problem    = ProblemDetails{}
	)

	afCtx := r.Context().Value(keyType("af-ctx")).(*Context)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err = json.NewDecoder(r.Body).Decode(&en); err != nil {
		log.Errf("Traffic Influance Subscription notify: %s", err.Error())
		problem = ProblemDetails{
			Status: http.StatusInternalServerError,
			Title:  "Decoding response body",
			Detail: "Traffic Influance Subscription notify: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		if prJSON, err = json.Marshal(problem); err == nil {
			if _, err = w.Write(prJSON); err == nil {
				return
			}
		}
		log.Errf("Traffic Influance Subscription notify: %s", err.Error())
		return
	}

	if statusCode, err = verifyAFTransID(afCtx, en.AFTransID,
		&problem); err != nil {

		w.WriteHeader(statusCode)
		if prJSON, err = json.Marshal(problem); err == nil {
			if _, err = w.Write(prJSON); err != nil {
				log.Errf("Traffic Influance Subscription notify: %s",
					err.Error())
			}
			return
		}
		log.Errf("Traffic Influance Subscription notify: %s", err.Error())
		return
	}
	w.WriteHeader(statusCode)
}
