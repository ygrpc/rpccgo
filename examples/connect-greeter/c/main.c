#define _POSIX_C_SOURCE 200809L

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include "librpccgo_connect_greeter.h"

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

static char demo_output[256];
static unsigned char proto_output[256];
static char collect_names[128];
static char broadcast_name[64];
static char chat_name[64];
static volatile int chat_ready = 0;
static volatile int chat_closed = 0;
static int32_t broadcast_index = 0;
static int32_t next_stream = 1000;
static int32_t native_free_count = 0;

static void sleep_briefly(void) {
  struct timespec ts;
  ts.tv_sec = 0;
  ts.tv_nsec = 1000000;
  nanosleep(&ts, NULL);
}

static int32_t demo_eof(void) {
  return rpccgoStoreErrorText("EOF", 3);
}

static void demo_free(void *ptr) {
  native_free_count++;
  free(ptr);
}

static void copy_field(char *dst, size_t dst_len, const char *src, int32_t src_len) {
  if (dst_len == 0) {
    return;
  }
  size_t n = src_len < 0 ? 0 : (size_t)src_len;
  if (n >= dst_len) {
    n = dst_len - 1;
  }
  if (n > 0) {
    memcpy(dst, src, n);
  }
  dst[n] = '\0';
}

static int32_t store_native_output(const char *text, uintptr_t *out_ptr, int32_t *out_len,
                                   int32_t *out_ownership) {
  size_t len = strlen(text);
  char *output = (char *)malloc(len == 0 ? 1 : len);
  if (output == NULL) {
    return rpccgoStoreErrorText("native output allocation failed", 31);
  }
  if (len > 0) {
    memcpy(output, text, len);
  }
  *out_ptr = (uintptr_t)output;
  *out_len = (int32_t)len;
  *out_ownership = 1;
  return 0;
}

static int32_t write_proto_string(unsigned char *dst, size_t dst_len, const char *text,
                                  int32_t *out_len) {
  size_t text_len = strlen(text);
  if (text_len > 127 || text_len + 2 > dst_len) {
    return 99992;
  }
  dst[0] = 0x0a;
  dst[1] = (unsigned char)text_len;
  memcpy(dst + 2, text, text_len);
  *out_len = (int32_t)(text_len + 2);
  return 0;
}

static int32_t store_message_output(const char *text, uintptr_t *out_ptr, int32_t *out_len) {
  int32_t err = write_proto_string(proto_output, sizeof(proto_output), text, out_len);
  if (err != 0) {
    return err;
  }
  *out_ptr = (uintptr_t)proto_output;
  return 0;
}

static int read_varint(const unsigned char *data, int32_t len, int32_t *index, int32_t *value) {
  int32_t shift = 0;
  int32_t result = 0;
  while (*index < len && shift <= 21) {
    unsigned char b = data[*index];
    (*index)++;
    result |= (int32_t)(b & 0x7f) << shift;
    if ((b & 0x80) == 0) {
      *value = result;
      return 1;
    }
    shift += 7;
  }
  return 0;
}

static void parse_message_request(uintptr_t request_ptr, int32_t request_len, char *name,
                                  size_t name_len, char *city, size_t city_len) {
  const unsigned char *data = (const unsigned char *)request_ptr;
  int32_t index = 0;
  name[0] = '\0';
  city[0] = '\0';
  while (index < request_len) {
    unsigned char tag = data[index++];
    int32_t length = 0;
    if (!read_varint(data, request_len, &index, &length) || length < 0 || index + length > request_len) {
      return;
    }
    if (tag == 0x0a) {
      copy_field(name, name_len, (const char *)(data + index), length);
    } else if (tag == 0x12) {
      copy_field(city, city_len, (const char *)(data + index), length);
    }
    index += length;
  }
}

