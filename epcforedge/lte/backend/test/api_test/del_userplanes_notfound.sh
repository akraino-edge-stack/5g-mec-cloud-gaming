#! /bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2019 Intel Corporation
#

setup_dir=${PWD}

set -e

curl -v -X DELETE http://epc.oam:8080/userplanes/666
