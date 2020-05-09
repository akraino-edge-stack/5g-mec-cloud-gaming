/* SPDX-License-Identifier: Apache-2.0
* Copyright (c) 2020 Intel Corporation
 */

package ngcnef

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	//"strconv"

	"github.com/gorilla/mux"
)

// TestNEFSB is the Test variable for injecting errors in NEF SB APIs
var TestNEFSB = false

func createNewPFDTrans(nefCtx *nefContext, afID string,
	trans PfdManagement) (loc string, rsp map[string]nefPFDSBRspData,
	err error) {

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

	loc, rsp, err = af.afAddPFDTransaction(nefCtx, trans)

	if err != nil {
		return loc, rsp, err
	}

	return loc, rsp, nil
}

// ReadAllPFDManagementTransaction : API to read all the PFD Transactions
func ReadAllPFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	var pfdTrans []PfdManagement
	var rsp nefPFDSBRspData
	var err error

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID : %s", vars["scsAsId"])

	af, err := nef.nefGetAf(vars["scsAsId"])

	if err != nil {
		/* Failure in getting AF with afId received. In this case no
		 * transaction data will be returned to AF */
		log.Infoln(err)
	} else {
		rsp, pfdTrans, err = af.afGetPfdTransactionList(nefCtx)
		if err != nil {
			log.Err(err)
			rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
			sendErrorResponseToAF(w, rsp1)
			return
		}
	}

	mdata, err2 := json.Marshal(pfdTrans)

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

// CreatePFDManagementTransaction  Handles the PFD Management requested
// by AF
func CreatePFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])

	b, err := ioutil.ReadAll(r.Body)
	defer closeReqBody(r)

	if err != nil {
		sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP POST Body")
		return
	}

	//Pfd Management data
	pfdBody := PfdManagement{}

	//Convert the json Traffic Influence data into struct
	err1 := json.Unmarshal(b, &pfdBody)

	if err1 != nil {
		log.Err(err1)
		sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal POST data")
		return
	}
	pfdBody.PfdReports = make(map[string]PfdReport)

	// Validate the mandatory parameters and generate Pfd Report if  failure
	resRsp, status := validateAFPfdManagementData(nefCtx, pfdBody, "POST")
	if !status {
		log.Err(resRsp.result.pd.Title)
		rsp1 := nefSBRspData{errorCode: resRsp.result.errorCode,
			pd: resRsp.result.pd}
		sendErrorResponseToAF(w, rsp1)
		return
	}

	// All PFDs have failed, 500 response
	if len(pfdBody.PfdDatas) == 0 {

		pd := ProblemDetails{Title: pfdAppsFailed}
		rsp1 := nefSBRspData{errorCode: 500, pd: pd}
		sendPFDErrorResponseToAF(w, rsp1, "", pfdBody.PfdReports)
		return
	}
	loc, _, err3 := createNewPFDTrans(nefCtx, vars["scsAsId"], pfdBody)

	if err3 != nil {
		log.Err(err3)

		// All PFDs failed at UDR
		if err3.Error() == pfdAppsFailed {

			// fatal error return
			rsp1 := nefSBRspData{errorCode: 500}
			rsp1.pd.Title = pfdAppsFailed

			sendPFDErrorResponseToAF(w, rsp1, "", pfdBody.PfdReports)
			return
		}
		// fatal error return
		rsp1 := nefSBRspData{errorCode: 400}
		rsp1.pd.Title = "UDR Error in creating PFD transaction"
		sendErrorResponseToAF(w, rsp1)

		return
	}

	log.Infoln(loc)

	pfdBody.Self = Link(loc)

	log.Info(pfdBody)
	//Martshal data and send into the body
	mdata, err2 := json.Marshal(pfdBody)

	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GET response data")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Location", loc)

	// Response should be 201 Created as per 3GPP 29.522
	w.WriteHeader(http.StatusCreated)
	log.Infof("CreatePFDManagementresponses => %d",
		http.StatusCreated)
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}
	// Deleting the PFD reports from the stored transactions once sent
	afID := vars["scsAsId"]
	transID := strconv.Itoa((nef.afs[afID].transIDnum) - 1)

	for k := range nef.afs[afID].pfdtrans[transID].pfdManagement.PfdReports {
		delete(nef.afs[afID].pfdtrans[transID].pfdManagement.PfdReports, k)
	}
	logNef(nef)
}

