class UsageStats {
  final String clientId;
  final int totalRequests;
  final int allowedRequests;
  final int blockedRequests;
  final double avgResponseTimeMs;
  final DateTime periodStart;
  final DateTime periodEnd;

  UsageStats({
    required this.clientId,
    required this.totalRequests,
    required this.allowedRequests,
    required this.blockedRequests,
    required this.avgResponseTimeMs,
    required this.periodStart,
    required this.periodEnd,
  });

  factory UsageStats.fromJson(Map<String, dynamic> json) {
    return UsageStats(
      clientId: json['client_id'] as String,
      totalRequests: json['total_requests'] as int,
      allowedRequests: json['allowed_requests'] as int,
      blockedRequests: json['blocked_requests'] as int,
      avgResponseTimeMs: (json['avg_response_time_ms'] as num).toDouble(),
      periodStart: DateTime.parse(json['period_start']),
      periodEnd: DateTime.parse(json['period_end']),
    );
  }

  double get blockRate {
    if (totalRequests == 0) return 0.0;
    return (blockedRequests / totalRequests) * 100;
  }

  double get allowRate {
    if (totalRequests == 0) return 0.0;
    return (allowedRequests / totalRequests) * 100;
  }
}
