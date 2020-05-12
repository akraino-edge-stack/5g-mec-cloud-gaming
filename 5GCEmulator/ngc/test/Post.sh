#! /bin/sh
#SPDX-License-Identifier: Apache-2.0
#Copyright © 2019 Intel Corporation

setup_dir=${PWD}

set -e

curl -X POST -i "Content-Type: application/json" --data @./json/100_AF_NB_SUB_POST001.json http://localhost:8050/af/v1/subscriptions

exit 0

