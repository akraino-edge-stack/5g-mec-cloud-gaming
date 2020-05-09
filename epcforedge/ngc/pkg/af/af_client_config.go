// SPDX-License-Identifier: Apache-2.0
// Copyright © 2019-2020 Intel Corporation

package af

import (
	"net/http"
)

// CliConfig struct
type CliConfig struct {
	Protocol       string `json:"Protocol"`
	NEFHostname    string `json:"NEFHostname"`
	NEFPort        string `json:"NEFPort"`
	NEFBasePath    string `json:"NEFBasePath"`
	NEFPFDBasePath string `json:"NEFPFDBasePath"`
	UserAgent      string `json:"UserAgent"`
	NEFCliCertPath string `json:"NEFCliCertPath"`
	HTTPClient     *http.Client
	OAuth2Support  bool `json:"OAuth2Support"`
}

// NewConfiguration function initializes client configuration
func NewConfiguration(afCtx *Context) *CliConfig {

	cfg := &CliConfig{
		Protocol:       afCtx.cfg.CliCfg.Protocol,
		NEFPort:        afCtx.cfg.CliCfg.NEFPort,
		NEFHostname:    afCtx.cfg.CliCfg.NEFHostname,
		NEFBasePath:    afCtx.cfg.CliCfg.NEFBasePath,
		NEFPFDBasePath: afCtx.cfg.CliCfg.NEFPFDBasePath,
		UserAgent:      afCtx.cfg.CliCfg.UserAgent,
		NEFCliCertPath: afCtx.cfg.CliCfg.NEFCliCertPath,
		OAuth2Support:  afCtx.cfg.CliCfg.OAuth2Support,
	}

	return cfg
}