// ReadPFDManagementTransaction : Read a particular PFD transaction details
func ReadPFDManagementTransaction(w http.ResponseWriter, r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" TRANSACTION ID  : %s", vars["transactionId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])

	if ok != nil {
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")
		return
	}

	rsp, pfdTrans, err := af.afGetPfdTransaction(nefCtx, vars["transactionId"])

	if err != nil {
		log.Err(err)
		rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
		sendErrorResponseToAF(w, rsp1)
		return
	}

	mdata, err2 := json.Marshal(pfdTrans)
	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GETPFDresponse data")
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

// ReadPFDManagementApplication : Read a particular PFD transaction details of
// and external application identifier
func ReadPFDManagementApplication(w http.ResponseWriter, r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID : %s", vars["appId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])

	if ok != nil {
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")
		return
	}

	pfdTransID := vars["transactionId"]
	appID := vars["appId"]
	rsp, pfdData, err := af.afGetPfdApplication(nefCtx, pfdTransID, appID)

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	mdata, err2 := json.Marshal(pfdData)
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

// DeletePFDManagementApplication deletes the existing PFD transaction of
// an application identifier
func DeletePFDManagementApplication(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID  : %s", vars["appId"])

	af, err := nef.nefGetAf(vars["scsAsId"])

	if err != nil {
		log.Err(err)
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF entry")
		return
	}

	pfdTransID := vars["transactionId"]
	appID := vars["appId"]

	rsp, err := af.afDeletePfdApplication(nefCtx, pfdTransID, appID)

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	nef.nefCheckDeleteAf(vars["scsAsId"])
	// Response should be 204 as per 3GPP 29.522
	w.WriteHeader(http.StatusNoContent)

	log.Infof("HTTP Response sent: %d", http.StatusNoContent)

	logNef(nef)
}

// DeletePFDManagementTransaction deletes the existing PFD transaction
func DeletePFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])

	af, err := nef.nefGetAf(vars["scsAsId"])

	if err != nil {
		log.Err(err)
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF entry")
		return
	}
	rsp, err := af.afDeletePfdTransaction(nefCtx, vars["transactionId"])

	if err != nil {
		log.Err(err)
		rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
		sendErrorResponseToAF(w, rsp1)
		return
	}

	nef.nefCheckDeleteAf(vars["scsAsId"])
	// Response should be 204 as per 3GPP 29.522
	w.WriteHeader(http.StatusNoContent)

	log.Infof("HTTP Response sent: %d", http.StatusNoContent)

	logNef(nef)
}

func (af *afData) afGetPfdTransCount() (afPfdCount int) {

	return len(af.pfdtrans)
}

// UpdatePutPFDManagementTransaction updates an existing PFD transaction
func UpdatePutPFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//PFD Transaction data
		pfdTrans := PfdManagement{}

		//Convert the json PFD Management data into struct
		err1 := json.Unmarshal(b, &pfdTrans)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		pfdTrans.PfdReports = make(map[string]PfdReport)
		// Validate the mandatory parameters and generate pfd reports if
		// failure
		resRsp, status := validateAFPfdManagementData(nefCtx, pfdTrans, "PUT")
		if !status {
			log.Err(resRsp.result.pd.Title)
			rsp1 := nefSBRspData{errorCode: resRsp.result.errorCode}
			sendErrorResponseToAF(w, rsp1)
			return
		}

		_, newPfdTrans, err := af.afUpdatePutPfdTransaction(nefCtx,
			vars["transactionId"], pfdTrans)

		if err != nil {
			log.Err(err)
			// All PFDs failed at UDR
			if err.Error() == pfdAppsFailed {

				// fatal error return
				rsp1 := nefSBRspData{errorCode: 500}
				rsp1.pd.Title = "ALL PFD APP FAILED TO PROVISION"

				sendPFDErrorResponseToAF(w, rsp1, "", pfdTrans.PfdReports)
				return
			}

			// fatal error return
			rsp1 := nefSBRspData{errorCode: 400}
			rsp1.pd.Title = "UDR Error in updating PFD transaction"
			sendErrorResponseToAF(w, rsp1)

			return
		}

		mdata, err2 := json.Marshal(newPfdTrans)

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

