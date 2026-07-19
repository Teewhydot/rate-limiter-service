class RateLimitResponse {
  final bool allowed;
  final int remaining;
  final int limit;
  final int resetAt;
  final int? retryAfter;

  RateLimitResponse({
    required this.allowed,
    required this.remaining,
    required this.limit,
    required this.resetAt,
    this.retryAfter,
  });

  factory RateLimitResponse.fromJson(Map<String, dynamic> json) {
    return RateLimitResponse(
      allowed: json['allowed'] as bool,
      remaining: json['remaining'] as int,
      limit: json['limit'] as int,
      resetAt: json['reset_at'] as int,
      retryAfter: json['retry_after'] as int?,
    );
  }

  DateTime get resetTime => DateTime.fromMillisecondsSinceEpoch(resetAt * 1000);

  double get usagePercentage => ((limit - remaining) / limit) * 100;
}
