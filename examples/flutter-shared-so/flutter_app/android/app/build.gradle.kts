import com.google.protobuf.gradle.id
import com.google.protobuf.gradle.proto

plugins {
    id("com.android.application")
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
    inputs.dir(exampleDir.resolve("internal"))
    inputs.dir(exampleDir.resolve("proto"))
    inputs.file(exampleDir.resolve("flutter_app/tool/build_android_so.sh"))
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
        applicationId = "com.ygrpc.examples.rpccgofluttersharedso"
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        versionCode = flutter.versionCode
        versionName = flutter.versionName
        ndk {
            abiFilters.addAll(listOf("arm64-v8a", "armeabi-v7a", "x86_64"))
        }
        externalNativeBuild {
            cmake {
                arguments += listOf("-DANDROID_STL=c++_shared")
            }
        }
    }

    externalNativeBuild {
        cmake {
            path = file("src/main/cpp/CMakeLists.txt")
        }
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("debug")
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    sourceSets.named("main") {
        proto {
            srcDir(protoDir)
        }
    }
}

dependencies {
    implementation("com.google.protobuf:protobuf-javalite:4.33.2")
}

tasks.named("preBuild") {
    dependsOn(buildSharedSoForAndroid)
}

tasks.matching { it.name.startsWith("externalNativeBuild") }.configureEach {
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

flutter {
    source = "../.."
}
