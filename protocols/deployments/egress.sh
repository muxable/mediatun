#!/bin/bash

GST_DEBUG=3 /egress -id $1 -destination rtmp://localhost/live/$1 >> /var/log/egress.log 2>> /var/log/egress.err