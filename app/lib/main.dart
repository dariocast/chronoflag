import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() => runApp(const ChronoflagApp());

enum Access { control, view }

class Capability {
  const Capability(
    this.token,
    this.access, [
    this.origin = 'https://app.chronoflag.com',
    this.viewToken,
  ]);
  final String token, origin;
  final String? viewToken;
  final Access access;
  String get path => '/${access == Access.control ? 'c' : 'v'}/$token';
  Uri get link => Uri.parse('$origin$path');
  Uri get viewerLink => Uri.parse('$origin/v/${viewToken ?? token}');
  static Capability? parse(Uri uri) {
    if (uri.pathSegments.length != 2 ||
        !['c', 'v'].contains(uri.pathSegments.first) ||
        uri.pathSegments.last.isEmpty) {
      return null;
    }
    return Capability(
      uri.pathSegments.last,
      uri.pathSegments.first == 'c' ? Access.control : Access.view,
      '${uri.scheme}://${uri.host}${uri.hasPort ? ':${uri.port}' : ''}',
    );
  }

  Map<String, dynamic> toJson() => {
    'token': token,
    'access': access.name,
    'origin': origin,
    'view_token': viewToken,
  };
  factory Capability.fromJson(Map<String, dynamic> json) => Capability(
    json['token'],
    json['access'] == 'control' ? Access.control : Access.view,
    json['origin'] ?? 'https://app.chronoflag.com',
    json['view_token'],
  );
  @override
  bool operator ==(Object other) =>
      other is Capability &&
      other.token == token &&
      other.access == access &&
      other.origin == origin;
  @override
  int get hashCode => Object.hash(token, access, origin);
}

class Clock {
  Clock({
    required this.id,
    required this.type,
    required this.state,
    required this.accumulated,
    required this.duration,
    this.anchor,
    this.label = '',
    this.laps = const [],
  });
  final String id, type, state, label;
  final int accumulated, duration;
  final DateTime? anchor;
  final List<dynamic> laps;
  factory Clock.fromJson(Map<String, dynamic> j) => Clock(
    id: j['id'],
    type: j['type'],
    state: j['state'],
    accumulated: ((j['accumulated'] ?? 0) as num).toInt() ~/ 1000000,
    duration: ((j['duration'] ?? 0) as num).toInt() ~/ 1000000,
    anchor: j['anchor'] == null ? null : DateTime.tryParse(j['anchor']),
    label: j['label'] ?? '',
    laps: j['laps'] ?? [],
  );
}

class ClockProjection {
  static int elapsed(Clock c, DateTime now) =>
      c.state == 'running' && c.anchor != null
      ? c.accumulated +
            max(0, now.toUtc().difference(c.anchor!.toUtc()).inMilliseconds)
      : c.accumulated;
  static String display(Clock c, DateTime now) {
    var value = elapsed(c, now);
    if (c.type == 'timer') value = max(0, c.duration - value);
    final cs = value ~/ 10, s = cs ~/ 100, h = s ~/ 3600, m = (s % 3600) ~/ 60;
    final prefix = h > 0 ? '${h.toString().padLeft(2, '0')}:' : '';
    return '$prefix${m.toString().padLeft(2, '0')}:${(s % 60).toString().padLeft(2, '0')}.${(cs % 100).toString().padLeft(2, '0')}';
  }
}

class Snapshot {
  Snapshot({
    required this.id,
    required this.title,
    required this.lifecycle,
    required this.clocks,
    this.highlighted = '',
  });
  final String id, title, lifecycle, highlighted;
  final List<Clock> clocks;
  factory Snapshot.fromJson(Map<String, dynamic> j) => Snapshot(
    id: j['id'],
    title: j['title'] ?? '',
    lifecycle: j['lifecycle'] ?? 'active',
    highlighted: j['highlighted_clock_id'] ?? '',
    clocks: (j['clocks'] as List).map((e) => Clock.fromJson(e)).toList(),
  );
}

