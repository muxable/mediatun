#ifndef GST_H
#define GST_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandleAppSinkBuffer(void *, int, int, void *);
extern void goHandleRtcpAppSinkBuffer(void *, int, int, void *);

void gstreamer_init(void);
void gstreamer_main_loop(void);
GstElement *gstreamer_start(char *, void *);
void gstreamer_stop(GstElement *);
void gstreamer_push_rtp(GstElement *, void *, int);
void gstreamer_push_rtcp(GstElement *, void *, int);

#endif