// UpdatePutPFDManagementApplication updates an existing PFD transaction
func UpdatePutPFDManagementApplication(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID  : %s", vars["appId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//PFD Transaction data
		pfdData := PfdData{}

		pfdReportList := make(map[string]PfdReport)

		//Convert the json PFD Management data into struct
		err1 := json.Unmarshal(b, &pfdData)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		for _, pfd := range pfdData.Pfds {
			rsp, status := validateAFPfdData(pfd)
			if !status {
				log.Err(rsp.result.pd.Title)
				rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
				sendErrorResponseToAF(w, rsp1)
				return

			}
		}

		rsp, newPfdData, err := af.afUpdatePutPfdApplication(nefCtx,
			vars["transactionId"], vars["appId"], pfdData, pfdReportList)

		if err != nil {
			log.Err(err)

			if err.Error() == pfdAppsFailed {
				// fatal error return
				rsp1 := nefSBRspData{errorCode: 500}
				rsp1.pd.Title = pfdAppsFailed

				sendPFDErrorResponseToAF(w, rsp1, "SINGLE_APP",
					pfdReportList)
				return

			}
			// fatal error return
			rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
			rsp1.pd.Title = "UDR Error in updating PFD Application"
			sendErrorResponseToAF(w, rsp1)

			return
		}

		mdata, err2 := json.Marshal(newPfdData)

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

// PatchPFDManagementApplication patches the PFD application PFDs
func PatchPFDManagementApplication(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID  : %s", vars["appId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//PFD Transaction data
		pfdData := PfdData{}
		pfdReportList := make(map[string]PfdReport)

		//Convert the json PFD Management data into struct
		err1 := json.Unmarshal(b, &pfdData)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		for _, pfd := range pfdData.Pfds {
			rsp, status := validateAFPfdData(pfd)
			if !status {
				log.Infof("PFD %s is invalid in Application %s", pfd.PfdID,
					pfdData.ExternalAppID)
				rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
				sendErrorResponseToAF(w, rsp1)
				return
			}
		}

		rsp, newPfdData, err := af.afUpdatePatchPfdApplication(nefCtx,
			vars["transactionId"], vars["appId"], pfdData, pfdReportList)

		if err != nil {
			log.Err(err)
			if err.Error() == pfdAppsFailed {
				// fatal error return
				rsp1 := nefSBRspData{errorCode: 500}
				rsp1.pd.Title = pfdAppsFailed

				sendPFDErrorResponseToAF(w, rsp1, "SINGLE_APP",
					pfdReportList)
				return

			}
			// fatal error return
			rsp1 := nefSBRspData{errorCode: rsp.result.errorCode}
			rsp1.pd.Title = "UDR Error in updating PFD Application"
			sendErrorResponseToAF(w, rsp1)

			return
		}

		mdata, err2 := json.Marshal(newPfdData)

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

//PFD Management functions

func (af *afData) afUpdatePutPfdApplication(nefCtx *nefContext, transID string,
	appID string, pfdData PfdData,
	pfdReportList map[string]PfdReport) (rsp nefPFDSBRspData, updPfd PfdData,
	err error) {

	pfdTrans, ok := af.pfdtrans[transID]

	if !ok {
		rsp.result.errorCode = 404
		rsp.result.pd.Title = pfdNotFound

		return rsp, updPfd, errors.New(pfdNotFound)
	}
	trans, ok := pfdTrans.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.result.errorCode = 404
		rsp.result.pd.Title = appNotFound
		return rsp, trans, errors.New(appNotFound)
	}

	rsp, err = pfdTrans.NEFSBAppPfdPut(pfdTrans, nefCtx, pfdData)

	if err != nil {
		log.Err("Failed to Update the PFD Application")
		return rsp, pfdData, err
	}

	err = updatePfdAppOnRsp(rsp, pfdData, appID, pfdReportList)

	if err != nil {

		return rsp, updPfd, err

	}

	pfdData.Self = trans.Self
	pfdTrans.pfdManagement.PfdDatas[appID] = pfdData

	updPfd = pfdData

	log.Infoln("Update PFD transaction Successful")
	return rsp, updPfd, err
}

func (af *afData) afUpdatePatchPfdApplication(nefCtx *nefContext,
	transID string, appID string, pfdData PfdData,
	pfdReportList map[string]PfdReport) (rsp nefPFDSBRspData,
	updPfd PfdData, err error) {

	pfdTrans, ok := af.pfdtrans[transID]

	if !ok {
		rsp.result.errorCode = 404
		rsp.result.pd.Title = pfdNotFound

		return rsp, updPfd, errors.New(pfdNotFound)
	}

	trans, ok := pfdTrans.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.result.errorCode = 404
		rsp.result.pd.Title = appNotFound
		return rsp, trans, errors.New(appNotFound)

	}

	rsp, err = pfdTrans.NEFSBAppPfdPut(pfdTrans, nefCtx, pfdData)

	if err != nil {
		log.Err("Failed to Update the PFD Application")
		return rsp, pfdData, err
	}

	err = updatePfdAppOnRsp(rsp, pfdData, appID, pfdReportList)

	if err != nil {

		return rsp, updPfd, errors.New(pfdAppsFailed)

	}

	// Updating the PFDs present in the
	for key := range pfdData.Pfds {

		pfd, ok := trans.Pfds[key]
		if ok {
			pfd = pfdData.Pfds[key]
			trans.Pfds[key] = pfd
			log.Infof("PFD id %s updated by PATCH ", pfd.PfdID)
		}

	}
	pfdData.Self = trans.Self
	pfdTrans.pfdManagement.PfdDatas[appID] = pfdData

	updPfd = pfdData

	log.Infoln("Patch PFD Application PFDs Successful")
	return rsp, updPfd, err
}

