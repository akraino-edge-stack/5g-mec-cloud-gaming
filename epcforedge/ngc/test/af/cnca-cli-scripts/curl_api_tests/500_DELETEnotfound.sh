#! /bin/sh
#SPDX-License-Identifier: Apache-2.0
#Copyright © 2019 Intel Corporation

setup_dir=${PWD}

set -e

curl -X DELETE http://localhost:8181/af/v1/subscriptions/1001

exit 0