class Api {
  Api(this.origin);
  final String origin;
  Future<Map<String, dynamic>> _request(
    String method,
    String path, [
    Object? body,
    Map<String, String>? headers,
  ]) async {
    final client = HttpClient();
    try {
      final request = await client.openUrl(method, Uri.parse('$origin$path'));
      request.headers.contentType = ContentType.json;
      headers?.forEach(request.headers.set);
      if (body != null) request.write(jsonEncode(body));
      final response = await request.close();
      final text = await response.transform(utf8.decoder).join();
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw HttpException(jsonDecode(text)['error'] ?? 'request_failed');
      }
      return text.isEmpty ? {} : jsonDecode(text);
    } finally {
      client.close(force: true);
    }
  }

  Future<Capability> create() async {
    final data = await _request('POST', '/api/v1/instances', {});
    final control = Capability.parse(
      Uri.parse('$origin${data['control_url']}'),
    )!;
    return Capability(
      control.token,
      control.access,
      origin,
      Uri.parse(data['view_url']).pathSegments.last,
    );
  }

  Future<Snapshot> snapshot(Capability c) async => Snapshot.fromJson(
    await _request('GET', '/api/v1/${c.access.name}/${c.token}'),
  );
  Future<Snapshot> command(
    Capability c,
    Clock clock,
    String type,
  ) async => Snapshot.fromJson(
    await _request(
      'POST',
      '/api/v1/control/${c.token}/clocks/${clock.id}/commands',
      {'type': type, 'device_id': 'flutter'},
      {
        'Idempotency-Key':
            '${DateTime.now().microsecondsSinceEpoch}-${Random().nextInt(1 << 32)}',
      },
    ),
  );
  Future<Snapshot> add(Capability c, String type, int duration) async =>
      Snapshot.fromJson(
        await _request('POST', '/api/v1/control/${c.token}/clocks', {
          'type': type,
          'duration_ms': duration,
        }),
      );
  Future<Snapshot> patch(
    Capability c,
    String id,
    Map<String, dynamic> body,
  ) async => Snapshot.fromJson(
    await _request('PATCH', '/api/v1/control/${c.token}/clocks/$id', body),
  );
  Future<Snapshot> title(Capability c, String value) async => Snapshot.fromJson(
    await _request('PATCH', '/api/v1/control/${c.token}', {'title': value}),
  );
  Future<Capability> rotate(Capability c, Access access) async {
    final data = await _request(
      'POST',
      '/api/v1/control/${c.token}/capabilities/${access.name}/rotate',
    );
    final next = Capability.parse(Uri.parse('$origin${data['url']}'))!;
    return access == Access.control
        ? Capability(next.token, next.access, origin, c.viewToken)
        : Capability(c.token, c.access, origin, next.token);
  }
}

class Sessions {
  static const key = 'chronoflag.sessions.v1';
  Future<List<Capability>> load() async {
    final value = (await SharedPreferences.getInstance()).getString(key);
    if (value == null) return [];
    return (jsonDecode(value) as List)
        .map((e) => Capability.fromJson(e))
        .toList();
  }

  Future<void> save(Capability c) async {
    final all = await load();
    all.removeWhere((e) => e == c);
    all.insert(0, c);
    await (await SharedPreferences.getInstance()).setString(
      key,
      jsonEncode(all.map((e) => e.toJson()).toList()),
    );
  }

  Future<void> remove(Capability c) async {
    final all = await load();
    all.removeWhere((e) => e == c);
    await (await SharedPreferences.getInstance()).setString(
      key,
      jsonEncode(all.map((e) => e.toJson()).toList()),
    );
  }
}

