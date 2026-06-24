import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';

const _assetName = 'gen/rpccgo.dart';

void main(List<String> args) async {
  await build(args, (input, output) async {
    if (!input.config.buildCodeAssets) return;

    output.assets.code.add(
      CodeAsset(
        package: input.packageName,
        name: _assetName,
        linkMode: DynamicLoadingSystem(Uri.file('librpccgo_flutter_shared.so')),
      ),
    );
  });
}
