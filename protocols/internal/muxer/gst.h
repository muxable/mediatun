#ifndef GST_H
#define GST_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandlePipelineBuffer(void *, int, int, void *);
extern void goHandlePipelineRtcp(void *, int, int, void *);

void gstreamer_init(void);
GstElement *gstreamer_run(void *);
void gstreamer_push_rtcp(GstElement *, void *, int);
void gstreamer_push_rtp(GstElement *, void *, int);

#endif