func (af *afData) afUpdatePutPfdTransaction(nefCtx *nefContext, transID string,
	trans PfdManagement) (rsp map[string]nefPFDSBRspData, updPfd PfdManagement,
	err error) {

	pfdTrans, ok := af.pfdtrans[transID]

	if !ok {

		return rsp, updPfd, errors.New(pfdNotFound)
	}

	// PUT should not have any new application
	for key := range trans.PfdDatas {
		_, ok = pfdTrans.pfdManagement.PfdDatas[key]
		if !ok {
			//Return error
			return rsp, trans, errors.New("Application not present")
		}
	}

	rsp, err = pfdTrans.NEFSBPfdPut(pfdTrans, nefCtx, trans)

	if err != nil {

		//Return error
		return rsp, trans, err
	}
	err = updatePfdTransOnRsp(rsp, trans)

	if err != nil {

		return rsp, updPfd, errors.New(pfdAppsFailed)

	}
	updPfd = trans
	updPfd.Self = pfdTrans.pfdManagement.Self
	for key, v := range updPfd.PfdDatas {
		v.Self = pfdTrans.pfdManagement.PfdDatas[key].Self
		updPfd.PfdDatas[key] = v
	}
	pfdTrans.pfdManagement = updPfd

	log.Infoln("Update PFD transaction Successful")
	return rsp, updPfd, err
}

func (af *afData) afDeletePfdTransaction(nefCtx *nefContext,
	pfdTrans string) (rsp nefPFDSBRspData, err error) {

	//Check if PFD transaction is already present
	trans, ok := af.pfdtrans[pfdTrans]

	if !ok {
		rsp.result.errorCode = 404
		rsp.result.pd.Title = pfdNotFound
		return rsp, errors.New(pfdNotFound)
	}

	rsp, err = trans.NEFSBPfdDelete(trans, nefCtx)
	if err != nil {
		log.Err("Failed to Delete PFD transaction")
		rsp.result.errorCode = 400
		return rsp, err
	}

	//Delete local entry in map of pfd transactions
	delete(af.pfdtrans, pfdTrans)

	// TBD check if all trans and sub deleted for AF then delete AF

	return rsp, err
}

func (af *afData) afGetPfdApplication(nefCtx *nefContext,
	transID string, appID string) (rsp nefSBRspData, trans PfdData, err error) {

	transPfd, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = pfdNotFound
		return rsp, trans, errors.New(pfdNotFound)
	}

	trans, ok = transPfd.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = appNotFound
		return rsp, trans, errors.New(appNotFound)
	}

	//Return locally
	return rsp, trans, err
}