class ChronoflagApp extends StatelessWidget {
  const ChronoflagApp({super.key});
  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Chronoflag',
    theme: ThemeData(
      useMaterial3: true,
      colorSchemeSeed: const Color(0xff9ebf00),
    ),
    darkTheme: ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      colorSchemeSeed: const Color(0xffc6f000),
    ),
    home: const HomeScreen(),
  );
}

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});
  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final sessions = Sessions();
  List<Capability> items = [];
  @override
  void initState() {
    super.initState();
    sessions.load().then((v) {
      if (mounted) setState(() => items = v);
    });
  }

  Future<void> open(Capability c) async {
    await sessions.save(c);
    if (!mounted) return;
    await Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => BoardScreen(capability: c)),
    );
    setState(() {});
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Chronoflag')),
    floatingActionButton: FloatingActionButton.extended(
      onPressed: () async {
        final c = await Api('https://app.chronoflag.com').create();
        await open(c);
      },
      icon: const Icon(Icons.add),
      label: const Text('New session'),
    ),
    body: ListView(
      children: [
        Padding(
          padding: const EdgeInsets.all(16),
          child: Text(
            'Recent sessions',
            style: Theme.of(context).textTheme.headlineSmall,
          ),
        ),
        for (final c in items)
          ListTile(
            leading: Icon(
              c.access == Access.control ? Icons.timer : Icons.visibility,
            ),
            title: Text(
              c.access == Access.control ? 'Operator session' : 'Live board',
            ),
            subtitle: Text(c.origin),
            onTap: () => open(c),
            trailing: IconButton(
              icon: const Icon(Icons.delete_outline),
              onPressed: () async {
                await sessions.remove(c);
                setState(() => items.remove(c));
              },
            ),
          ),
        ListTile(
          leading: const Icon(Icons.qr_code_scanner),
          title: const Text('Scan QR code'),
          onTap: () => Navigator.push(
            context,
            MaterialPageRoute(builder: (_) => Scanner(onFound: open)),
          ),
        ),
      ],
    ),
  );
}

class Scanner extends StatelessWidget {
  const Scanner({super.key, required this.onFound});
  final Future<void> Function(Capability) onFound;
  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Scan Chronoflag QR')),
    body: MobileScanner(
      onDetect: (capture) {
        final value = capture.barcodes.firstOrNull?.rawValue;
        final c = value == null
            ? null
            : Capability.parse(Uri.tryParse(value) ?? Uri());
        if (c != null) {
          Navigator.pop(context);
          onFound(c);
        }
      },
    ),
  );
}

class BoardScreen extends StatefulWidget {
  const BoardScreen({super.key, required this.capability});
  final Capability capability;
  @override
  State<BoardScreen> createState() => _BoardScreenState();
}

class _BoardScreenState extends State<BoardScreen> {
  late Capability capability;
  Snapshot? snapshot;
  String error = '';
  Timer? refresh;
  @override
  void initState() {
    super.initState();
    capability = widget.capability;
    load();
    refresh = Timer.periodic(
      const Duration(seconds: 2),
      (_) => load(silent: true),
    );
  }

  @override
  void dispose() {
    refresh?.cancel();
    super.dispose();
  }

  Api get api => Api(capability.origin);
  Future<void> load({bool silent = false}) async {
    try {
      final s = await api.snapshot(capability);
      if (mounted) setState(() => snapshot = s);
    } catch (e) {
      if (!silent && mounted) setState(() => error = '$e');
    }
  }

