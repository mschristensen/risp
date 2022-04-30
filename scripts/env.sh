#!/bin/bash

if [ -f "$1" ]; then
    export $(egrep -v '^#' $1 | xargs)
fi
