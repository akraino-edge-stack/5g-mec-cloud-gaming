/* SPDX-License-Identifier: Apache-2.0
* Copyright (c) 2019 Intel Corporation
 */

package ngcnef

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	//"strconv"

	"github.com/gorilla/mux"
)

func createNewSub(nefCtx *nefContext, afID string,
	ti TrafficInfluSub) (loc string, rsp nefSBRspData, err error) {

	var af *afData
	nef := &nefCtx.nef

	af, err = nef.nefGetAf(afID)

	if err != nil {
		log.Err("NO AF PRESENT CREATE AF")
		af, err = nef.nefAddAf(nefCtx, afID)
		if err != nil {
			return loc, rsp, err
		}
	} else {
		log.Infoln("AF PRESENT")
	}

	loc, rsp, err = af.afAddSubscription(nefCtx, ti)

	if err != nil {
		return loc, rsp, err
	}

	return loc, rsp, nil
}

// ReadAllTrafficInfluenceSubscription : API to read all the subscritions
func ReadAllTrafficInfluenceSubscription(w http.ResponseWriter,
	r *http.Request) {

	var subslist []TrafficInfluSub
	var rsp nefSBRspData
	var err error

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID : %s", vars["afId"])

	af, err := nef.nefGetAf(vars["afId"])

	if err != nil {
		/* Failure in getting AF with afId received. In this case no
		 * subscription data will be returned to AF */
		log.Infoln(err)
	} else {
		rsp, subslist, err = af.afGetSubscriptionList(nefCtx)
		if err != nil {
			log.Err(err)
			sendErrorResponseToAF(w, rsp)
			return
		}
	}

	mdata, err2 := json.Marshal(subslist)

	if err2 != nil {
		sendCustomeErrorRspToAF(w, 400, "Failed to MARSHAL Subscription data ")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//Send Success response to Network
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}

	log.Infof("HTTP Response sent: %d", http.StatusOK)
}

// CreateTrafficInfluenceSubscription : Handles the traffic influence requested
// by AF
func CreateTrafficInfluenceSubscription(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["afId"])

	b, err := ioutil.ReadAll(r.Body)
	defer closeReqBody(r)

	if err != nil {
		sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP POST Body")
		return
	}

	//Traffic Influence data
	trInBody := TrafficInfluSub{}

	//Convert the json Traffic Influence data into struct
	err1 := json.Unmarshal(b, &trInBody)

	if err1 != nil {
		log.Err(err1)
		sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal POST data")
		return
	}

	//validate the mandatory parameters
	resRsp, status := validateAFTrafficInfluenceData(trInBody)
	if !status {
		log.Err(resRsp.pd.Title)
		sendErrorResponseToAF(w, resRsp)
		return
	}

	loc, rsp, err3 := createNewSub(nefCtx, vars["afId"], trInBody)

	if err3 != nil {
		log.Err(err3)
		// we return bad request here since we have reached the max
		rsp.errorCode = 400
		sendErrorResponseToAF(w, rsp)
		return
	}
	log.Infoln(loc)

	trInBody.Self = Link(loc)
	Emulator_file := os.Getenv("Emulator_path") + "/on"
	cmd := exec.Command("touch", Emulator_file)
	go cmd.Run()
	/*
	if err != nil {
		log.Err("Offloading failed!")
		return
	}
	*/

	//Martshal data and send into the body
	mdata, err2 := json.Marshal(trInBody)

	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GET response data")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Location", loc)

	// Response should be 201 Created as per 3GPP 29.522
	w.WriteHeader(http.StatusCreated)
	log.Infof("CreateTrafficInfluenceSubscription responses => %d",
		http.StatusCreated)
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}
	nef := &nefCtx.nef
	logNef(nef)

}

// ReadTrafficInfluenceSubscription : Read a particular subscription details
func ReadTrafficInfluenceSubscription(w http.ResponseWriter, r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["afId"])
	log.Infof(" SUBSCRIPTION ID  : %s", vars["subscriptionId"])

	af, ok := nef.nefGetAf(vars["afId"])

	if ok != nil {
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")
		return
	}

	rsp, sub, err := af.afGetSubscription(nefCtx, vars["subscriptionId"])

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	mdata, err2 := json.Marshal(sub)
	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GET response data")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}

	log.Infof("HTTP Response sent: %d", http.StatusOK)
}

