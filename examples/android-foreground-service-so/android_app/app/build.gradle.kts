plugins {
    id("com.android.application")
}

val exampleDir = rootProject.projectDir.parentFile
val buildSharedSoForAndroid by tasks.registering(Exec::class) {
    group = "build"
    description = "Builds the rpccgo Android shared library."
    workingDir = exampleDir
    commandLine("bash", exampleDir.resolve("android_app/tool/build_android_so.sh").absolutePath)
    inputs.dir(exampleDir.resolve("cmd/rpc"))
    inputs.dir(exampleDir.resolve("proto"))
    inputs.file(exampleDir.resolve("android_app/tool/build_android_so.sh"))
    inputs.file(exampleDir.resolve("go.mod"))
    inputs.file(exampleDir.resolve("go.sum"))
    outputs.dir(projectDir.resolve("src/main/jniLibs"))
}

android {
    namespace = "com.ygrpc.examples.rpccgoandroidforegroundservice"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.ygrpc.examples.rpccgoandroidforegroundservice"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "1.0"
        ndk {
            abiFilters.addAll(listOf("arm64-v8a", "armeabi-v7a", "x86_64"))
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        compilerOptions {
            jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
        }
    }

    externalNativeBuild {
        cmake {
            path = file("src/main/cpp/CMakeLists.txt")
        }
    }

    buildTypes {
        release {
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

}

dependencies {
    implementation("androidx.annotation:annotation:1.9.1")
    implementation("com.google.protobuf:protobuf-javalite:4.33.2")
}

tasks.named("preBuild") {
    dependsOn(buildSharedSoForAndroid)
}

tasks.matching { it.name.startsWith("externalNativeBuild") }.configureEach {
    dependsOn(buildSharedSoForAndroid)
}
