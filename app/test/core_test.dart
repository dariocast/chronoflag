import 'package:chronoflag_app/main.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses Chronoflag control and view capability links', () {
    expect(
      Capability.parse(Uri.parse('https://app.chronoflag.com/c/control-token')),
      const Capability('control-token', Access.control),
    );
    expect(
      Capability.parse(Uri.parse('https://app.chronoflag.com/v/view-token')),
      const Capability('view-token', Access.view),
    );
    expect(Capability.parse(Uri.parse('https://example.com/other')), isNull);
  });

  test('projects a running stopwatch from its server anchor', () {
    final clock = Clock.fromJson({
      'id': 'c',
      'type': 'stopwatch',
      'state': 'running',
      'accumulated': 1000000000,
      'anchor': '2026-07-16T12:00:00.000Z',
    });
    expect(
      ClockProjection.elapsed(
        clock,
        DateTime.parse('2026-07-16T12:00:02.500Z'),
      ),
      3500,
    );
  });
}
