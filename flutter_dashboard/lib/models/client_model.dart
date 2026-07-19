class Client {
  final String id;
  final String name;
  final int limit;
  final int windowSec;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  Client({
    required this.id,
    required this.name,
    required this.limit,
    required this.windowSec,
    this.createdAt,
    this.updatedAt,
  });

  factory Client.fromJson(Map<String, dynamic> json) {
    return Client(
      id: json['id'] as String,
      name: json['name'] as String,
      limit: json['limit'] as int,
      windowSec: json['window_sec'] as int,
      createdAt: json['created_at'] != null
          ? DateTime.tryParse(json['created_at'])
          : null,
      updatedAt: json['updated_at'] != null
          ? DateTime.tryParse(json['updated_at'])
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'limit': limit,
      'window_sec': windowSec,
    };
  }

  String get windowDisplay {
    if (windowSec < 60) {
      return '$windowSec sec';
    } else if (windowSec < 3600) {
      return '${windowSec ~/ 60} min';
    } else {
      return '${windowSec ~/ 3600} hr';
    }
  }
}