static int32_t cgo_native_unary(uintptr_t name_ptr, int32_t name_len, int32_t name_ownership,
                                uintptr_t city_ptr, int32_t city_len, int32_t city_ownership,
                                uintptr_t *out_message_ptr, int32_t *out_message_len,
                                int32_t *out_message_ownership) {
  char name[64];
  char city[64];
  (void)name_ownership;
  (void)city_ownership;
  copy_field(name, sizeof(name), (const char *)name_ptr, name_len);
  copy_field(city, sizeof(city), (const char *)city_ptr, city_len);
  snprintf(demo_output, sizeof(demo_output), "hello %s from %s", name, city);
  return store_native_output(demo_output, out_message_ptr, out_message_len, out_message_ownership);
}

static int32_t cgo_native_collect_start(int32_t *stream) {
  *stream = next_stream++;
  collect_names[0] = '\0';
  return 0;
}

static int32_t cgo_native_collect_send(int32_t stream, uintptr_t name_ptr, int32_t name_len,
                                       int32_t name_ownership, uintptr_t city_ptr,
                                       int32_t city_len, int32_t city_ownership) {
  char name[64];
  (void)stream;
  (void)name_ownership;
  (void)city_ptr;
  (void)city_len;
  (void)city_ownership;
  copy_field(name, sizeof(name), (const char *)name_ptr, name_len);
  if (collect_names[0] != '\0') {
    strncat(collect_names, ",", sizeof(collect_names) - strlen(collect_names) - 1);
  }
  strncat(collect_names, name, sizeof(collect_names) - strlen(collect_names) - 1);
  return 0;
}

static int32_t cgo_native_collect_finish(int32_t stream, uintptr_t *out_message_ptr,
                                         int32_t *out_message_len,
                                         int32_t *out_message_ownership) {
  (void)stream;
  snprintf(demo_output, sizeof(demo_output), "collect:%s", collect_names);
  return store_native_output(demo_output, out_message_ptr, out_message_len, out_message_ownership);
}

static int32_t cgo_native_stream_cancel(int32_t stream) {
  (void)stream;
  return 0;
}

static int32_t cgo_native_broadcast_start(uintptr_t name_ptr, int32_t name_len,
                                          int32_t name_ownership, uintptr_t city_ptr,
                                          int32_t city_len, int32_t city_ownership,
                                          int32_t *stream) {
  (void)name_ownership;
  (void)city_ptr;
  (void)city_len;
  (void)city_ownership;
  *stream = next_stream++;
  broadcast_index = 0;
  copy_field(broadcast_name, sizeof(broadcast_name), (const char *)name_ptr, name_len);
  return 0;
}

static int32_t cgo_native_broadcast_recv(int32_t stream, uintptr_t *out_message_ptr,
                                         int32_t *out_message_len,
                                         int32_t *out_message_ownership) {
  (void)stream;
  if (broadcast_index >= 2) {
    return demo_eof();
  }
  snprintf(demo_output, sizeof(demo_output), "broadcast[%d]:%s", broadcast_index, broadcast_name);
  broadcast_index++;
  return store_native_output(demo_output, out_message_ptr, out_message_len, out_message_ownership);
}

static int32_t cgo_native_stream_finish(int32_t stream) {
  (void)stream;
  return 0;
}

static int32_t cgo_native_chat_start(int32_t *stream) {
  *stream = next_stream++;
  chat_name[0] = '\0';
  chat_ready = 0;
  chat_closed = 0;
  return 0;
}

static int32_t cgo_native_chat_send(int32_t stream, uintptr_t name_ptr, int32_t name_len,
                                    int32_t name_ownership, uintptr_t city_ptr,
                                    int32_t city_len, int32_t city_ownership) {
  (void)stream;
  (void)name_ownership;
  (void)city_ptr;
  (void)city_len;
  (void)city_ownership;
  copy_field(chat_name, sizeof(chat_name), (const char *)name_ptr, name_len);
  chat_ready = 1;
  return 0;
}