func (af *afData) afDeletePfdApplication(nefCtx *nefContext,
	transID string, appID string) (rsp nefSBRspData, err error) {

	transPfd, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = pfdNotFound
		return rsp, errors.New(pfdNotFound)
	}

	_, ok = transPfd.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = appNotFound
		return rsp, errors.New(appNotFound)
	}

	delete(transPfd.pfdManagement.PfdDatas, appID)

	// If all apps in trans are deleted, delete the trans
	if len(transPfd.pfdManagement.PfdDatas) == 0 {
		delete(af.pfdtrans, transID)
	}

	return rsp, err
}

func (af *afData) afGetPfdTransaction(nefCtx *nefContext,
	transID string) (rsp nefPFDSBRspData, trans PfdManagement, err error) {

	transPfd, ok := af.pfdtrans[transID]

	if !ok {
		rsp.result.errorCode = 404
		rsp.result.pd.Title = pfdNotFound
		return rsp, trans, errors.New(pfdNotFound)
	}

	_, rsp, err = transPfd.NEFSBPfdGet(transPfd, nefCtx)
	if err != nil {
		log.Infoln("Failed to Get PFD transaction")
		return rsp, transPfd.pfdManagement, err
	}

	//Return locally
	return rsp, transPfd.pfdManagement, err
}

func (af *afData) afGetPfdTransactionList(nefCtx *nefContext) (
	rsp nefPFDSBRspData, transList []PfdManagement, err error) {

	var transPfd PfdManagement

	if len(af.pfdtrans) > 0 {

		for key := range af.pfdtrans {

			rsp, transPfd, err = af.afGetPfdTransaction(nefCtx, key)

			if err != nil {
				return rsp, transList, err
			}
			transList = append(transList, transPfd)
		}
	}
	return rsp, transList, err
}

//Creates a new PFD Transaction
func (af *afData) afAddPFDTransaction(nefCtx *nefContext,
	trans PfdManagement) (loc string, rsp map[string]nefPFDSBRspData,
	err error) {

	rsp = make(map[string]nefPFDSBRspData)
	/*Check if max subscription reached */
	if len(af.pfdtrans) >= nefCtx.cfg.MaxPfdTransSupport {

		return "", rsp, errors.New("MAX TRANS Created")
	}

	//Generate a unique transaction ID string
	transIDStr := strconv.Itoa(af.transIDnum)
	af.transIDnum++

	//Create PFD transaction data
	aftrans := afPfdTransaction{transID: transIDStr, pfdManagement: trans}

	aftrans.NEFSBPfdGet = nefSBUDRPFDGet
	aftrans.NEFSBAppPfdPut = nefSBUDRAPPPFDPut
	aftrans.NEFSBPfdPut = nefSBUDRPFDPut
	aftrans.NEFSBPfdDelete = nefSBUDRPFDDelete

	rsp, err = aftrans.NEFSBPfdPut(&aftrans, nefCtx, trans)

	if err != nil {
		//Return fatal  error
		return "", rsp, err
	}

	// Check the rsp from UDR and generate PFD reports ,
	// if all failed then err (500)
	err = updatePfdTransOnRsp(rsp, trans)

	if err != nil {
		return "", rsp, errors.New(pfdAppsFailed)
	}

	//Link the PFD transaction with the AF
	af.pfdtrans[transIDStr] = &aftrans

	//Create Location URI
	loc = nefCtx.nef.locationURLPrefixPfd + af.afID + "/transactions/" +
		transIDStr

	af.pfdtrans[transIDStr].pfdManagement.Self = Link(loc)

	//Also update the self link in each application
	for k, v := range af.pfdtrans[transIDStr].pfdManagement.PfdDatas {

		/*Assign the application ID in the link */
		v.Self = Link(loc) + "/applications/" + Link(k)
		log.Infof("Application ID is %s", k)
		af.pfdtrans[transIDStr].pfdManagement.PfdDatas[k] = v

	}

	log.Infoln(" NEW AF PFD transaction added " + transIDStr)

	return loc, rsp, nil
}

