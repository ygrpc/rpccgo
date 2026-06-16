import com.google.protobuf.gradle.id
import com.google.protobuf.gradle.proto

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
    id("com.google.protobuf")
}

val flutterAppDir = rootProject.projectDir.parentFile
val exampleDir = flutterAppDir.parentFile
val protoDir = exampleDir.resolve("proto")
val buildSharedSoForAndroid by tasks.registering(Exec::class) {
    group = "build"
    description = "Builds the rpccgo Android shared libraries consumed by Flutter FFI and Kotlin/JNI."
    workingDir = exampleDir
    commandLine("bash", exampleDir.resolve("flutter_app/tool/build_android_so.sh").absolutePath)
    inputs.dir(exampleDir.resolve("cmd/rpc"))
    inputs.dir(exampleDir.resolve("proto"))
    inputs.file(exampleDir.resolve("go.mod"))
    inputs.file(exampleDir.resolve("go.sum"))
    outputs.dir(projectDir.resolve("src/main/jniLibs"))
}

android {
    namespace = "com.ygrpc.examples.rpccgofluttersharedso"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "com.ygrpc.examples.rpccgofluttersharedso"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
        ndk {
            abiFilters += listOf("arm64-v8a", "x86_64")
        }
    }

    buildTypes {
        release {
            // TODO: Add your own signing config for the release build.
            // Signing with the debug keys for now, so `flutter run --release` works.
            signingConfig = signingConfigs.getByName("debug")
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }
}

dependencies {
    implementation("com.google.protobuf:protobuf-javalite:4.33.2")
}

tasks.named("preBuild") {
    dependsOn(buildSharedSoForAndroid)
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:4.33.2"
    }
    generateProtoTasks {
        all().forEach { task ->
            task.builtins {
                id("java") {
                    option("lite")
                }
            }
        }
    }
}

android.sourceSets.named("main") {
    proto {
        srcDir(protoDir)
    }
}

flutter {
    source = "../.."
}
