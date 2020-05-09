/* SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2019 Intel Corporation
 */

/**
 * @file Exception.cpp
 * @brief Implementation of Exception
 ********************************************************************/

#include "Exception.h"

void Exception::handlerException(Exception e, string &res, string &statusCode)
{
    switch (e.code)
    {
    // ADDED_USERPLANE
    case Exception::ADDED_USERPLANE:
        statusCode = HTTP_SC_ADDED_USERPLANE;
        res= "ADDED_USERPLANE";
        break;	    
    case Exception::INVALID_TYPE:
        statusCode = HTTP_SC_BAD_REQUEST;
        res= "ParameterInvalid";
        break;		
    case Exception::INVALID_PARAMETER:
        statusCode = HTTP_SC_BAD_REQUEST;
        res= "ParameterInvalid";
        break;
    case Exception::INVALID_UERPLANE_FUNCTION:
        res= "INVALID_UERPLANE_PROPERTISE";
        statusCode = HTTP_SC_INVALID_UERPLANE_PROPERTISE;
        break;
    case Exception::INVALID_DATA_SCHEMA:
    case Exception::USERPLANE_NOT_FOUND:
        statusCode = HTTP_SC_USERPLANE_NOT_FOUND;
        res = "USERPLANE_NOT_FOUND";
        break;
    case Exception::INTERNAL_SOFTWARE_ERROR:
        statusCode = HTTP_SC_INTERNAL_SOFTWARE_ERROR;
        res = "INTERNAL_SOFTWARE_ERROR";
        break;    
    case Exception::CONNECT_EPC_ERROR:
        statusCode = HTTP_SC_EPC_CONNECT_ERROR;
        res= "EPC CP Connect failure";
        break;
    case Exception::DISPATCH_NOTARGET:
        statusCode = HTTP_SC_NOT_FOUND;
        res= "404 not found";
        break;
    case Exception::DISPATCH_NOTYPE:
        statusCode = HTTP_SC_BAD_REQUEST;
        res= "BadRequest";
        break;
    default:
        statusCode = HTTP_SC_INTERNAL_SERVER_ERROR;
        res= "UnknownError";
        break;
    }
}
