#define _POSIX_C_SOURCE 200809L

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "librpccgo_grpc_greeter.h"

static void fail_with_message(const char *message) {
  fprintf(stderr, "%s\n", message);
  exit(1);
}

static void print_error_and_exit(const char *prefix, int32_t err_id) {
  uintptr_t text_ptr = 0;
  int32_t text_len = 0;
  if (rpccgoTakeErrorText(err_id, &text_ptr, &text_len) != 0 || text_ptr == 0) {
    fprintf(stderr, "%s <missing>\n", prefix);
    exit(1);
  }
  printf("%s %.*s\n", prefix, (int)text_len, (const char *)text_ptr);
  if (rpccgoRelease(text_ptr) != 0) {
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

static void verify_shared_error_exports(void) {
  const char *want = "shared error";
  int32_t err_id = rpccgoStoreErrorText((char *)want, (int32_t)strlen(want));
  uintptr_t text_ptr = 0;
  int32_t text_len = 0;
  if (err_id == 0 || rpccgoTakeErrorText(err_id, &text_ptr, &text_len) != 0 ||
      text_ptr == 0) {
    fail_with_message("shared error text roundtrip failed");
  }
  assert_string_equals("shared error text", (const char *)text_ptr, text_len, want);
  if (rpccgoRelease(text_ptr) != 0) {
    fail_with_message("release shared error text failed");
  }
}

static const char *arg_value(int argc, char **argv, const char *name) {
  size_t name_len = strlen(name);
  for (int i = 1; i < argc; i++) {
    if (strcmp(argv[i], name) == 0 && i + 1 < argc) {
      return argv[i + 1];
    }
    if (strncmp(argv[i], name, name_len) == 0 && argv[i][name_len] == '=') {
      return argv[i] + name_len + 1;
    }
  }
  return NULL;
}

static void run_native_unary_demo(void) {
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;
  const char *name = "ffi";
  const char *city = "c";

  assert_status_ok(rpccgoNativeGreeterv1GreeterSayHello(
                       (uintptr_t)name, 3, 0,
                       (uintptr_t)city, 1, 0,
                       &message_ptr, &message_len, &message_ownership),
                   "native unary error:");
  assert_string_equals("native unary", (const char *)message_ptr, message_len,
                       "hello ffi from c");
  printf("native unary: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgoRelease(message_ptr) != 0) {
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

  assert_status_ok(rpccgoNativeGreeterv1GreeterCollectStart(&handle),
                   "native collect start error:");
  assert_status_ok(rpccgoNativeGreeterv1GreeterCollectSend(
                       handle,
                       (uintptr_t)name1, 3, 0,
                       (uintptr_t)city, 1, 0),
                   "native collect send ada error:");
  assert_status_ok(rpccgoNativeGreeterv1GreeterCollectSend(
                       handle,
                       (uintptr_t)name2, 5, 0,
                       (uintptr_t)city, 1, 0),
                   "native collect send grace error:");
  assert_status_ok(rpccgoNativeGreeterv1GreeterCollectFinish(handle, &message_ptr, &message_len, &message_ownership),
                   "native collect finish error:");
  assert_string_equals("native collect", (const char *)message_ptr, message_len,
                       "collect:ada,grace");
  printf("native collect: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgoRelease(message_ptr) != 0) {
    fail_with_message("release native collect output failed");
  }
}

static void read_native_stream_message(int32_t handle, const char *want,
                                       const char *error_prefix) {
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;

  assert_status_ok(rpccgoNativeGreeterv1GreeterBroadcastRead(
                       handle, &message_ptr, &message_len, &message_ownership),
                   error_prefix);
  assert_string_equals("native stream", (const char *)message_ptr, message_len, want);
  printf("native broadcast: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgoRelease(message_ptr) != 0) {
    fail_with_message("release native broadcast output failed");
  }
}

static void run_native_broadcast_demo(void) {
  int32_t handle = 0;
  const char *name = "stream";
  const char *city = "c";

  assert_status_ok(rpccgoNativeGreeterv1GreeterBroadcastStart(
                       (uintptr_t)name, 6, 0,
                       (uintptr_t)city, 1, 0,
                       &handle),
                   "native broadcast start error:");
  read_native_stream_message(handle, "broadcast[0]:stream", "native broadcast read 0 error:");
  read_native_stream_message(handle, "broadcast[1]:stream", "native broadcast read 1 error:");
  assert_status_ok(rpccgoNativeGreeterv1GreeterBroadcastFinish(handle),
                   "native broadcast finish error:");
}

static void send_native_chat_message(int32_t handle, const char *name, const char *city,
                                     const char *want);

static void run_native_chat_demo(void) {
  int32_t handle = 0;
  const char *city = "c";

  assert_status_ok(rpccgoNativeGreeterv1GreeterChatStart(&handle),
                   "native chat start error:");
  send_native_chat_message(handle, "ada", city, "chat:ada");
  send_native_chat_message(handle, "grace", city, "chat:grace");
  assert_status_ok(rpccgoNativeGreeterv1GreeterChatCloseSend(handle),
                   "native chat close send error:");
  assert_status_ok(rpccgoNativeGreeterv1GreeterChatFinish(handle),
                   "native chat finish error:");
}

static void send_native_chat_message(int32_t handle, const char *name, const char *city,
                                     const char *want) {
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;

  assert_status_ok(rpccgoNativeGreeterv1GreeterChatSend(
                       handle,
                       (uintptr_t)name, (int32_t)strlen(name), 0,
                       (uintptr_t)city, (int32_t)strlen(city), 0),
                   "native chat send error:");
  printf("native chat c->server: %s\n", name);
  assert_status_ok(rpccgoNativeGreeterv1GreeterChatRead(
                       handle, &message_ptr, &message_len, &message_ownership),
                   "native chat read error:");
  assert_string_equals("native chat", (const char *)message_ptr, message_len, want);
  printf("native chat server->c: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgoRelease(message_ptr) != 0) {
    fail_with_message("release native chat output failed");
  }
}

int main(int argc, char **argv) {
  const char *route = arg_value(argc, argv, "--route");
  assert_status_ok(rpccgoRegisterFree(free), "register c free callback error:");
  verify_shared_error_exports();
  if (route != NULL && route[0] != '\0') {
    printf("route: %s\n", route);
  }
  run_native_unary_demo();
  run_native_collect_demo();
  run_native_broadcast_demo();
  run_native_chat_demo();
  return 0;
}