// UpdatePutTrafficInfluenceSubscription : Updates a traffic influence created
// earlier (PUT Req)
func UpdatePutTrafficInfluenceSubscription(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["afId"])
	log.Infof(" SUBSCRIPTION ID  : %s", vars["subscriptionId"])

	af, ok := nef.nefGetAf(vars["afId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//Traffic Influence data
		trInBody := TrafficInfluSub{}

		//Convert the json Traffic Influence data into struct
		err1 := json.Unmarshal(b, &trInBody)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		rsp, newTI, err := af.afUpdateSubscription(nefCtx,
			vars["subscriptionId"], trInBody)

		if err != nil {
			sendErrorResponseToAF(w, rsp)
			return
		}

		mdata, err2 := json.Marshal(newTI)

		if err2 != nil {
			log.Err(err2)
			sendCustomeErrorRspToAF(w, 400, "Failed to Marshal PUT"+
				"response data")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		_, err = w.Write(mdata)
		if err != nil {
			log.Errf("Write Failed: %v", err)
		}
		return

	}
	log.Infoln(ok)
	sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")

}

// UpdatePatchTrafficInfluenceSubscription : Updates a traffic influence created
//  earlier (PATCH Req)
func UpdatePatchTrafficInfluenceSubscription(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["afId"])
	log.Infof(" SUBSCRIPTION ID  : %s", vars["subscriptionId"])

	af, ok := nef.nefGetAf(vars["afId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)

		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PATCH Body")
			return
		}

		//Traffic Influence Sub Patch data
		TrInSPBody := TrafficInfluSubPatch{}

		//Convert the json Traffic Influence data into struct
		err1 := json.Unmarshal(b, &TrInSPBody)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PATCH data")
			return
		}

		rsp, ti, err := af.afPartialUpdateSubscription(nefCtx,
			vars["subscriptionId"], TrInSPBody)

		if err != nil {
			sendErrorResponseToAF(w, rsp)
			return
		}

		mdata, err2 := json.Marshal(ti)

		if err2 != nil {
			log.Err(err2)
			sendCustomeErrorRspToAF(w, 400,
				"Failed to Marshal PATCH response data")
			return

		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		_, err = w.Write(mdata)
		if err != nil {
			log.Errf("Write Failed: %v", err)
		}
		return
	}

	log.Infoln(ok)
	sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")
}

// DeleteTrafficInfluenceSubscription : Deletes a traffic influence created by
//  AF
func DeleteTrafficInfluenceSubscription(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["afId"])
	log.Infof(" SUBSCRIPTION ID  : %s", vars["subscriptionId"])

	af, err := nef.nefGetAf(vars["afId"])

	if err != nil {
		log.Err(err)
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF entry")
		return
	}
	rsp, err := af.afDeleteSubscription(nefCtx, vars["subscriptionId"])

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	// Response should be 204 as per 3GPP 29.522
	w.WriteHeader(http.StatusNoContent)

	Emulator_file := os.Getenv("Emulator_path") + "/on"
	cmd := exec.Command("rm", Emulator_file)
	go cmd.Run()

	log.Infof("HTTP Response sent: %d", http.StatusNoContent)

	if af.afGetSubCount() == 0 {

		_ = nef.nefDeleteAf(vars["afId"])
	}

	logNef(nef)
}

func closeReqBody(r *http.Request) {
	err := r.Body.Close()
	if err != nil {
		log.Errf("response body was not closed properly")
	}
}

