import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:rpccgofluttersharedso/gen/rpccgo.dart';

void main() {
  runApp(const SharedSoApp());
}

class SharedSoApp extends StatelessWidget {
  const SharedSoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'rpccgo Shared .so',
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF0B7285)),
      ),
      home: const SharedSoHomePage(),
    );
  }
}

class SharedSoHomePage extends StatefulWidget {
  const SharedSoHomePage({super.key});

  @override
  State<SharedSoHomePage> createState() => _SharedSoHomePageState();
}

class _SharedSoHomePageState extends State<SharedSoHomePage> {
  static const _jniChannel = MethodChannel('rpccgo.shared.so/jni');
  static const _client = SharedSoDemoRpccgoClient();

  final _nameController = TextEditingController(text: 'Ada');
  String _latestActivityTitle = 'Latest Activity';
  String _latestActivityBody = 'Tap any button above to execute a call.';
  Color _latestActivityColor = Colors.grey;
  bool _flutterBusy = false;
  bool _jniBusy = false;
  bool _runtimeBusy = false;

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _callViaFlutter() async {
    setState(() {
      _flutterBusy = true;
    });
    await _showBusyState();
    try {
      final response = _client.ComposeGreeting(
        ComposeGreetingRequest(name: _effectiveName, caller: 'flutter-ffi'),
      );
      setState(() {
        _latestActivityTitle = 'Latest Activity: Flutter FFI';
        _latestActivityBody =
            '${response.message} | served_by=${response.servedBy} | library=${response.library}';
        _latestActivityColor = const Color(0xFF0B7285);
      });
    } catch (error) {
      setState(() {
        _latestActivityTitle = 'Latest Activity: Flutter FFI (Error)';
        _latestActivityBody = 'flutter ffi error: $error';
        _latestActivityColor = Colors.red;
      });
    } finally {
      setState(() {
        _flutterBusy = false;
      });
    }
  }

  Future<void> _callViaJNI() async {
    setState(() {
      _jniBusy = true;
    });
    await _showBusyState();
    try {
      final response = await _jniChannel.invokeMethod<String>(
        'composeGreeting',
        <String, Object?>{'name': _effectiveName},
      );
      setState(() {
        final result = response ?? 'jni returned null';
        _latestActivityTitle = 'Latest Activity: Kotlin/JNI';
        _latestActivityBody = result;
        _latestActivityColor = const Color(0xFF2B8A3E);
      });
    } on PlatformException catch (error) {
      setState(() {
        final errMsg = 'jni platform error: ${error.message ?? error.code}';
        _latestActivityTitle = 'Latest Activity: Kotlin/JNI (Error)';
        _latestActivityBody = errMsg;
        _latestActivityColor = Colors.red;
      });
    } catch (error) {
      setState(() {
        final errMsg = 'jni error: $error';
        _latestActivityTitle = 'Latest Activity: Kotlin/JNI (Error)';
        _latestActivityBody = errMsg;
        _latestActivityColor = Colors.red;
      });
    } finally {
      setState(() {
        _jniBusy = false;
      });
    }
  }

  Future<void> _verifySharedRuntime() async {
    setState(() {
      _runtimeBusy = true;
    });
    await _showBusyState();
    try {
      final written = _client.IncrementRuntimeState(
        IncrementRuntimeStateRequest(delta: 1, caller: 'flutter-ffi'),
      );
      debugPrint(
        'Flutter FFI wrote runtime_id=${written.runtimeId} '
        'value=${written.value} revision=${written.revision}',
      );
      final observed = await _jniChannel.invokeMethod<String>(
        'readRuntimeState',
      );
      debugPrint('Kotlin/JNI observed $observed');
      setState(() {
        _latestActivityTitle = 'Latest Activity: Shared Go runtime state';
        _latestActivityBody =
            'Flutter wrote: runtime_id=${written.runtimeId} | '
            'value=${written.value} | revision=${written.revision}\n'
            'Kotlin read: ${observed ?? 'jni returned null'}';
        _latestActivityColor = const Color(0xFFE67700);
      });
    } catch (error) {
      setState(() {
        _latestActivityTitle = 'Latest Activity: Shared Go runtime state (Error)';
        _latestActivityBody = 'shared runtime verification error: $error';
        _latestActivityColor = Colors.red;
      });
    } finally {
      setState(() {
        _runtimeBusy = false;
      });
    }
  }

  Future<void> _showBusyState() async {
    await WidgetsBinding.instance.endOfFrame;
  }

  String get _effectiveName {
    final value = _nameController.text.trim();
    if (value.isEmpty) {
      return 'Ada';
    }
    return value;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: DecoratedBox(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            colors: [Color(0xFFF3FBFD), Color(0xFFE6F4EA), Color(0xFFFFF7E6)],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
        ),
        child: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 760),
              child: ListView(
                padding: const EdgeInsets.all(24),
                children: [
                  Text(
                    'One .so, two call paths',
                    style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Kotlin/JNI loads the shared library first. Flutter FFI then '
                    'opens the same Android library by SONAME and resolves the '
                    'rpccgo symbols from that library handle.',
                    style: Theme.of(context).textTheme.bodyLarge,
                  ),
                  const SizedBox(height: 24),
                  Card(
                    elevation: 0,
                    color: Colors.white.withValues(alpha: 0.88),
                    child: Padding(
                      padding: const EdgeInsets.all(20),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          TextField(
                            controller: _nameController,
                            decoration: const InputDecoration(
                              labelText: 'Greeting Target',
                              hintText: 'Ada',
                              border: OutlineInputBorder(),
                            ),
                          ),
                          const SizedBox(height: 16),
                          Wrap(
                            spacing: 12,
                            runSpacing: 12,
                            children: [
                              FilledButton.tonalIcon(
                                onPressed: _flutterBusy
                                    ? null
                                    : _callViaFlutter,
                                icon: const Icon(Icons.flutter_dash),
                                label: Text(
                                  _flutterBusy
                                      ? 'Calling...'
                                      : 'Call via Flutter FFI',
                                ),
                              ),
                              FilledButton.icon(
                                onPressed: _jniBusy ? null : _callViaJNI,
                                icon: const Icon(Icons.android),
                                label: Text(
                                  _jniBusy
                                      ? 'Calling...'
                                      : 'Call via Kotlin/JNI',
                                ),
                              ),
                              FilledButton.tonalIcon(
                                onPressed: _runtimeBusy
                                    ? null
                                    : _verifySharedRuntime,
                                icon: const Icon(Icons.sync_alt),
                                label: Text(
                                  _runtimeBusy
                                      ? 'Verifying...'
                                      : 'Flutter write, Kotlin read',
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),
                  _ResultCard(
                    title: _latestActivityTitle,
                    body: _latestActivityBody,
                    stripeColor: _latestActivityColor,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _ResultCard extends StatelessWidget {
  const _ResultCard({
    required this.title,
    required this.body,
    required this.stripeColor,
  });

  final String title;
  final String body;
  final Color stripeColor;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.9),
        borderRadius: BorderRadius.circular(20),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.06),
            blurRadius: 24,
            offset: const Offset(0, 12),
          ),
        ],
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            width: 10,
            decoration: BoxDecoration(
              color: stripeColor,
              borderRadius: const BorderRadius.horizontal(
                left: Radius.circular(20),
              ),
            ),
          ),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 10),
                  SelectableText(body),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
