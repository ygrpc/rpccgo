import 'package:rpccgo_connect_greeter_dart_client/rpccgo.dart';

void main(List<String> args) {
  const client = GreeterRpccgoClient();

  final unary = client.SayHello(
    SayHelloRequest(name: 'dart-ffi', city: 'connect-greeter'),
  );
  print('dart unary: ${unary.message}');

  final collect = client.Collect();
  collect.send(SayHelloRequest(name: 'ada', city: 'dart'));
  collect.send(SayHelloRequest(name: 'grace', city: 'dart'));
  final collectResponse = collect.finish();
  print('dart collect: ${collectResponse.message}');

  final broadcast = client.Broadcast(
    SayHelloRequest(name: 'stream', city: 'dart'),
  );
  print('dart broadcast: ${broadcast.read().message}');
  print('dart broadcast: ${broadcast.read().message}');
  broadcast.finish();

  final chat = client.Chat();
  for (final name in ['ada', 'grace']) {
    chat.send(SayHelloRequest(name: name, city: 'dart'));
    print('dart chat: ${chat.read().message}');
  }
  chat.closeSend();
  chat.finish();
}
