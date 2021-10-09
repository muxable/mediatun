#ifndef GST_H
#define GST_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandleVP8Buffer(void *, int, unsigned long, unsigned long, void *);
extern void goHandleOpusBuffer(void *, int, unsigned long, unsigned long, void *);
extern void goHandleRtcpAppSinkBuffer(void *, int, void *);

void gstreamer_init(void);
void gstreamer_main_loop(void);
GstElement *gstreamer_start(char *, void *);
void gstreamer_stop(GstElement *);
void gstreamer_push_rtp(GstElement *, char *, void *, int);

#endif