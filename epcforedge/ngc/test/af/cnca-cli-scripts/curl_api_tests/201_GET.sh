#! /bin/sh
#SPDX-License-Identifier: Apache-2.0
#Copyright © 2019 Intel Corporation

setup_dir=${PWD}

set -e

curl http://localhost:8181/AF/v1/subscriptions/11112

exit 0

