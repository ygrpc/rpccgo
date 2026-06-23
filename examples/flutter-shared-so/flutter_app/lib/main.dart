import 'dart:async' as async;

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
  String _latestActivityTitle = 'Latest Result';
  String _latestActivityBody =
      'Choose a call path to see which runtime handled the request and how the shared state changed.';
  Color _latestActivityColor = Colors.grey;
  bool _flutterBusy = false;
  bool _jniBusy = false;
  bool _runtimeBusy = false;
  bool _streamBusy = false;

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
    final response = _client.ComposeGreeting(
      ComposeGreetingRequest(name: _effectiveName, caller: 'flutter-ffi'),
    );
    final error = response.error;
    final value = response.value;
    if (error != null || value == null) {
      setState(() {
        _latestActivityTitle = 'Latest Activity: Flutter FFI (Error)';
        _latestActivityBody =
            'flutter ffi error: ${error ?? 'missing response'}';
        _latestActivityColor = Colors.red;
      });
    } else {
      setState(() {
        _latestActivityTitle = 'Latest Activity: Flutter FFI';
        _latestActivityBody = _formatGreetingResult('Flutter FFI', value);
        _latestActivityColor = const Color(0xFF0B7285);
      });
    }
    setState(() {
      _flutterBusy = false;
    });
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
    final writtenResult = _client.IncrementRuntimeState(
      IncrementRuntimeStateRequest(delta: 1, caller: 'flutter-ffi'),
    );
    final writtenError = writtenResult.error;
    final written = writtenResult.value;
    if (writtenError != null || written == null) {
      setState(() {
        _latestActivityTitle =
            'Latest Activity: Shared Go runtime state (Error)';
        _latestActivityBody =
            'shared runtime verification error: ${writtenError ?? 'missing response'}';
        _latestActivityColor = Colors.red;
        _runtimeBusy = false;
      });
      return;
    }
    try {
      debugPrint(
        'Flutter FFI wrote instance_address=${written.instanceAddress} pid=${written.pid} '
        'value=${written.value} revision=${written.revision}',
      );
      final observed = await _jniChannel.invokeMethod<String>(
        'readRuntimeState',
      );
      debugPrint('Kotlin/JNI observed $observed');
      setState(() {
        _latestActivityTitle = 'Latest Activity: Shared Go runtime state';
        _latestActivityBody =
            'Flutter FFI wrote shared state\n'
            'Value: ${written.value}\n'
            'Revision: ${written.revision}\n'
            'Go instance: ${written.instanceAddress}\n'
            'Process ID: ${written.pid}\n\n'
            'Kotlin/JNI then read the same state\n'
            '${observed ?? 'jni returned null'}';
        _latestActivityColor = const Color(0xFFE67700);
      });
    } catch (error) {
      setState(() {
        _latestActivityTitle =
            'Latest Activity: Shared Go runtime state (Error)';
        _latestActivityBody = 'shared runtime verification error: $error';
        _latestActivityColor = Colors.red;
      });
    } finally {
      setState(() {
        _runtimeBusy = false;
      });
    }
  }

  Future<void> _runStreams() async {
    setState(() {
      _streamBusy = true;
    });
    await _showBusyState();
    try {
      final flutterSummary = await _runFlutterStreams();
      final jniSummary = await _jniChannel.invokeMethod<String>('runStreams');
      setState(() {
        _latestActivityTitle = 'Latest Activity: Streaming';
        _latestActivityBody =
            'Same Go counter. Kotlin/JNI continues after Flutter FFI.\n'
            '$flutterSummary\n'
            '${jniSummary ?? 'Kotlin/JNI: no result'}';
        _latestActivityColor = const Color(0xFF7048E8);
      });
    } catch (error) {
      setState(() {
        _latestActivityTitle = 'Latest Activity: Streaming (Error)';
        _latestActivityBody = 'streaming error: $error';
        _latestActivityColor = Colors.red;
      });
    } finally {
      setState(() {
        _streamBusy = false;
      });
    }
  }

  Future<String> _runFlutterStreams() async {
    final collectResult = _client.CollectRuntimeStateStart();
    final collect = collectResult.value;
    if (collectResult.error != null || collect == null) {
      throw StateError('client stream start: ${collectResult.error}');
    }
    for (final request in [
      IncrementRuntimeStateRequest(
        delta: 2,
        caller: 'flutter-ffi-client-stream-a',
      ),
      IncrementRuntimeStateRequest(
        delta: 3,
        caller: 'flutter-ffi-client-stream-b',
      ),
    ]) {
      final error = collect.Send(request);
      if (error != null) {
        throw StateError('client stream send: $error');
      }
    }
    final collected = collect.Finish();
    final collectedValue = collected.value;
    if (collected.error != null || collectedValue == null) {
      throw StateError('client stream finish: ${collected.error}');
    }

    final serverValues = [];
    final serverDone = async.Completer<void>();
    final streamResult = _client.StreamRuntimeStateStartCallback(
      ReadRuntimeStateRequest(caller: 'flutter-ffi-server-stream'),
      onRecv: (value) {
        serverValues.add(value.value);
      },
      onDone: (error) {
        if (error != null) {
          serverDone.completeError(
            StateError('server stream callback: $error'),
          );
          return;
        }
        serverDone.complete();
      },
    );
    final serverStream = streamResult.value;
    if (streamResult.error != null || serverStream == null) {
      throw StateError('server stream start: ${streamResult.error}');
    }
    await serverDone.future.timeout(
      const Duration(seconds: 3),
      onTimeout: () {
        serverStream.Cancel();
        throw async.TimeoutException('server stream callback timed out');
      },
    );

    final chatResult = _client.ChatRuntimeStateStart();
    final chat = chatResult.value;
    if (chatResult.error != null || chat == null) {
      throw StateError('bidi stream start: ${chatResult.error}');
    }
    final bidiValues = [];
    for (final request in [
      IncrementRuntimeStateRequest(delta: 4, caller: 'flutter-ffi-bidi-a'),
      IncrementRuntimeStateRequest(delta: 5, caller: 'flutter-ffi-bidi-b'),
    ]) {
      final sendError = chat.Send(request);
      if (sendError != null) {
        throw StateError('bidi stream send: $sendError');
      }
      final next = chat.Recv();
      final value = next.value;
      if (next.error != null || value == null) {
        throw StateError('bidi stream read: ${next.error}');
      }
      bidiValues.add(value.value);
    }
    final closeSendError = chat.CloseSend();
    if (closeSendError != null) {
      throw StateError('bidi stream close-send: $closeSendError');
    }
    final finishError = chat.Finish();
    if (finishError != null) {
      throw StateError('bidi stream finish: $finishError');
    }

    return _formatStreamResult(
      'Flutter FFI',
      collectedValue.value,
      collectedValue.revision,
      serverValues,
      bidiValues,
    );
  }

  String _formatGreetingResult(String path, ComposeGreetingResponse value) {
    return '$path unary call\n'
        'Message: ${value.message}\n'
        'Go handler: ${value.servedBy}\n'
        'Shared library: ${value.library}';
  }

  String _formatStreamResult(
    String path,
    Object? finalValue,
    Object? revision,
    List<Object?> serverValues,
    List<Object?> bidiValues,
  ) {
    final lastValue = bidiValues.isEmpty ? finalValue : bidiValues.last;
    return '$path: +2+3 -> $finalValue (rev $revision); '
        'read ${serverValues.join(', ')}; +4+5 -> ${bidiValues.join(' -> ')}; '
        'final $lastValue';
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
                    'Shared .so',
                    style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    'Flutter FFI and Kotlin/JNI call the same Go runtime in one shared library.',
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
                                      : 'Flutter FFI unary',
                                ),
                              ),
                              FilledButton.icon(
                                onPressed: _jniBusy ? null : _callViaJNI,
                                icon: const Icon(Icons.android),
                                label: Text(
                                  _jniBusy ? 'Calling...' : 'Kotlin/JNI unary',
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
                                      : 'Verify shared state',
                                ),
                              ),
                              FilledButton.icon(
                                onPressed: _streamBusy ? null : _runStreams,
                                icon: const Icon(Icons.sync),
                                label: Text(
                                  _streamBusy
                                      ? 'Streaming...'
                                      : 'Compare stream RPCs',
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
                  SelectableText(
                    body,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium?.copyWith(height: 1.35),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
