# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2019 Intel Corporation

#! /bin/sh

setup_dir=${PWD}

set -e

curl -X POST -i "Content-Type: application/json" --data @./json/SMF_NEF_NOTIF_01.json http://localhost:8091/3gpp-traffic-influence/v1/notification/upf

exit 0

