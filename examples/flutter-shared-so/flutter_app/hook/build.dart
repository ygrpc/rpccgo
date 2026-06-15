import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';

const _assetName = 'gen/rpccgo.dart';

void main(List<String> args) async {
  await build(args, (input, output) async {
    if (!input.config.buildCodeAssets) {
      return;
    }

    if (input.config.code.targetOS != OS.android) {
      throw UnsupportedError(
        'This example loads the Android rpccgo shared library by SONAME. '
        'Current target: ${input.config.code.targetOS.name}.',
      );
    }

    output.assets.code.add(
      CodeAsset(
        package: input.packageName,
        name: _assetName,
        linkMode: DynamicLoadingSystem(Uri.file('librpccgo_flutter_shared.so')),
      ),
    );
  });
}
