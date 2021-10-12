#include "gst.h"

#include <gst/app/gstappsrc.h>

GMainLoop *main_loop = NULL;

void gstreamer_init(void)
{
    gst_init(NULL, NULL);

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
            goHandlePipelineBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
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
            goHandlePipelineRtcp(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

GstElement *gstreamer_run(void *data)
{
    GstElement *pipeline =
        gst_parse_launch(
            "rtpsession name=audiortpsession rtp-profile=avpf rtpsession name=videortpsession rtp-profile=avpf"
            "     appsrc name=rtpsrc time=live port=5002 caps='application/x-rtp,media=(string)audio,clock-rate=(int)48000,encoding-name=(string)OPUS,payload=(int)96' !"
            "         audiortpsession.recv_rtp_sink"
            "     audiortpsession.recv_rtp_src !"
            "         rtprtxreceive payload-type-map='application/x-rtp-pt-map,96=(uint)97' !"
            "         rtpssrcdemux name=audiortpssrcdemux ! rtpjitterbuffer do-lost=true do-retransmission=true !"
            "         rtpopusdepay ! decodebin ! audioconvert ! audioresample ! appsink name=appsink"
            "     audiortpsession.send_rtcp_src ! appsink name=rtcpsink sync=false async=false"
            "     appsrc name=rtcpsrc ! audiortpsession.recv_rtcp_sink",
            NULL);

    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    gst_bus_add_watch(bus, gstreamer_bus_call, NULL);
    gst_object_unref(bus);

    GstElement *appsink = gst_bin_get_by_name(GST_BIN(pipeline), "appsink");
    g_object_set(appsink, "emit-signals", TRUE, NULL);
    g_signal_connect(appsink, "new-sample", G_CALLBACK(gstreamer_pull_buffer), data);
    gst_object_unref(appsink);

    GstElement *rtcpsink = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpsink");
    g_object_set(rtcpsink, "emit-signals", TRUE, NULL);
    g_signal_connect(rtcpsink, "new-sample", G_CALLBACK(gstreamer_pull_rtcp), data);
    gst_object_unref(rtcpsink);

    gst_element_set_state(pipeline, GST_STATE_PLAYING);

    return pipeline;
}

void gstreamer_push_rtp(GstElement *pipeline, void *buffer, int len)
{
    GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), "rtpsrc");
    gpointer p = g_memdup2(buffer, len);
    GstBuffer *b = gst_buffer_new_wrapped(p, len);
    gst_app_src_push_buffer(GST_APP_SRC(src), b);
    gst_object_unref(src);
}

void gstreamer_push_rtcp(GstElement *pipeline, void *buffer, int len)
{
    GstElement *src = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpsrc");
    gpointer p = g_memdup2(buffer, len);
    GstBuffer *b = gst_buffer_new_wrapped(p, len);
    gst_app_src_push_buffer(GST_APP_SRC(src), b);
    gst_object_unref(src);
}
