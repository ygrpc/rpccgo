#define _POSIX_C_SOURCE 200809L

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "librpccgo_minimal_greeter.h"

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
  fprintf(stderr, "%s %.*s\n", prefix, (int)text_len, (const char *)text_ptr);
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
  const char *name = "minimal-c";

  assert_status_ok(rpccgo_native_greeterv1_Greeter_SayHello((uintptr_t)name, 9, 0, &message_ptr, &message_len, &message_ownership),
                   "native unary error:");
  assert_string_equals("native unary", (const char *)message_ptr, message_len,
                       "hello, minimal-c");
  printf("native unary: %.*s\n", (int)message_len, (const char *)message_ptr);
  if (rpccgo_release(message_ptr) != 0) {
    fail_with_message("release native unary output failed");
  }
}

static void run_output_error_demo(void) {
  const char *name = "minimal-c";
  int32_t err_id = rpccgo_native_greeterv1_Greeter_SayHello((uintptr_t)name, 9, 0, NULL, NULL, NULL);
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
  run_output_error_demo();
  return 0;
}
