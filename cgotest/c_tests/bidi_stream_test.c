#define _POSIX_C_SOURCE 200809L

#include <assert.h>
#include <inttypes.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "test_helpers.h"
#include "libygrpc.h"
#include "pb/stream.pb.h"


typedef struct {
    int done;
    uint64_t done_error_id;
    int count;
    char results[16][64];
} stream_state;

// NOTE: callbacks may be invoked from a different OS thread than main.
// So we must not rely on thread-local storage here.
static stream_state *g_bidi_state = NULL;
static uint64_t g_bidi_call_id = 0;

static void on_read_bytes(uint64_t call_id, void *resp_ptr, int resp_len, FreeFunc resp_free)
{
    (void)call_id;
    stream_state *st = g_bidi_state;

    if (!g_bidi_state) {
        YGRPC_FAILF("callback before state init\n");
    }
    if (g_bidi_call_id == 0) {
        g_bidi_call_id = call_id;
    } else if (g_bidi_call_id != call_id) {
        YGRPC_FAILF("unexpected call_id: got=%llu want=%llu\n",
                (unsigned long long)call_id, (unsigned long long)g_bidi_call_id);
    }

    cgotest_StreamResponse resp = cgotest_StreamResponse_init_zero;
    pb_istream_t istream = pb_istream_from_buffer((const pb_byte_t*)resp_ptr, (size_t)resp_len);
    if (!pb_decode(&istream, cgotest_StreamResponse_fields, &resp)) {
        snprintf(st->results[st->count++], sizeof(st->results[0]), "<decode error>");
    } else {
        int n = (int)strlen(resp.result);
        if (n > (int)sizeof(st->results[0]) - 1) n = (int)sizeof(st->results[0]) - 1;
        memcpy(st->results[st->count], resp.result, (size_t)n);
        st->results[st->count][n] = 0;
        st->count++;
    }

    if (resp_free) resp_free(resp_ptr);
}

static void on_done(uint64_t call_id, uint64_t error_id)
{
    (void)call_id;
    stream_state *st = g_bidi_state;

    if (!g_bidi_state) {
        YGRPC_FAILF("callback before state init\n");
    }
    if (g_bidi_call_id == 0) {
        g_bidi_call_id = call_id;
    } else if (g_bidi_call_id != call_id) {
        YGRPC_FAILF("unexpected call_id: got=%llu want=%llu\n",
                (unsigned long long)call_id, (unsigned long long)g_bidi_call_id);
    }

    st->done = 1;
    st->done_error_id = error_id;
}

static void on_read_native(uint64_t call_id, void *result_ptr, int result_len, FreeFunc result_free, int32_t sequence)
{
    (void)sequence;
    (void)call_id;
    stream_state *st = g_bidi_state;
    int n = result_len;
    if (n > (int)sizeof(st->results[0]) - 1) n = (int)sizeof(st->results[0]) - 1;
    memcpy(st->results[st->count], result_ptr, (size_t)n);
    st->results[st->count][n] = 0;
    st->count++;
    if (result_free) result_free(result_ptr);
}

int main(void) {
    uint64_t rc = Ygrpc_SetProtocol(YGRPC_PROTOCOL_UNSET);
    ygrpc_expect_err0_i64(rc, "Ygrpc_SetProtocol");

    // Binary bidi-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        g_bidi_state = &st;
        g_bidi_call_id = 0;

        GoUint64 handle = 0;
        uint64_t err_id = Ygrpc_StreamService_BidiStreamCallStart((void *)on_read_bytes, (void *)on_done, &handle);
        YGRPC_ASSERTF(err_id == 0 && handle != 0, "BidiStart failed: err=%" PRIu64 " handle=%llu\n", err_id, (unsigned long long)handle);

        const char* msgs[] = {"X", "Y", "Z"};
        for (int i = 0; i < 3; i++) {
            cgotest_StreamRequest req = cgotest_StreamRequest_init_zero;
            strncpy(req.data, msgs[i], sizeof(req.data) - 1);
            req.sequence = (int32_t)i;

            uint8_t req_buf[cgotest_StreamRequest_size];
            pb_ostream_t ostream = pb_ostream_from_buffer(req_buf, sizeof(req_buf));
            if (!pb_encode(&ostream, cgotest_StreamRequest_fields, &req)) {
                fprintf(stderr, "pb_encode StreamRequest failed: %s\n", PB_GET_ERROR(&ostream));
                return 1;
            }
            int req_len = (int)ostream.bytes_written;

            err_id = Ygrpc_StreamService_BidiStreamCallSend(handle, req_buf, req_len);
            ygrpc_expect_err0_i64(err_id, "BidiSend");
        }

        err_id = Ygrpc_StreamService_BidiStreamCallCloseSend(handle);
        ygrpc_expect_err0_i64(err_id, "BidiCloseSend");

        ygrpc_wait_done_flag((const volatile int*)&st.done, 200, 10*1000*1000);
        YGRPC_ASSERTF(st.done && st.done_error_id == 0, "expected done with error=0, got done=%d err=%" PRIu64 "\n", st.done, st.done_error_id);
        YGRPC_ASSERTF(st.count == 3, "expected 3 responses, got %d\n", st.count);
        YGRPC_ASSERTF(strcmp(st.results[0], "echo:X") == 0 && strcmp(st.results[1], "echo:Y") == 0 && strcmp(st.results[2], "echo:Z") == 0,
            "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);

        g_bidi_state = NULL;
    }

    // Native bidi-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        g_bidi_state = &st;
        g_bidi_call_id = 0;

        uint64_t handle = 0;
        uint64_t err_id = Ygrpc_StreamService_BidiStreamCallStart_Native((void *)on_read_native, (void *)on_done, &handle);
        YGRPC_ASSERTF(err_id == 0 && handle != 0, "BidiStart_Native failed: err=%" PRIu64 " handle=%llu\n", err_id, (unsigned long long)handle);

        const char* msgs[] = {"X", "Y", "Z"};
        for (int i = 0; i < 3; i++) {
            err_id = Ygrpc_StreamService_BidiStreamCallSend_Native(handle, (char*)msgs[i], 1, (int32_t)i);
            ygrpc_expect_err0_i64(err_id, "BidiSend_Native");
        }

        err_id = Ygrpc_StreamService_BidiStreamCallCloseSend_Native(handle);
        ygrpc_expect_err0_i64(err_id, "BidiCloseSend_Native");

        ygrpc_wait_done_flag((const volatile int*)&st.done, 200, 10*1000*1000);
        YGRPC_ASSERTF(st.done && st.done_error_id == 0, "expected done with error=0, got done=%d err=%" PRIu64 "\n", st.done, st.done_error_id);
        YGRPC_ASSERTF(st.count == 3, "expected 3 responses, got %d\n", st.count);
        YGRPC_ASSERTF(strcmp(st.results[0], "echo:X") == 0 && strcmp(st.results[1], "echo:Y") == 0 && strcmp(st.results[2], "echo:Z") == 0,
            "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
    }

    printf("bidi_stream_test OK\n");
    return 0;
}
