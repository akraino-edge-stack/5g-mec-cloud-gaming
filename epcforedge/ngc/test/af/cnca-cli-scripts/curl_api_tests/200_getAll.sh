#! /bin/sh
#SPDX-License-Identifier: Apache-2.0
#Copyright © 2019 Intel Corporation

setup_dir=${PWD}

set -e

curl http://localhost:8181/af/v1/subscriptions

exit 0
