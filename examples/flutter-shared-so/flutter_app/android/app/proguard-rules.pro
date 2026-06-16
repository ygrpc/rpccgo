# protobuf-javalite generated messages pass field names such as "caller_" to
# GeneratedMessageLite.newMessageInfo. R8 may shrink/obfuscate those private
# fields in release builds, which makes protobuf parsing fail at runtime.
-keepclassmembers class examples.flutter.sharedso.v1.** extends com.google.protobuf.GeneratedMessageLite {
    <fields>;
}
