#!/bin/bash

GST_DEBUG=3 /pull -source $1 >> /var/log/pull.log 2>> /var/log/pull.err