// Generate the notification uri for PFD
func getNefLocationURLPrefixPfd(cfg *Config) string {

	var uri string
	// If http2 port is configured use it else http port
	if cfg.HTTP2Config.Endpoint != "" {
		uri = "https://" + cfg.NefAPIRoot +
			cfg.HTTP2Config.Endpoint
	} else {
		uri = "http://" + cfg.NefAPIRoot +
			cfg.HTTPConfig.Endpoint
	}
	uri += cfg.LocationPrefixPfd
	return uri

}

/* validateAFPfdManagementData Function to validate mandatory parameters of
PFD Management received from AF

	- Atleast one PfdData should be present
	- appID in pfdData should be present
	- If the method is POST, check the appID should be unique
	- pfdID should be present in PFD
	- In PFD, only one of the params - FlowDescription, urls or domainnames
	   should be present
*/
func validateAFPfdManagementData(nefCtx *nefContext,
	pfdTrans PfdManagement, method string) (rsp nefPFDSBRspData, status bool) {

	if len(pfdTrans.PfdDatas) == 0 {
		rsp.result.errorCode = 400
		rsp.result.pd.Title = "Missing PFD Data"
		return rsp, false
	}

	//Validation of Individual PFD
	for _, v := range pfdTrans.PfdDatas {
		if len(v.Pfds) == 0 {
			rsp.result.errorCode = 400
			rsp.result.pd.Title = "Missing PFD Data"
			return rsp, false
		}

		if len(v.ExternalAppID) == 0 {
			rsp.result.errorCode = 400
			rsp.result.pd.Title = "Missing Application ID"
			return rsp, false
		}

	}

	//Validation of parameters in PfdDatas where AppID is the key
	for key, pfdData := range pfdTrans.PfdDatas {

		if method == "POST" {
			// Validate for Duplicate Application ID
			if nefCheckPfdAppIDExists(key, nefCtx) {

				// Prepare Pfd report APP ID DUPLICATED
				log.Infof("Application ID %s Duplicate", key)
				generatePfdReport(key, "APP_ID_DUPLICATED", pfdTrans.PfdReports)
				delete(pfdTrans.PfdDatas, key)
			}
		}

		for _, pfd := range pfdData.Pfds {
			rsp1, status := validateAFPfdData(pfd)
			if !status {
				log.Infof("PFD %s is invalid in Application %s", pfd.PfdID, key)
				return rsp1, status

			}
		}

	}

	return rsp, true
}

// validateAFPfdData Function to validate mandatory parameters of
// PFD  received from AF
func validateAFPfdData(pfd Pfd) (rsp nefPFDSBRspData,
	status bool) {

	if len(pfd.PfdID) == 0 {
		rsp.result.errorCode = 400
		rsp.result.pd.Title = "PFD ID missing"
		log.Info(rsp.result.pd.Title)
		return rsp, false
	}
	if len(pfd.DomainNames) == 0 && len(pfd.FlowDescriptions) == 0 &&
		len(pfd.Urls) == 0 {
		rsp.result.errorCode = 400
		rsp.result.pd.Title = "No domainNames " +
			"FlowDescriptions and Urls present for PFD"
		log.Info(rsp.result.pd.Title)
		return rsp, false
	}
	return rsp, true
}

func nefSBUDRAPPPFDGet(transData *afPfdTransaction, nefCtx *nefContext,
	appID ApplicationID) (appPfd PfdData, rsp nefPFDSBRspData, err error) {

	log.Info("nefSBUDRAPPPFDGet Entered ")
	nef := &nefCtx.nef

	cliCtx, cancel := context.WithCancel(nef.ctx)
	defer cancel()

	r, e := nef.udrPfdClient.UdrPfdDataGet(cliCtx, UdrAppID(appID))
	if e != nil {

		return appPfd, rsp, e
	}
	appPfd.ExternalAppID = string(r.AppPfd.AppID)

	appPfd.Pfds = make(map[string]Pfd)
	for _, ele := range r.AppPfd.Pfds {
		var conData = Pfd{}

		conData.PfdID = ele.PfdID
		conData.DomainNames = ele.DomainNames
		conData.FlowDescriptions = ele.FlowDescriptions
		conData.Urls = ele.Urls

		appPfd.Pfds[ele.PfdID] = conData
	}
	log.Info("nefSBUDRAPPPFDGet Exited ")

	return appPfd, rsp, err
}

