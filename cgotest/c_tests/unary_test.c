#define _POSIX_C_SOURCE 200809L

#include <assert.h>
#include <inttypes.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libygrpc.h"
#include "pb/unary.pb.h"
#include "test_helpers.h"

static int g_free_called = 0;
static void counting_free(void* p) {
    g_free_called++;
    free(p);
}

static void test_error_path(void) {
    uint8_t bad[1] = {0xFF};

    void* resp_ptr = NULL;
    GoInt resp_len = 0;
    void* resp_free = NULL;

    uint64_t err_id = Ygrpc_TestService_Ping(bad, 1, &resp_ptr, &resp_len, &resp_free);
    if (err_id == 0) {
        fprintf(stderr, "expected non-zero error id in invalid-protobuf test\n");
        abort();
    }

    void* emsg_ptr = NULL;
    GoInt emsg_len = 0;
    void* emsg_free = NULL;
    uint64_t rc = Ygrpc_GetErrorMsg(err_id, &emsg_ptr, &emsg_len, &emsg_free);
    if (rc != 0 || emsg_len <= 0 || emsg_ptr == NULL || emsg_free == NULL) {
        fprintf(stderr, "Ygrpc_GetErrorMsg failed: rc=%" PRIu64 " len=%lld ptr=%p free=%p\n", rc, (long long)emsg_len, emsg_ptr, emsg_free);
        abort();
    }

    call_free_func((FreeFunc)emsg_free, emsg_ptr);
}

int main(void) {
    uint64_t rc = Ygrpc_SetProtocol(YGRPC_PROTOCOL_UNSET);
    if (rc != 0)
    {
        fprintf(stderr, "Ygrpc_SetProtocol failed: %" PRIu64 "\n", rc);
        return 1;
    }

    {
        const char* msg = "hello";
        cgotest_PingRequest req = cgotest_PingRequest_init_zero;
        strncpy(req.msg, msg, sizeof(req.msg) - 1);

        uint8_t req_buf[cgotest_PingRequest_size];
        pb_ostream_t ostream = pb_ostream_from_buffer(req_buf, sizeof(req_buf));
        if (!pb_encode(&ostream, cgotest_PingRequest_fields, &req)) {
            fprintf(stderr, "pb_encode PingRequest failed: %s\n", PB_GET_ERROR(&ostream));
            return 1;
        }
        int req_len = (int)ostream.bytes_written;

        void* resp_ptr = NULL;
        GoInt resp_len = 0;
        void* resp_free = NULL;

        uint64_t err_id = Ygrpc_TestService_Ping(req_buf, req_len, &resp_ptr, &resp_len, &resp_free);

        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping failed: %" PRIu64 "\n", err_id);
            return 1;
        }
        if (resp_ptr == NULL || resp_free == NULL) {
            fprintf(stderr, "expected resp_ptr and resp_free\n");
            return 1;
        }

        cgotest_PingResponse resp = cgotest_PingResponse_init_zero;
        pb_istream_t istream = pb_istream_from_buffer((const pb_byte_t*)resp_ptr, (size_t)resp_len);
        if (!pb_decode(&istream, cgotest_PingResponse_fields, &resp)) {
            fprintf(stderr, "pb_decode PingResponse failed: %s\n", PB_GET_ERROR(&istream));
            return 1;
        }
        ygrpc_expect_eq_str(resp.msg, (int)strlen(resp.msg), "pong: hello");

        call_free_func((FreeFunc)resp_free, resp_ptr);
    }

    {
        g_free_called = 0;
        const char* msg = "hello";
        cgotest_PingRequest req = cgotest_PingRequest_init_zero;
        strncpy(req.msg, msg, sizeof(req.msg) - 1);

        uint8_t req_buf[cgotest_PingRequest_size];
        pb_ostream_t ostream = pb_ostream_from_buffer(req_buf, sizeof(req_buf));
        if (!pb_encode(&ostream, cgotest_PingRequest_fields, &req)) {
            fprintf(stderr, "pb_encode PingRequest failed: %s\n", PB_GET_ERROR(&ostream));
            return 1;
        }
        size_t req_len = ostream.bytes_written;
        uint8_t* req_heap = (uint8_t*)malloc(req_len);
        if (!req_heap) {
            fprintf(stderr, "malloc failed for request buffer\n");
            return 1;
        }
        memcpy(req_heap, req_buf, req_len);

        void* resp_ptr = NULL;
        GoInt resp_len = 0;
        void* resp_free = NULL;

        uint64_t err_id = Ygrpc_TestService_Ping_TakeReq(req_heap, (int)req_len, (void*)counting_free, &resp_ptr, &resp_len, &resp_free);

        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping_TakeReq failed: %" PRIu64 "\n", err_id);
            return 1;
        }
        if (g_free_called != 1) {
            fprintf(stderr, "expected req free called once, got %d\n", g_free_called);
            return 1;
        }

        call_free_func((FreeFunc)resp_free, resp_ptr);
    }

    {
        const char* msg = "hello";
        char* out_msg = NULL;
        int out_len = 0;
        FreeFunc out_free = NULL;

        uint64_t err_id = Ygrpc_TestService_Ping_Native((char*)msg, (int)strlen(msg), &out_msg, &out_len, &out_free);
        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping_Native failed: %" PRIu64 "\n", err_id);
            return 1;
        }
        if (out_msg == NULL || out_free == NULL) {
            fprintf(stderr, "expected native resp buffer/free\n");
            return 1;
        }
        ygrpc_expect_eq_str(out_msg, out_len, "pong: hello");
        out_free(out_msg);
    }

    test_error_path();

    printf("unary_test OK\n");
    return 0;
}
