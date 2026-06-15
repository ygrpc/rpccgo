import 'dart:io';

import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';

const _assetName = 'gen/rpccgo.dart';
const _artifactRelativePath = '../build/librpccgo_connect_greeter.so';
const _bundledLibraryName = 'librpccgo_connect_greeter.so';

void main(List<String> args) async {
  await build(args, (input, output) async {
    if (!input.config.buildCodeAssets) {
      return;
    }

    if (input.config.code.targetOS != OS.linux) {
      throw UnsupportedError(
        'This example hook currently supports only linux targets. '
        'Publish target-specific rpccgo runtime artifacts before using it for ${input.config.code.targetOS.name}.',
      );
    }

    // Re-export the existing rpccgo c-shared artifact so Dart and other
    // consumers keep using the same runtime binary instead of rebuilding it.
    final source = File.fromUri(
      input.packageRoot.resolve(_artifactRelativePath),
    );
    if (!await source.exists()) {
      throw StateError(
        'Expected a prebuilt rpccgo runtime at ${source.path}. '
        'Build ../build/librpccgo_connect_greeter.so before running this package.',
      );
    }

    output.dependencies.add(source.uri);

    final bundled = File.fromUri(
      input.outputDirectory.resolve(_bundledLibraryName),
    );
    await bundled.parent.create(recursive: true);
    if (await bundled.exists()) {
      await bundled.delete();
    }
    await source.copy(bundled.path);

    output.assets.code.add(
      CodeAsset(
        package: input.packageName,
        name: _assetName,
        linkMode: DynamicLoadingBundled(),
        file: bundled.uri,
      ),
    );
  });
}
