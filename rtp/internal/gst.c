#include "gst.h"

#include <gst/app/gstappsrc.h>
#include <gst/app/gstappsink.h>
#include <stdio.h>

typedef struct _SessionData
{
    GstElement *pipeline;
    gpointer userdata;
} SessionData;

typedef struct _BufferData
{
    guint64 ssrc;
    gpointer userdata;
} BufferData;

void gstreamer_init(void)
{
    gst_init(NULL, NULL);
}

void gstreamer_main_loop(void)
{
    GMainLoop *main_loop = g_main_loop_new(NULL, FALSE);

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

static GstFlowReturn gstreamer_pull_vp8_buffer(GstElement *object, gpointer user_data)
{
    GstSample *sample = gst_app_sink_pull_sample(GST_APP_SINK(object));
    if (sample)
    {
        GstBuffer *buffer = gst_sample_get_buffer(sample);
        if (buffer)
        {
            gpointer copy = NULL;
            gsize copy_size = 0;
            gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
            goHandleVP8Buffer(copy, copy_size, GST_BUFFER_TIMESTAMP(buffer), (void*) user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

static GstFlowReturn gstreamer_pull_opus_buffer(GstElement *object, gpointer user_data)
{
    GstSample *sample = gst_app_sink_pull_sample(GST_APP_SINK(object));
    if (sample)
    {
        GstBuffer *buffer = gst_sample_get_buffer(sample);
        if (buffer)
        {
            gpointer copy = NULL;
            gsize copy_size = 0;
            gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
            goHandleOpusBuffer(copy, copy_size, GST_BUFFER_TIMESTAMP(buffer), (void *) user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

static GstFlowReturn gstreamer_pull_rtcp(GstElement *object, gpointer user_data)
{
    GstSample *sample = gst_app_sink_pull_sample(GST_APP_SINK(object));
    if (sample)
    {
        GstBuffer *buffer = gst_sample_get_buffer(sample);
        if (buffer)
        {
            gpointer copy = NULL;
            gsize copy_size = 0;
            gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
            goHandleRtcpAppSinkBuffer(copy, copy_size, (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

static GstCaps *gstreamer_request_pt_map(GstElement *rtpbin, guint session, guint pt, gpointer user_data)
{
    if (pt == 96)
    {
        return gst_caps_new_simple("application/x-rtp",
                                   "payload", G_TYPE_INT, 96,
                                   "media", G_TYPE_STRING, "video",
                                   "clock-rate", G_TYPE_INT, 90000,
                                   "encoding-name", G_TYPE_STRING, "VP8",
                                   "extmap-5", G_TYPE_STRING, "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
                                   NULL);
    }
    else if (pt == 111)
    {

        return gst_caps_new_simple("application/x-rtp",
                                   "payload", G_TYPE_INT, 111,
                                   "media", G_TYPE_STRING, "audio",
                                   "clock-rate", G_TYPE_INT, 48000,
                                   "encoding-name", G_TYPE_STRING, "OPUS",
                                   NULL);
    }
    else
    {
        return NULL;
    }
}

static GstElement *gstreamer_request_aux_receiver(GstElement *rtpbin, guint sessid, gpointer user_data)
{
    GstElement *rtx, *bin;
    GstPad *pad;
    gchar *name;
    GstStructure *pt_map;

    bin = gst_bin_new(NULL);
    rtx = gst_element_factory_make("rtprtxreceive", NULL);
    pt_map = gst_structure_new(
        "application/x-rtp-pt-map",
        "96", G_TYPE_UINT, 97,
        "111", G_TYPE_UINT, 112,
        NULL);
    g_object_set(rtx, "payload-type-map", pt_map, NULL);
    gst_structure_free(pt_map);
    gst_bin_add(GST_BIN(bin), rtx);

    pad = gst_element_get_static_pad(rtx, "src");
    name = g_strdup_printf("src_%u", sessid);
    gst_element_add_pad(bin, gst_ghost_pad_new(name, pad));
    g_free(name);
    gst_object_unref(pad);

    pad = gst_element_get_static_pad(rtx, "sink");
    name = g_strdup_printf("sink_%u", sessid);
    gst_element_add_pad(bin, gst_ghost_pad_new(name, pad));
    g_free(name);
    gst_object_unref(pad);

    return bin;
}

GstElement *gstreamer_start(char *pipelineStr, void *data)
{
    GstElement *pipeline = gst_parse_launch(pipelineStr, NULL);

    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    gst_bus_add_watch(bus, gstreamer_bus_call, NULL);
    gst_object_unref(bus);

    GstElement *rtcpappsink = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpappsink");
    if (rtcpappsink != NULL)
    {
        g_object_set(rtcpappsink, "emit-signals", TRUE, NULL);
        g_signal_connect(rtcpappsink, "new-sample", G_CALLBACK(gstreamer_pull_rtcp), data);
        gst_object_unref(rtcpappsink);
    }

    GstElement *vp8appsink = gst_bin_get_by_name(GST_BIN(pipeline), "vp8appsink");
    if (vp8appsink != NULL)
    {
        g_object_set(vp8appsink, "emit-signals", TRUE, NULL);
        g_signal_connect(vp8appsink, "new-sample", G_CALLBACK(gstreamer_pull_vp8_buffer), data);
        gst_object_unref(vp8appsink);
    }
    
    GstElement *opusappsink = gst_bin_get_by_name(GST_BIN(pipeline), "opusappsink");
    if (opusappsink != NULL)
    {
        g_object_set(opusappsink, "emit-signals", TRUE, NULL);
        g_signal_connect(opusappsink, "new-sample", G_CALLBACK(gstreamer_pull_opus_buffer), data);
        gst_object_unref(opusappsink);
    }

    gst_element_set_state(pipeline, GST_STATE_PLAYING);

    return pipeline;
}

void gstreamer_stop(GstElement *pipeline)
{
    gst_element_set_state(pipeline, GST_STATE_NULL);
}

void gstreamer_push_rtp(GstElement *pipeline, char *name, void *buffer, int len)
{
    GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), name);
    if (src != NULL)
    {
        gpointer p = g_memdup(buffer, len);
        GstBuffer *buffer = gst_buffer_new_wrapped(p, len);
        gst_app_src_push_buffer(GST_APP_SRC(src), buffer);
        gst_object_unref(src);
    }
}