import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'gen/rpccgo.dart';

void main() {
  runApp(const RpccgoLifecycleScope(child: SharedSoApp()));
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
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF006C67)),
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
  static const _command = MethodChannel('rpccgo.shared.so/command');
  static const _events = EventChannel('rpccgo.shared.so/events');

  final _client = const SharedSoDemoRpccgoClient();
  final _android = const AndroidDeviceRpccgoClient();
  final _logs = <String>['app opened'];
  RuntimeStateResponse? _dartState;
  String? _kotlinLine;
  SharedSoDemoWatchRuntimeStateStream? _countStream;
  StreamSubscription<dynamic>? _serviceEvents;
  bool _kotlinStreamRunning = false;
  bool _torchOn = false;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _serviceEvents = _events.receiveBroadcastStream().listen((value) {
      final line = value?.toString() ?? '';
      if (line.isEmpty) return;
      setState(() {
        _kotlinLine = line;
        if (line.startsWith('kotlin stream done')) {
          _kotlinStreamRunning = false;
        }
      });
      _append(line);
    }, onError: (error) => _append('service event error=$error'));
  }

  @override
  void dispose() {
    _countStream?.Close();
    _countStream = null;
    _serviceEvents?.cancel();
    super.dispose();
  }

  Future<void> _callKotlin(String method) async {
    await _run(method, () async {
      await _command.invokeMethod<void>(method);
      _append('$method requested');
    });
  }

  Future<void> _dartRead() async {
    await _run('dart read', () async {
      final result = _client.ReadRuntimeState(
        ReadRuntimeStateRequest(caller: 'dart-ffi-read'),
      );
      final value = result.value;
      if (result.error != null || value == null) {
        _append('dart read error=${result.error ?? "missing response"}');
        return;
      }
      setState(() => _dartState = value);
      _append(_formatState('dart read', value));
    });
  }

  Future<void> _dartIncrement() async {
    await _run('dart increment', () async {
      final result = _client.IncrementRuntimeState(
        IncrementRuntimeStateRequest(delta: 1, caller: 'dart-ffi-increment'),
      );
      final value = result.value;
      if (result.error != null || value == null) {
        _append('dart increment error=${result.error ?? "missing response"}');
        return;
      }
      setState(() => _dartState = value);
      _append(_formatState('dart increment', value));
    });
  }

  Future<void> _startCountStream() async {
    await _run('start count stream', () async {
      if (_countStream != null) {
        _append('count stream already running');
        return;
      }
      final result = _client.WatchRuntimeStateStartCallback(
        ReadRuntimeStateRequest(caller: 'dart-ffi-count-stream'),
        onRecv: (value) {
          setState(() => _dartState = value);
          _append(_formatState('dart count', value));
        },
        onDone: (error) {
          _countStream = null;
          _append('dart count done error=${error ?? "none"}');
        },
      );
      if (result.error != null || result.value == null) {
        _append('start count error=${result.error ?? "missing stream"}');
        return;
      }
      setState(() => _countStream = result.value);
      _append('dart count stream started');
    });
  }

  Future<void> _stopCountStream() async {
    await _run('stop count stream', () async {
      final stream = _countStream;
      if (stream == null) {
        _append('count stream not running');
        return;
      }
      final error = stream.Cancel();
      setState(() => _countStream = null);
      _append('dart count cancel error=${error ?? "none"}');
    });
  }

  Future<void> _startKotlinStream() async {
    await _run('kotlin start stream', () async {
      final started = await _command.invokeMethod<bool>('kotlinStartStream');
      if (started != true) {
        _append('kotlin stream start failed');
        return;
      }
      setState(() => _kotlinStreamRunning = true);
      _append('kotlin stream started');
    });
  }

  Future<void> _stopKotlinStream() async {
    await _run('kotlin stop stream', () async {
      final stopped = await _command.invokeMethod<bool>('kotlinStopStream');
      setState(() => _kotlinStreamRunning = false);
      _append('kotlin stream stop result=${stopped == true}');
    });
  }

  Future<void> _toggleTorch() async {
    await _run('torch toggle', () async {
      final next = !_torchOn;
      final result = _android.SetTorch(
        SetTorchRequest(enabled: next, caller: 'dart-ffi-go-kotlin'),
      );
      final value = result.value;
      if (result.error != null || value == null) {
        _append('torch error=${result.error ?? "missing response"}');
        return;
      }
      setState(() => _torchOn = value.enabled);
      _append(
        'torch ${value.status} camera=${value.cameraId} caller=${value.caller}',
      );
    });
  }

  Future<void> _probeTorchStreams() async {
    await _run('torch stream', () async {
      final watch = _android.WatchTorchStart(
        SetTorchRequest(enabled: false, caller: 'dart-watch-torch'),
      );
      final watchStream = watch.value;
      if (watch.error != null || watchStream == null) {
        _append('torch watch start error=${watch.error ?? "missing stream"}');
        return;
      }
      final watched = watchStream.Recv();
      final watchCancel = watchStream.Cancel();
      if (watched.error != null || watched.value == null) {
        _append(
          'torch watch recv error=${watched.error ?? "missing response"}',
        );
        return;
      }
      _append(
        'torch watch ${watched.value!.status} cancel=${watchCancel ?? "none"}',
      );

      final collect = _android.CollectTorchStart();
      final collectStream = collect.value;
      if (collect.error != null || collectStream == null) {
        _append(
          'torch collect start error=${collect.error ?? "missing stream"}',
        );
        return;
      }
      final collectSend = collectStream.Send(
        SetTorchRequest(enabled: false, caller: 'dart-collect-torch'),
      );
      if (collectSend != null) {
        _append('torch collect send error=$collectSend');
        return;
      }
      final collected = collectStream.Finish();
      if (collected.error != null || collected.value == null) {
        _append(
          'torch collect finish error=${collected.error ?? "missing response"}',
        );
        return;
      }
      _append('torch collect ${collected.value!.status}');

      final chat = _android.ChatTorchStart();
      final chatStream = chat.value;
      if (chat.error != null || chatStream == null) {
        _append('torch chat start error=${chat.error ?? "missing stream"}');
        return;
      }
      final chatSend = chatStream.Send(
        SetTorchRequest(enabled: false, caller: 'dart-chat-torch'),
      );
      if (chatSend != null) {
        _append('torch chat send error=$chatSend');
        return;
      }
      final chatted = chatStream.Recv();
      final closeSend = chatStream.CloseSend();
      final finish = chatStream.Finish();
      if (chatted.error != null || chatted.value == null) {
        _append('torch chat recv error=${chatted.error ?? "missing response"}');
        return;
      }
      _append(
        'torch chat ${chatted.value!.status} close=${closeSend ?? "none"} finish=${finish ?? "none"}',
      );
    });
  }

  Future<void> _run(String label, Future<void> Function() action) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await action();
    } catch (error) {
      _append('$label failed: $error');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  String _formatState(String label, RuntimeStateResponse value) {
    return '$label value=${value.value} rev=${value.revision} pid=${value.pid} instance=${value.instanceAddress}';
  }

  void _append(String line) {
    if (!mounted) return;
    setState(() {
      _logs.insert(
        0,
        '${DateTime.now().toIso8601String().substring(11, 19)}  $line',
      );
      if (_logs.length > 80) _logs.removeLast();
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('Shared .so Runtime')),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _StatePanel(
                kotlinLine: _kotlinLine,
                dartState: _dartState,
                dartStreamRunning: _countStream != null,
                kotlinStreamRunning: _kotlinStreamRunning,
                torchOn: _torchOn,
              ),
              const SizedBox(height: 12),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  FilledButton.icon(
                    onPressed: _busy ? null : () => _callKotlin('kotlinRead'),
                    icon: const Icon(Icons.android),
                    label: const Text('Kotlin Read'),
                  ),
                  FilledButton.icon(
                    onPressed: _busy
                        ? null
                        : () => _callKotlin('kotlinIncrement'),
                    icon: const Icon(Icons.add_circle_outline),
                    label: const Text('Kotlin Increment'),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: _busy ? null : _dartRead,
                    icon: const Icon(Icons.memory),
                    label: const Text('Dart Read'),
                  ),
                  FilledButton.tonalIcon(
                    onPressed: _busy ? null : _dartIncrement,
                    icon: const Icon(Icons.add),
                    label: const Text('Dart Increment'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _busy ? null : _startCountStream,
                    icon: const Icon(Icons.play_arrow),
                    label: const Text('Dart Start Stream'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _busy ? null : _stopCountStream,
                    icon: const Icon(Icons.stop),
                    label: const Text('Dart Stop Stream'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _busy ? null : _startKotlinStream,
                    icon: const Icon(Icons.play_circle_outline),
                    label: const Text('Kotlin Start Stream'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _busy ? null : _stopKotlinStream,
                    icon: const Icon(Icons.stop_circle_outlined),
                    label: const Text('Kotlin Stop Stream'),
                  ),
                  FilledButton.icon(
                    onPressed: _busy ? null : _toggleTorch,
                    icon: Icon(
                      _torchOn ? Icons.flashlight_off : Icons.flashlight_on,
                    ),
                    label: Text(_torchOn ? 'Torch Off' : 'Torch On'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _busy ? null : _probeTorchStreams,
                    icon: const Icon(Icons.stream),
                    label: const Text('Torch Stream'),
                  ),
                  OutlinedButton.icon(
                    onPressed: _busy ? null : SystemNavigator.pop,
                    icon: const Icon(Icons.close_fullscreen),
                    label: const Text('Close Activity'),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Text('Log', style: theme.textTheme.titleMedium),
              const SizedBox(height: 8),
              Expanded(
                child: DraggableScrollableSheet(
                  initialChildSize: 0.38,
                  minChildSize: 0.18,
                  maxChildSize: 0.85,
                  builder: (context, scrollController) => DecoratedBox(
                    decoration: BoxDecoration(
                      border: Border.all(color: theme.dividerColor),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        const SizedBox(height: 8),
                        Center(
                          child: Container(
                            width: 48,
                            height: 4,
                            decoration: BoxDecoration(
                              color: theme.dividerColor,
                              borderRadius: BorderRadius.circular(2),
                            ),
                          ),
                        ),
                        const SizedBox(height: 8),
                        Expanded(
                          child: ListView.separated(
                            controller: scrollController,
                            reverse: true,
                            padding: const EdgeInsets.all(12),
                            itemCount: _logs.length,
                            separatorBuilder: (_, _) => const Divider(height: 12),
                            itemBuilder: (_, index) => SelectableText(
                              _logs[index],
                              style: theme.textTheme.bodyMedium?.copyWith(
                                fontFamily: 'monospace',
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _StatePanel extends StatelessWidget {
  const _StatePanel({
    required this.kotlinLine,
    required this.dartState,
    required this.dartStreamRunning,
    required this.kotlinStreamRunning,
    required this.torchOn,
  });

  final String? kotlinLine;
  final RuntimeStateResponse? dartState;
  final bool dartStreamRunning;
  final bool kotlinStreamRunning;
  final bool torchOn;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Service: ${kotlinLine == null ? "waiting" : "running"}'),
            Text('Dart stream: ${dartStreamRunning ? "running" : "stopped"}'),
            Text(
              'Kotlin stream: ${kotlinStreamRunning ? "running" : "stopped"}',
            ),
            Text('Torch: ${torchOn ? "on" : "off"}'),
            const SizedBox(height: 8),
            SelectableText('Kotlin/JNI: ${kotlinLine ?? "no state yet"}'),
            SelectableText(
              'Dart/FFI: ${dartState == null ? "no state yet" : _dartSummary(dartState!)}',
            ),
          ],
        ),
      ),
    );
  }

  String _dartSummary(RuntimeStateResponse value) {
    return 'value=${value.value} rev=${value.revision} pid=${value.pid} instance=${value.instanceAddress}';
  }
}
