#ifndef GST_H
#define GST_H

#include <glib.h>
#include <gst/gst.h>
#include <stdint.h>
#include <stdlib.h>

extern void goHandleVP8AppSinkBuffer(void *, int, int, void *);
extern void goHandleOpusAppSinkBuffer(void *, int, int, void *);

void gstreamer_init(void);
void gstreamer_main_loop(void);
GstElement *gstreamer_start(char *, void *);
void gstreamer_stop(GstElement *);
void gstreamer_push(GstElement *, void *, int);

#endif