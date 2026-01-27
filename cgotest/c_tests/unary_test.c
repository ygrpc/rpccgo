#define _POSIX_C_SOURCE 200809L

#include <assert.h>
#include <inttypes.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "libygrpc.h"
#include "proto_helpers.h"
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
        uint8_t* req_buf = NULL;
        int req_len = 0;
        assert(ygrpc_encode_string_field(1, (const uint8_t*)msg, (int)strlen(msg), &req_buf, &req_len) == 0);

        void* resp_ptr = NULL;
        GoInt resp_len = 0;
        void* resp_free = NULL;

        uint64_t err_id = Ygrpc_TestService_Ping(req_buf, req_len, &resp_ptr, &resp_len, &resp_free);
        free(req_buf);

        if (err_id != 0) {
            fprintf(stderr, "Ygrpc_TestService_Ping failed: %" PRIu64 "\n", err_id);
            return 1;
        }
        if (resp_ptr == NULL || resp_free == NULL) {
            fprintf(stderr, "expected resp_ptr and resp_free\n");
            return 1;
        }

        const uint8_t* out_msg_ptr = NULL;
        int out_msg_len = 0;
        assert(ygrpc_decode_string_field((const uint8_t*)resp_ptr, (int)resp_len, 1, &out_msg_ptr, &out_msg_len) == 0);
        ygrpc_expect_eq_str((const char*)out_msg_ptr, out_msg_len, "pong: hello");

        call_free_func((FreeFunc)resp_free, resp_ptr);
    }

    {
        g_free_called = 0;
        const char* msg = "hello";
        uint8_t* req_buf = NULL;
        int req_len = 0;
        assert(ygrpc_encode_string_field(1, (const uint8_t*)msg, (int)strlen(msg), &req_buf, &req_len) == 0);

        void* resp_ptr = NULL;
        GoInt resp_len = 0;
        void* resp_free = NULL;

        uint64_t err_id = Ygrpc_TestService_Ping_TakeReq(req_buf, req_len, (void*)counting_free, &resp_ptr, &resp_len, &resp_free);

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