//NEFSBGetPfdFn is the callback for SB API to get PFD transaction
func nefSBUDRPFDGet(transData *afPfdTransaction, nefCtx *nefContext) (
	trans PfdManagement, rsp nefPFDSBRspData, err error) {

	trans.PfdDatas = make(map[string]PfdData)

	for k, v := range transData.pfdManagement.PfdDatas {
		appPfd, r, e := nefSBUDRAPPPFDGet(transData, nefCtx,
			ApplicationID(v.ExternalAppID))
		if e != nil {
			return trans, r, e
		}

		trans.PfdDatas[k] = appPfd
	}

	return trans, rsp, nil
}

// NEFSBAppPutPfdFn is the callback for SB API to put PFD transaction
func nefSBUDRAPPPFDPut(transData *afPfdTransaction, nefCtx *nefContext,
	app PfdData) (rsp nefPFDSBRspData, err error) {

	nef := &nefCtx.nef
	var pfdApp PfdDataForApp

	cliCtx, cancel := context.WithCancel(nef.ctx)
	defer cancel()

	/*Timer for pfdApp.AllowedDelay can be started here, not supported
	currently */
	pfdApp.AppID = ApplicationID(app.ExternalAppID)

	log.Info("nefSBUDRAPPPFDPut ->  ")
	if app.CachingTime != nil {

		i := time.Duration(*app.CachingTime)
		timeLater := DateTime(time.Now().Add(time.Second * i).String())
		pfdApp.CachingTime = &timeLater
	}

	for _, pfdv := range app.Pfds {

		var c PfdContent
		c.PfdID = pfdv.PfdID
		c.DomainNames = pfdv.DomainNames
		c.FlowDescriptions = pfdv.FlowDescriptions
		c.Urls = pfdv.Urls
		pfdApp.Pfds = append(pfdApp.Pfds, c)
	}

	_, e := nef.udrPfdClient.UdrPfdDataCreate(cliCtx, pfdApp)

	if e != nil {
		rsp.result.errorCode = 400
		return rsp, e
	}

	if TestNEFSB {
		rsp.result.errorCode = 400
		var fc FailureCode = OtherReason
		rsp.fc = &fc
	} else {
		rsp.result.errorCode = 200
	}
	return rsp, err

}

// nefSBUDRPFDPut is the callback for SB API to put PFD transaction
func nefSBUDRPFDPut(transData *afPfdTransaction, nefCtx *nefContext,
	trans PfdManagement) (rsp map[string]nefPFDSBRspData, err error) {

	rspDetails := make(map[string]nefPFDSBRspData)

	for k, v := range trans.PfdDatas {

		r, e := nefSBUDRAPPPFDPut(transData, nefCtx, v)
		if e != nil {
			//fatal error return
			return rspDetails, e
		}
		/*Update the map with the response*/
		rspDetails[k] = r
	}
	return rspDetails, nil
}

//NEFSBDeletePfdFn is the callback for SB API to delete PFD transaction
func nefSBUDRAPPPFDDelete(transData *afPfdTransaction, nefCtx *nefContext,
	appID ApplicationID) (rsp nefPFDSBRspData, err error) {

	nef := &nefCtx.nef
	cliCtx, cancel := context.WithCancel(nef.ctx)
	defer cancel()

	_, e := nef.udrPfdClient.UdrPfdDataDelete(cliCtx, UdrAppID(appID))

	if e != nil {
		return rsp, e
	}

	return rsp, nil
}

//NEFSBDeletePfdFn is the callback for SB API to delete PFD transaction
func nefSBUDRPFDDelete(transData *afPfdTransaction, nefCtx *nefContext) (
	rsp nefPFDSBRspData, err error) {

	for _, v := range transData.pfdManagement.PfdDatas {
		r, e := nefSBUDRAPPPFDDelete(transData, nefCtx,
			ApplicationID(v.ExternalAppID))
		if e != nil {
			return r, e
		}
	}
	return rsp, nil
}

