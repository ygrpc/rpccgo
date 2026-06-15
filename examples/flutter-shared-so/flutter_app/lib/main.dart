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
  String _flutterResult = 'Tap the Flutter FFI button to call rpccgo.';
  String _jniResult = 'Tap the Kotlin/JNI button to call the same library.';
  String _runtimeResult =
      'Tap the shared runtime button: Flutter writes, then Kotlin/JNI reads.';
  String _activityStatus = 'No call has run yet.';
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
      _activityStatus = 'Calling Go through Flutter FFI...';
    });
    await _showBusyState();
    try {
      final response = _client.ComposeGreeting(
        ComposeGreetingRequest(name: _effectiveName, caller: 'flutter-ffi'),
      );
      setState(() {
        _flutterResult =
            '${response.message} | served_by=${response.servedBy} | library=${response.library}';
        _activityStatus = 'Flutter FFI succeeded: ${response.message}';
      });
    } catch (error) {
      setState(() {
        _flutterResult = 'flutter ffi error: $error';
        _activityStatus = 'Flutter FFI failed: $error';
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
      _activityStatus = 'Calling Go through Kotlin/JNI...';
    });
    await _showBusyState();
    try {
      final response = await _jniChannel.invokeMethod<String>(
        'composeGreeting',
        <String, Object?>{'name': _effectiveName},
      );
      setState(() {
        _jniResult = response ?? 'jni returned null';
        _activityStatus = 'Kotlin/JNI succeeded: $_jniResult';
      });
    } on PlatformException catch (error) {
      setState(() {
        _jniResult = 'jni platform error: ${error.message ?? error.code}';
        _activityStatus = _jniResult;
      });
    } catch (error) {
      setState(() {
        _jniResult = 'jni error: $error';
        _activityStatus = _jniResult;
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
      _activityStatus = 'Flutter is writing shared Go runtime state...';
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
        _runtimeResult =
            'Flutter wrote: runtime_id=${written.runtimeId} | '
            'value=${written.value} | revision=${written.revision}\n'
            'Kotlin read: ${observed ?? 'jni returned null'}';
        _activityStatus =
            'Shared runtime verified: value=${written.value}, '
            'revision=${written.revision}, runtime_id=${written.runtimeId}';
      });
    } catch (error) {
      setState(() {
        _runtimeResult = 'shared runtime verification error: $error';
        _activityStatus = 'Shared runtime verification failed: $error';
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
                          const SizedBox(height: 20),
                          Text(
                            'Latest activity',
                            style: Theme.of(context).textTheme.labelLarge,
                          ),
                          const SizedBox(height: 6),
                          SelectableText(_activityStatus),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),
                  _ResultCard(
                    title: 'Flutter FFI',
                    body: _flutterResult,
                    stripeColor: const Color(0xFF0B7285),
                  ),
                  const SizedBox(height: 16),
                  _ResultCard(
                    title: 'Kotlin/JNI',
                    body: _jniResult,
                    stripeColor: const Color(0xFF2B8A3E),
                  ),
                  const SizedBox(height: 16),
                  _ResultCard(
                    title: 'Shared Go runtime state',
                    body: _runtimeResult,
                    stripeColor: const Color(0xFFE67700),
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
