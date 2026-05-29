#define _POSIX_C_SOURCE 200809L

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "librpccgo_connect_greeter.h"

static void fail_with_message(const char *message) {
  fprintf(stderr, "%s\n", message);
  exit(1);
}

static void print_error_and_exit(const char *prefix, int32_t err_id) {
  uintptr_t text_ptr = 0;
  int32_t text_len = 0;
  if (rpccgo_take_error_text(err_id, &text_ptr, &text_len) != 0 || text_ptr == 0) {
    fprintf(stderr, "%s <missing>\n", prefix);
    exit(1);
  }
  printf("%s %.*s\n", prefix, (int)text_len, (const char *)text_ptr);
  if (rpccgo_release(text_ptr) != 0) {
    fail_with_message("release error text failed");
  }
  exit(1);
}

static void assert_status_ok(int32_t status, const char *prefix) {
  if (status != 0) {
    print_error_and_exit(prefix, status);
  }
}

static void assert_string_equals(const char *label, const char *got, int32_t got_len,
                                 const char *want) {
  size_t want_len = strlen(want);
  if ((size_t)got_len != want_len || memcmp(got, want, want_len) != 0) {
    fprintf(stderr, "%s mismatch: got %.*s want %s\n", label, (int)got_len, got, want);
    exit(1);
  }
}

static void run_native_unary_demo(void) {
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;
  const char *name = "ffi";
  const char *city = "c";

  assert_status_ok(rpccgo_native_greeterv1_Greeter_SayHello(
                       (uintptr_t)name, 3, 0,
                       (uintptr_t)city, 1, 0,
                       &message_ptr, &message_len, &message_ownership),
                   "native unary error:");
  assert_string_equals("native unary", (const char *)message_ptr, message_len,
                       "hello ffi from c");
  printf("native unary: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgo_release(message_ptr) != 0) {
    fail_with_message("release native unary output failed");
  }
}

static void run_native_collect_demo(void) {
  int32_t handle = 0;
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;
  const char *city = "c";
  const char *name1 = "ada";
  const char *name2 = "grace";

  assert_status_ok(rpccgo_native_greeterv1_Greeter_Collect_start(&handle),
                   "native collect start error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Collect_send(
                       handle,
                       (uintptr_t)name1, 3, 0,
                       (uintptr_t)city, 1, 0),
                   "native collect send ada error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Collect_send(
                       handle,
                       (uintptr_t)name2, 5, 0,
                       (uintptr_t)city, 1, 0),
                   "native collect send grace error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Collect_finish(handle, &message_ptr, &message_len, &message_ownership),
                   "native collect finish error:");
  assert_string_equals("native collect", (const char *)message_ptr, message_len,
                       "collect:ada,grace");
  printf("native collect: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgo_release(message_ptr) != 0) {
    fail_with_message("release native collect output failed");
  }
}

static void read_native_stream_message(int32_t handle, const char *want,
                                       const char *error_prefix) {
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;

  assert_status_ok(rpccgo_native_greeterv1_Greeter_Broadcast_read(
                       handle, &message_ptr, &message_len, &message_ownership),
                   error_prefix);
  assert_string_equals("native stream", (const char *)message_ptr, message_len, want);
  printf("native broadcast: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgo_release(message_ptr) != 0) {
    fail_with_message("release native broadcast output failed");
  }
}

static void run_native_broadcast_demo(void) {
  int32_t handle = 0;
  const char *name = "stream";
  const char *city = "c";

  assert_status_ok(rpccgo_native_greeterv1_Greeter_Broadcast_start(
                       (uintptr_t)name, 6, 0,
                       (uintptr_t)city, 1, 0,
                       &handle),
                   "native broadcast start error:");
  read_native_stream_message(handle, "broadcast[0]:stream", "native broadcast read 0 error:");
  read_native_stream_message(handle, "broadcast[1]:stream", "native broadcast read 1 error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Broadcast_done(handle),
                   "native broadcast done error:");
}

static void run_native_chat_demo(void) {
  int32_t handle = 0;
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;
  const char *name = "bidi";
  const char *city = "c";

  assert_status_ok(rpccgo_native_greeterv1_Greeter_Chat_start(&handle),
                   "native chat start error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Chat_send(
                       handle,
                       (uintptr_t)name, 4, 0,
                       (uintptr_t)city, 1, 0),
                   "native chat send error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Chat_read(
                       handle, &message_ptr, &message_len, &message_ownership),
                   "native chat read error:");
  assert_string_equals("native chat", (const char *)message_ptr, message_len,
                       "chat:bidi");
  printf("native chat: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgo_release(message_ptr) != 0) {
    fail_with_message("release native chat output failed");
  }
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Chat_close_send(handle),
                   "native chat close send error:");
  assert_status_ok(rpccgo_native_greeterv1_Greeter_Chat_done(handle),
                   "native chat done error:");
}

static void run_output_error_demo(void) {
  const char *name = "ffi";
  const char *city = "c";
  int32_t err_id = rpccgo_native_greeterv1_Greeter_SayHello(
      (uintptr_t)name, 3, 0,
      (uintptr_t)city, 1, 0,
      NULL, NULL, NULL);
  if (err_id == 0) {
    fail_with_message("expected native output pointer error");
  }

  uintptr_t text_ptr = 0;
  int32_t text_len = 0;
  if (rpccgo_take_error_text(err_id, &text_ptr, &text_len) != 0 || text_ptr == 0) {
    fail_with_message("take output error text failed");
  }
  assert_string_equals("native output error", (const char *)text_ptr, text_len,
                       "rpccgo: native client output pointer is nil");
  printf("native output error: %.*s\n", (int)text_len, (const char *)text_ptr);
  if (rpccgo_release(text_ptr) != 0) {
    fail_with_message("release output error text failed");
  }
}

int main(void) {
  run_native_unary_demo();
  run_native_collect_demo();
  run_native_broadcast_demo();
  run_native_chat_demo();
  run_output_error_demo();
  return 0;
}