// NotifySmfUPFEvent : Handles the SMF notification for UPF event
func NotifySmfUPFEvent(w http.ResponseWriter,
	r *http.Request) {

	var (
		smfEv    NsmfEventExposureNotification
		ev       EventNotification
		afURL    URI
		nsmEvNo  NsmEventNotification
		i        int
		upfFound bool
	)

	if r.Body == nil {
		log.Errf("NotifySmfUPFEvent Empty Body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Retrieve the event notification information from the request
	if err := json.NewDecoder(r.Body).Decode(&smfEv); err != nil {
		log.Errf("NotifySmfUPFEvent body parse: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate the content of the NsmfEventExposureNotification
	// Check if notification id is present
	if smfEv.NotifID == "" {
		log.Errf("NotifySmfUPFEvent missing notif id")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if notification events with UP_PATH_CH is present
	if len(smfEv.EventNotifs) == 0 {
		log.Errf("NotifySmfUPFEvent missing event notifications")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for i, nsmEvNo = range smfEv.EventNotifs {
		if nsmEvNo.Event == "UP_PATH_CH" {
			log.Infof("NotifySmfUPFEvent found an entry for UP_PATH_CH"+
				"at index: %d", i)
			upfFound = true
			break
		}

	}

	if !upfFound {
		log.Errf("NotifySmfUPFEvent missing event with UP_PATH_CH")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Map the content of NsmfEventExposureNotification to EventNotificaiton
	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	afSubs, err1 := getSubFromCorrID(nefCtx, smfEv.NotifID)
	if err1 != nil {
		log.Errf("NotifySmfUPFEvent getSubFromCorrId [%s]: %s",
			smfEv.NotifID, err1.Error())
		w.WriteHeader(http.StatusNotFound)
		return
	}
	log.Infof("NotifySmfUPFEvent [NotifID, TransId, URL] => [%s,%s,%s",
		smfEv.NotifID, afSubs.ti.AfTransID,
		afSubs.ti.NotificationDestination)

	ev.AfTransID = afSubs.ti.AfTransID
	afURL = URI(afSubs.ti.NotificationDestination)
	ev.Gpsi = nsmEvNo.Gpsi
	ev.DnaiChgType = nsmEvNo.DnaiChgType
	ev.SrcUeIpv4Addr = nsmEvNo.SourceUeIpv4Addr
	ev.SrcUeIpv6Prefix = nsmEvNo.SourceUeIpv6Prefix
	ev.TgtUeIpv4Addr = nsmEvNo.TargetUeIpv4Addr
	ev.TgtUeIpv6Prefix = nsmEvNo.TargetUeIpv6Prefix
	ev.UeMac = nsmEvNo.UeMac
	ev.SourceTrafficRoute = nsmEvNo.SourceTraRouting
	ev.SubscribedEvent = SubscribedEvent("UP_PATH_CHANGE")
	ev.TargetTrafficRoute = nsmEvNo.TargetTraRouting

	w.WriteHeader(http.StatusOK)

	// Send the request towards AF
	var afClient AfNotification = NewAfClient(&nefCtx.cfg)
	err := afClient.AfNotificationUpfEvent(r.Context(), afURL, ev)
	if err != nil {
		log.Errf("NotifySmfUPFEvent sending to AF failed : %s",
			err.Error())
	}
}

func sendCustomeErrorRspToAF(w http.ResponseWriter, eCode int,
	custTitleString string) {

	eRsp := nefSBRspData{errorCode: eCode,
		pd: ProblemDetails{Title: custTitleString}}

	sendErrorResponseToAF(w, eRsp)

}
func sendErrorResponseToAF(w http.ResponseWriter, rsp nefSBRspData) {

	mdata, eCode := createErrorJSON(rsp)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(eCode)
	_, err := w.Write(mdata)
	if err != nil {
		log.Err("NEF ERROR : Failed to send response to AF !!!")
	}
	log.Infof("HTTP Response sent: %d", eCode)
}

func createErrorJSON(rsp nefSBRspData) (mdata []byte, statusCode int) {

	var err error
	statusCode = 404

	/*
		TBD for future: Removed check for 401, 403, 413 and 429
		due cyclometrix complexity lint warning. Once a better mechanism
		is found to be added back. Anyhow currently these errors are not
		supported
	*/

	if rsp.errorCode == 400 || rsp.errorCode == 404 || rsp.errorCode == 411 ||
		rsp.errorCode == 415 || rsp.errorCode == 500 || rsp.errorCode == 503 {
		statusCode = rsp.errorCode
		mdata, err = json.Marshal(rsp.pd)

		if err == nil {
			/*No return */
			log.Info(statusCode)
			return mdata, statusCode
		}
	}
	/*Send default error */
	pd := ProblemDetails{Title: " NEF Error "}

	mdata, err = json.Marshal(pd)

	if err != nil {
		return mdata, statusCode
	}
	/*Any case return mdata */
	return mdata, statusCode
}

func logNef(nef *nefData) {

	log.Infof("AF count %+v", len(nef.afs))
	if len(nef.afs) > 0 {
		for key, value := range nef.afs {
			log.Infof(" AF ID : %+v, Sub Registered Count %+v",
				key, len(value.subs))
			for _, vs := range value.subs {
				log.Infof("   SubId : %+v, ServiceId: %+v", vs.subid,
					vs.ti.AfServiceID)
			}

		}
	}

}

func getSubFromCorrID(nefCtx *nefContext, corrID string) (sub *afSubscription,
	err error) {

	nef := &nefCtx.nef

	/*Search across all the AF registered */
	for _, value := range nef.afs {

		/*Search across all the Subscription*/
		for _, vs := range value.subs {

			if vs.NotifCorreID == corrID {
				/*Match found return sub handle*/
				return vs, nil
			}
		}
	}
	return sub, errors.New("Subscription Not Found")
}

//validateAFTrafficInfluenceData: Function to validate mandatory parameters of
//TrafficInfluence received from AF
func validateAFTrafficInfluenceData(ti TrafficInfluSub) (rsp nefSBRspData,
	status bool) {

	if len(ti.AfTransID) == 0 {
		rsp.errorCode = 400
		rsp.pd.Title = "Missing AfTransID atttribute"
		return rsp, false
	}

	//In case AfServiceID  is not present then DNN has to be included in TI
	if len(ti.AfServiceID) == 0 && len(ti.Dnn) == 0 {

		rsp.errorCode = 400
		rsp.pd.Title = "Missing afServiceId atttribute"
		return rsp, false
	}

	if len(ti.AfAppID) == 0 && ti.TrafficFilters == nil &&
		ti.EthTrafficFilters == nil {
		rsp.errorCode = 400
		rsp.pd.Title = "missing one of afAppId, trafficFilters," +
			"ethTrafficFilters"
		return rsp, false
	}
	return rsp, true
}
