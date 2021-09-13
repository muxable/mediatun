#!/bin/bash

GST_DEBUG=3 /egress -source $1 >> /var/log/egress.log 2>> /var/log/egress.err