static int32_t cgo_native_chat_recv(int32_t stream, uintptr_t *out_message_ptr,
                                    int32_t *out_message_len, int32_t *out_message_ownership) {
  (void)stream;
  while (!chat_ready && !chat_closed) {
    sleep_briefly();
  }
  if (!chat_ready) {
    return demo_eof();
  }
  snprintf(demo_output, sizeof(demo_output), "chat:%s", chat_name);
  chat_name[0] = '\0';
  chat_ready = 0;
  return store_native_output(demo_output, out_message_ptr, out_message_len, out_message_ownership);
}

static int32_t cgo_native_chat_close_send(int32_t stream) {
  (void)stream;
  chat_closed = 1;
  return 0;
}

static int32_t cgo_message_unary(uintptr_t request_ptr, int32_t request_len,
                                 uintptr_t *response_ptr, int32_t *response_len) {
  char name[64];
  char city[64];
  parse_message_request(request_ptr, request_len, name, sizeof(name), city, sizeof(city));
  snprintf(demo_output, sizeof(demo_output), "hello %s from %s", name, city);
  return store_message_output(demo_output, response_ptr, response_len);
}

static int32_t cgo_message_collect_start(int32_t *stream) {
  return cgo_native_collect_start(stream);
}

static int32_t cgo_message_collect_send(int32_t stream, uintptr_t request_ptr, int32_t request_len) {
  char name[64];
  char city[64];
  (void)stream;
  parse_message_request(request_ptr, request_len, name, sizeof(name), city, sizeof(city));
  if (collect_names[0] != '\0') {
    strncat(collect_names, ",", sizeof(collect_names) - strlen(collect_names) - 1);
  }
  strncat(collect_names, name, sizeof(collect_names) - strlen(collect_names) - 1);
  return 0;
}

static int32_t cgo_message_collect_finish(int32_t stream, uintptr_t *response_ptr,
                                          int32_t *response_len) {
  (void)stream;
  snprintf(demo_output, sizeof(demo_output), "collect:%s", collect_names);
  return store_message_output(demo_output, response_ptr, response_len);
}

static int32_t cgo_message_stream_cancel(int32_t stream) {
  (void)stream;
  return 0;
}

static int32_t cgo_message_broadcast_start(uintptr_t request_ptr, int32_t request_len,
                                           int32_t *stream) {
  char city[64];
  *stream = next_stream++;
  broadcast_index = 0;
  parse_message_request(request_ptr, request_len, broadcast_name, sizeof(broadcast_name), city, sizeof(city));
  return 0;
}

static int32_t cgo_message_broadcast_recv(int32_t stream, uintptr_t *response_ptr,
                                          int32_t *response_len) {
  (void)stream;
  if (broadcast_index >= 2) {
    return demo_eof();
  }
  snprintf(demo_output, sizeof(demo_output), "broadcast[%d]:%s", broadcast_index, broadcast_name);
  broadcast_index++;
  return store_message_output(demo_output, response_ptr, response_len);
}

static int32_t cgo_message_stream_finish(int32_t stream) {
  (void)stream;
  return 0;
}

static int32_t cgo_message_chat_start(int32_t *stream) {
  return cgo_native_chat_start(stream);
}

static int32_t cgo_message_chat_send(int32_t stream, uintptr_t request_ptr, int32_t request_len) {
  char city[64];
  (void)stream;
  parse_message_request(request_ptr, request_len, chat_name, sizeof(chat_name), city, sizeof(city));
  chat_ready = 1;
  return 0;
}

static int32_t cgo_message_chat_recv(int32_t stream, uintptr_t *response_ptr, int32_t *response_len) {
  (void)stream;
  while (!chat_ready && !chat_closed) {
    sleep_briefly();
  }
  if (!chat_ready) {
    return demo_eof();
  }
  snprintf(demo_output, sizeof(demo_output), "chat:%s", chat_name);
  chat_name[0] = '\0';
  chat_ready = 0;
  return store_message_output(demo_output, response_ptr, response_len);
}

