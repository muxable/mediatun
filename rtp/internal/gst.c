#include "gst.h"

#include <gst/app/gstappsrc.h>
#include <gst/app/gstappsink.h>
#include <stdio.h>

GMainLoop *main_loop = NULL;

void gstreamer_init(void)
{
    gst_init(NULL, NULL);
}

void gstreamer_main_loop(void)
{
    main_loop = g_main_loop_new(NULL, FALSE);

    g_main_loop_run(main_loop);
}

static gboolean gstreamer_bus_call(GstBus *bus, GstMessage *msg, gpointer data)
{
    switch (GST_MESSAGE_TYPE(msg))
    {
    case GST_MESSAGE_EOS:
        g_print("End of stream\n");
        exit(1);
        break;

    case GST_MESSAGE_ERROR:
    {
        gchar *debug;
        GError *error;

        gst_message_parse_error(msg, &error, &debug);
        g_free(debug);

        g_printerr("Error: %s\n", error->message);
        g_error_free(error);
        exit(1);
    }
    default:
        break;
    }

    return TRUE;
}

static GstFlowReturn gstreamer_pull_buffer(GstElement *object, gpointer user_data)
{
    GstSample *sample = NULL;
    GstBuffer *buffer = NULL;
    gpointer copy = NULL;
    gsize copy_size = 0;

    g_signal_emit_by_name(object, "pull-sample", &sample);
    if (sample)
    {
        buffer = gst_sample_get_buffer(sample);
        if (buffer)
        {
            gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
            goHandleAppSinkBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

static GstFlowReturn gstreamer_pull_rtcp(GstElement *object, gpointer user_data)
{
    GstSample *sample = NULL;
    GstBuffer *buffer = NULL;
    gpointer copy = NULL;
    gsize copy_size = 0;

    g_signal_emit_by_name(object, "pull-sample", &sample);
    if (sample)
    {
        buffer = gst_sample_get_buffer(sample);
        if (buffer)
        {
            gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
            goHandleRtcpAppSinkBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

GstCaps *gstreamer_rtpjitterbuffer_request_pt_map(GstElement *buffer, guint pt, gpointer udata)
{
    if (pt == 96)
    {
        return gst_caps_new_simple("application/x-rtp", "clock-rate", G_TYPE_INT, 48000, "payload", G_TYPE_INT, pt, NULL);
    }
    else if (pt == 120)
    {
        return gst_caps_new_simple("application/x-rtp", "clock-rate", G_TYPE_INT, 90000, "payload", G_TYPE_INT, pt, NULL);
    }
}

GstElement *gstreamer_start(char *pipelineStr, void *data)
{
    GstElement *pipeline = gst_parse_launch(pipelineStr, NULL);

    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    gst_bus_add_watch(bus, gstreamer_bus_call, NULL);
    gst_object_unref(bus);

    GstElement *appsink = gst_bin_get_by_name(GST_BIN(pipeline), "bufferappsink");
    if (appsink != NULL) {
        g_object_set(appsink, "emit-signals", TRUE, NULL);
        g_signal_connect(appsink, "new-sample", G_CALLBACK(gstreamer_pull_buffer), data);
        gst_object_unref(appsink);
    }

    GstElement *rtcpappsink = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpappsink");
    if (rtcpappsink != NULL) {
        g_object_set(rtcpappsink, "emit-signals", TRUE, NULL);
        g_signal_connect(rtcpappsink, "new-sample", G_CALLBACK(gstreamer_pull_rtcp), data);
        gst_object_unref(rtcpappsink);
    }

    GstElement *rtpjitterbuffer = gst_bin_get_by_name(GST_BIN(pipeline), "rtpjitterbuffer");
    if (rtpjitterbuffer != NULL) {
        g_signal_connect(rtpjitterbuffer, "request-pt-map", G_CALLBACK(gstreamer_rtpjitterbuffer_request_pt_map), data);
        gst_object_unref(rtpjitterbuffer);
    }

    gst_element_set_state(pipeline, GST_STATE_PLAYING);

    return pipeline;
}

void gstreamer_stop(GstElement *pipeline)
{
    gst_element_set_state(pipeline, GST_STATE_NULL);
}

void gstreamer_push_rtp(GstElement *pipeline, void *buffer, int len)
{
    GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), "rtpappsrc");
    if (src != NULL)
    {
        gpointer p = g_memdup(buffer, len);
        GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
        gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
        gst_object_unref(src);
    }
}

void gstreamer_push_rtcp(GstElement *pipeline, void *buffer, int len)
{
    GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpappsrc");
    if (src != NULL)
    {
        gpointer p = g_memdup(buffer, len);
        GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
        gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
        gst_object_unref(src);
    }
}