func updatePfdTransOnRsp(rsp map[string]nefPFDSBRspData,
	pfdTrans PfdManagement) (err error) {
	for key, v := range rsp {
		if v.result.errorCode != 200 && v.fc == nil {
			delete(pfdTrans.PfdDatas, key)
			generatePfdReport(key, "OTHER_REASON", pfdTrans.PfdReports)
		} else if v.result.errorCode != 200 && v.fc != nil {
			delete(pfdTrans.PfdDatas, key)
			generatePfdReport(key, string(*(v.fc)), pfdTrans.PfdReports)

		}
	}
	if len(pfdTrans.PfdDatas) == 0 {
		return errors.New(pfdAppsFailed)
	}
	return err
}

func updatePfdAppOnRsp(rsp nefPFDSBRspData, pfdData PfdData,
	appID string, pfdReportList map[string]PfdReport) (err error) {

	if rsp.result.errorCode != 200 && rsp.fc == nil {
		generatePfdReport(appID, "OTHER_REASON", pfdReportList)
		return errors.New(pfdAppsFailed)
	} else if rsp.result.errorCode != 200 && rsp.fc != nil {
		generatePfdReport(appID, string(*(rsp.fc)), pfdReportList)
		return errors.New(pfdAppsFailed)

	}
	return err
}

func generatePfdReport(appID string,
	failureReason string, pfdReportList map[string]PfdReport) {

	switch failureReason {

	case "APP_ID_DUPLICATED":
		if _, ok := pfdReportList[failureReason]; !ok {
			// Create the first PFD report
			var appIds []string
			appIds = append(appIds, appID)
			pfdReport := PfdReport{ExternalAppIds: appIds,
				FailureCode: AppIDDuplicated}
			pfdReportList[failureReason] = pfdReport

		} else {
			pfdReport := pfdReportList[failureReason]
			pfdReport.ExternalAppIds = append(pfdReport.ExternalAppIds,
				appID)
			pfdReportList[failureReason] = pfdReport
		}
	case "OTHER_REASON":
		if _, ok := pfdReportList[failureReason]; !ok {
			// Create the first PFD report
			var appIds []string
			appIds = append(appIds, appID)
			pfdReport := PfdReport{ExternalAppIds: appIds,
				FailureCode: OtherReason}
			pfdReportList[failureReason] = pfdReport

		} else {
			pfdReport := pfdReportList[failureReason]
			pfdReport.ExternalAppIds = append(pfdReport.ExternalAppIds,
				appID)
			pfdReportList[failureReason] = pfdReport
		}

	case "MALFUNCTION":
		log.Infoln("MALFUNCTION failure Code is not handled")

	case "RESOURCE_LIMITATION":
		log.Infoln("RESOURCE_LIMITATION failure code not handled")

	case "SHORT_DELAY":
		log.Infoln("SHORT_DELAY failure code is not handled")
	}

}

func sendPFDErrorResponseToAF(w http.ResponseWriter,
	rsp nefSBRspData, method string, pfdReports map[string]PfdReport) {

	//mdata, eCode := createErrorJSON(rsp)

	var e error
	var body []byte
	var pfdReport PfdReport

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//w.Header().Add("Content-Type", "application/problem+json; charset=UTF-8")
	w.WriteHeader(rsp.errorCode)

	// If single APP UPdate then only one pfd report else array of reports
	if method == "SINGLE_APP" {
		for _, value := range pfdReports {
			pfdReport = value
		}
		body, e = json.Marshal(pfdReport)
	} else {
		//Making the array of PfdReport from map
		pfdList := make([]PfdReport, len(pfdReports))
		idx := 0
		for _, value := range pfdReports {
			pfdList[idx] = value
			idx++
		}
		body, e = json.Marshal(pfdList)
	}
	if e != nil {
		log.Err("NEF ERROR : Failed to send response to AF !!!")
	}
	_, err := w.Write(body)
	if err != nil {
		log.Err("NEF ERROR : Failed to send response to AF !!!")
	}
	/*
		_, err = w.Write(mdata)
		if err != nil {
			log.Err("NEF ERROR : Failed to send response to AF !!!")
		}
	*/
	log.Infof("HTTP Response sent: %d", rsp.errorCode)

}

func nefCheckPfdAppIDExists(appID string, nefCtx *nefContext) bool {

	nef := &nefCtx.nef
	for _, v := range nef.afs {
		for _, trans := range v.pfdtrans {
			for key := range trans.pfdManagement.PfdDatas {
				if key == appID {
					return true
				}

			}
		}
	}
	return false

}