static int32_t cgo_message_chat_close_send(int32_t stream) {
  (void)stream;
  chat_closed = 1;
  return 0;
}

static void register_cgo_message_server(void) {
  assert_status_ok(rpccgoMsgGreeterv1GreeterRegister(
                       cgo_message_unary,
                       cgo_message_collect_start, cgo_message_collect_send,
                       cgo_message_collect_finish, cgo_message_stream_cancel,
                       cgo_message_broadcast_start, cgo_message_broadcast_recv,
                       cgo_message_stream_finish, cgo_message_stream_cancel,
                       cgo_message_chat_start, cgo_message_chat_send, cgo_message_chat_recv,
                       cgo_message_chat_close_send, cgo_message_stream_finish,
                       cgo_message_stream_cancel),
                   "register cgo message server error:");
}

static void register_cgo_native_server(void) {
  native_free_count = 0;
  assert_status_ok(rpccgoNativeGreeterv1GreeterRegister(
                       cgo_native_unary,
                       cgo_native_collect_start, cgo_native_collect_send,
                       cgo_native_collect_finish, cgo_native_stream_cancel,
                       cgo_native_broadcast_start, cgo_native_broadcast_recv,
                       cgo_native_stream_finish, cgo_native_stream_cancel,
                       cgo_native_chat_start, cgo_native_chat_send, cgo_native_chat_recv,
                       cgo_native_chat_close_send, cgo_native_stream_finish,
                       cgo_native_stream_cancel),
                   "register cgo native server error:");
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
  fflush(stdout);
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
  fflush(stdout);
  if (rpccgoRelease(message_ptr) != 0) {
    fail_with_message("release native collect output failed");
  }
}

static void read_native_stream_message(int32_t handle, const char *want,
                                       const char *error_prefix) {
  uintptr_t message_ptr = 0;
  int32_t message_len = 0;
  int32_t message_ownership = 0;

  assert_status_ok(rpccgoNativeGreeterv1GreeterBroadcastRecv(
                       handle, &message_ptr, &message_len, &message_ownership),
                   error_prefix);
  assert_string_equals("native stream", (const char *)message_ptr, message_len, want);
  printf("native broadcast: %.*s\n", (int)message_len, (const char *)message_ptr);
  fflush(stdout);
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
  fflush(stdout);
  assert_status_ok(rpccgoNativeGreeterv1GreeterChatRecv(
                       handle, &message_ptr, &message_len, &message_ownership),
                   "native chat read error:");
  assert_string_equals("native chat", (const char *)message_ptr, message_len, want);
  printf("native chat server->c: %.*s\n", (int)message_len, (const char *)message_ptr);
  fflush(stdout);
  if (rpccgoRelease(message_ptr) != 0) {
    fail_with_message("release native chat output failed");
  }
}

static void run_registered_server_demo(const char *route) {
  if (route != NULL && route[0] != '\0') {
    printf("route: %s\n", route);
    fflush(stdout);
  }
  run_native_unary_demo();
  run_native_collect_demo();
  run_native_broadcast_demo();
  run_native_chat_demo();
}

int main(int argc, char **argv) {
  const char *kind = arg_value(argc, argv, "--server");
  assert_status_ok(rpccgoRegisterFree(demo_free), "register c free callback error:");
  verify_shared_error_exports();
  if (kind != NULL && strcmp(kind, "cgo_message") == 0) {
    register_cgo_message_server();
  } else if (kind != NULL && strcmp(kind, "cgo_native") == 0) {
    register_cgo_native_server();
  }
  run_registered_server_demo(arg_value(argc, argv, "--route"));
  if (kind != NULL && strcmp(kind, "cgo_native") == 0 && native_free_count != 6) {
    fail_with_message("native free callback count mismatch");
  }
  return 0;
}