  Future<void> mutate(Future<Snapshot> Function() action) async {
    try {
      setState(() => error = '');
      final s = await action();
      if (mounted) setState(() => snapshot = s);
    } catch (e) {
      if (mounted) setState(() => error = '$e');
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = snapshot;
    if (s == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Chronoflag')),
        body: const Center(child: CircularProgressIndicator()),
      );
    }
    final control =
        capability.access == Access.control && s.lifecycle == 'active';
    return Scaffold(
      appBar: AppBar(
        title: Text(s.title.isEmpty ? 'Control deck' : s.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.ios_share),
            onPressed: () => showShare(context, s),
          ),
          if (control)
            IconButton(
              icon: const Icon(Icons.add),
              onPressed: () => addClock(context),
            ),
        ],
      ),
      body: Column(
        children: [
          if (error.isNotEmpty)
            MaterialBanner(
              content: Text(error),
              actions: [
                TextButton(onPressed: load, child: const Text('Retry')),
              ],
            ),
          Expanded(
            child: LayoutBuilder(
              builder: (context, size) => GridView.count(
                crossAxisCount: size.maxWidth > 700 ? 2 : 1,
                childAspectRatio: size.maxWidth > 700 ? 1.6 : 1.2,
                children: [
                  for (final clock in s.clocks)
                    ClockCard(
                      clock: clock,
                      control: control,
                      onCommand: (x) =>
                          mutate(() => api.command(capability, clock, x)),
                      onLabel: (x) => mutate(
                        () => api.patch(capability, clock.id, {'label': x}),
                      ),
                      onFocus: () => mutate(
                        () => api.patch(capability, clock.id, {
                          'highlighted': s.highlighted != clock.id,
                        }),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> addClock(BuildContext context) async {
    final duration = await showDialog<int>(
      context: context,
      builder: (context) {
        var minutes = 5;
        return AlertDialog(
          title: const Text('Add countdown'),
          content: StatefulBuilder(
            builder: (context, set) {
              return Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('$minutes minutes'),
                  Slider(
                    value: minutes.toDouble(),
                    min: 1,
                    max: 180,
                    divisions: 179,
                    label: '$minutes',
                    onChanged: (v) => set(() => minutes = v.round()),
                  ),
                ],
              );
            },
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            TextButton(
              onPressed: () => Navigator.pop(context, 0),
              child: const Text('Stopwatch'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, minutes),
              child: const Text('Countdown'),
            ),
          ],
        );
      },
    );
    if (duration == null) return;
    await mutate(
      () => api.add(
        capability,
        duration == 0 ? 'stopwatch' : 'timer',
        duration * 60000,
      ),
    );
  }

  Future<void> showShare(BuildContext context, Snapshot s) async {
    await showModalBottomSheet(
      context: context,
      builder: (context) => Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text('Share live board'),
            QrImageView(data: capability.viewerLink.toString(), size: 180),
            FilledButton.icon(
              onPressed: () => SharePlus.instance.share(
                ShareParams(text: capability.viewerLink.toString()),
              ),
              icon: const Icon(Icons.share),
              label: const Text('Share viewer link'),
            ),
            if (capability.access == Access.control)
              TextButton(
                onPressed: () async {
                  final replacement = await api.rotate(
                    capability,
                    Access.control,
                  );
                  if (context.mounted) {
                    setState(() => capability = replacement);
                    Navigator.pop(context);
                  }
                },
                child: const Text('Regenerate operator access'),
              ),
          ],
        ),
      ),
    );
  }
}

class ClockCard extends StatefulWidget {
  const ClockCard({
    super.key,
    required this.clock,
    required this.control,
    required this.onCommand,
    required this.onLabel,
    required this.onFocus,
  });
  final Clock clock;
  final bool control;
  final ValueChanged<String> onCommand, onLabel;
  final VoidCallback onFocus;
  @override
  State<ClockCard> createState() => _ClockCardState();
}

class _ClockCardState extends State<ClockCard> {
  late Timer tick;
  @override
  void initState() {
    super.initState();
    tick = Timer.periodic(const Duration(milliseconds: 100), (_) {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    tick.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final c = widget.clock;
    final primary = c.state == 'idle'
        ? 'start'
        : c.state == 'running'
        ? 'pause'
        : c.state == 'paused'
        ? 'resume'
        : 'reset';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Chip(label: Text(c.type)),
                const Spacer(),
                Text(c.state),
              ],
            ),
            if (widget.control)
              TextFormField(
                initialValue: c.label,
                decoration: const InputDecoration(hintText: 'Untitled'),
                onFieldSubmitted: widget.onLabel,
              )
            else
              Text(
                c.label.isEmpty ? 'Untitled' : c.label,
                style: Theme.of(context).textTheme.titleLarge,
              ),
            const Spacer(),
            Center(
              child: Text(
                ClockProjection.display(c, DateTime.now()),
                style: Theme.of(context).textTheme.displayMedium?.copyWith(
                  fontFeatures: const [FontFeature.tabularFigures()],
                ),
              ),
            ),
            const Spacer(),
            if (widget.control)
              Wrap(
                spacing: 8,
                children: [
                  FilledButton(
                    onPressed: () => widget.onCommand(primary),
                    child: Text(
                      primary[0].toUpperCase() + primary.substring(1),
                    ),
                  ),
                  if (c.type == 'stopwatch')
                    OutlinedButton(
                      onPressed: c.state == 'running'
                          ? () => widget.onCommand('lap')
                          : null,
                      child: const Text('Lap'),
                    ),
                  IconButton(
                    onPressed: widget.onFocus,
                    icon: const Icon(Icons.fullscreen),
                  ),
                  if (c.laps.isNotEmpty) Text('${c.laps.length} laps'),
                ],
              ),
          ],
        ),
      ),
    );
  }
}
