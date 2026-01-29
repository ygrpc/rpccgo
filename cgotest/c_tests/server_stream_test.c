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
    char results[8][64];
} stream_state;

static void on_read_bytes(uint64_t call_id, void *resp_ptr, int resp_len, FreeFunc resp_free)
{
    stream_state *st = (stream_state *)(uintptr_t)call_id;

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
    stream_state *st = (stream_state *)(uintptr_t)call_id;
    st->done = 1;
    st->done_error_id = error_id;
}

static void on_read_native(uint64_t call_id, void *result_ptr, int result_len, FreeFunc result_free, int32_t sequence)
{
    (void)sequence;
    stream_state *st = (stream_state *)(uintptr_t)call_id;
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

    // Binary server-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        cgotest_StreamRequest req = cgotest_StreamRequest_init_zero;
        strncpy(req.data, "test", sizeof(req.data) - 1);
        req.sequence = 7;

        uint8_t req_buf[cgotest_StreamRequest_size];
        pb_ostream_t ostream = pb_ostream_from_buffer(req_buf, sizeof(req_buf));
        if (!pb_encode(&ostream, cgotest_StreamRequest_fields, &req)) {
            fprintf(stderr, "pb_encode StreamRequest failed: %s\n", PB_GET_ERROR(&ostream));
            return 1;
        }
        int req_len = (int)ostream.bytes_written;

        uint64_t call_id = (uint64_t)(uintptr_t)&st;
        uint64_t err_id = Ygrpc_StreamService_ServerStreamCall(req_buf, req_len, (void *)on_read_bytes, (void *)on_done, call_id);

        ygrpc_expect_err0_i64(err_id, "ServerStreamCall");
        YGRPC_ASSERTF(st.done && st.done_error_id == 0, "expected done with error=0, got done=%d err=%" PRIu64 "\n", st.done, st.done_error_id);
        YGRPC_ASSERTF(st.count == 3, "expected 3 responses, got %d\n", st.count);
        YGRPC_ASSERTF(strcmp(st.results[0], "test-a") == 0 && strcmp(st.results[1], "test-b") == 0 && strcmp(st.results[2], "test-c") == 0, 
            "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
    }

    // Native server-streaming
    {
        stream_state st;
        memset(&st, 0, sizeof(st));

        uint64_t err_id = Ygrpc_StreamService_ServerStreamCall_Native(
            (char *)"test", 4, (int32_t)7,
            (void *)on_read_native,
            (void *)on_done,
            (uint64_t)(uintptr_t)&st);

        ygrpc_expect_err0_i64(err_id, "ServerStreamCall_Native");
        YGRPC_ASSERTF(st.done && st.done_error_id == 0, "expected done with error=0, got done=%d err=%" PRIu64 "\n", st.done, st.done_error_id);
        YGRPC_ASSERTF(st.count == 3, "expected 3 responses, got %d\n", st.count);
        YGRPC_ASSERTF(strcmp(st.results[0], "test-a") == 0 && strcmp(st.results[1], "test-b") == 0 && strcmp(st.results[2], "test-c") == 0,
            "unexpected responses: %s, %s, %s\n", st.results[0], st.results[1], st.results[2]);
    }

    printf("server_stream_test OK\n");
    return 0;
}
