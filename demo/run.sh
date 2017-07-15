#!/bin/bash

cd `dirname $0`
source common.sh
mkdir -p $testdir
../bin/memdbfs $testdir
