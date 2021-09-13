#!/bin/bash

GST_DEBUG=3 /ingress -id $1 -source rtmp://localhost/live/$1 >> /var/log/ingress.log 2>> /var/log/ingress.err