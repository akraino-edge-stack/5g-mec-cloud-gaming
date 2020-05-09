#! /bin/sh
#SPDX-License-Identifier: Apache-2.0
#Copyright © 2019 Intel Corporation

setup_dir=${PWD}

set -e

curl -X PATCH -i "Content-Type: application/json" --data @./json/400_AF_NB_SUB_SUBID_PATCH001.json http://localhost:8181/af/v1/subscriptions/11112

exit 0

