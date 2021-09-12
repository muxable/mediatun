#!/bin/bash

GST_DEBUG=3 /push -id $1 -source rtmp://localhost/live/$1 >> /var/log/push.log 2>> /var/log/push.err