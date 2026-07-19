import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/client_model.dart';
import '../models/usage_stats_model.dart';
import '../models/rate_limit_response.dart';

class ApiService {
  static const String baseUrl = 'http://localhost:8080';
  String? _apiKey;

  void setApiKey(String apiKey) {
    _apiKey = apiKey;
  }

  Map<String, String> _getHeaders() {
    final headers = {
      'Content-Type': 'application/json',
    };
    if (_apiKey != null) {
      headers['X-API-Key'] = _apiKey!;
    }
    return headers;
  }

  // Validate API key by fetching client info via /me
  Future<String?> validateApiKey(String apiKey) async {
    try {
      setApiKey(apiKey);
      
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/clients/me'),
        headers: _getHeaders(),
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        return data['id'];
      }
      return null;
    } catch (e) {
      return null;
    }
  }

  // Get all clients
  Future<List<Client>> getClients() async {
    try {
      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/clients'),
      );

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        final List clientsList = data['clients'] ?? [];
        return clientsList.map((json) => Client.fromJson(json)).toList();
      } else {
        throw Exception('Failed to load clients: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Failed to load clients: $e');
    }
  }

  // Get single client
  Future<Client> getClient(String clientId) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/clients/$clientId'),
    );

    if (response.statusCode == 200) {
      return Client.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to load client');
    }
  }

  // Create client
  Future<Client> createClient({
    required String id,
    required String name,
    required int limit,
    required int windowSec,
  }) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/clients'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'id': id,
        'name': name,
        'limit': limit,
        'window_sec': windowSec,
      }),
    );

    if (response.statusCode == 201) {
      return Client.fromJson(json.decode(response.body));
    } else {
      final error = json.decode(response.body);
      throw Exception(error['error'] ?? 'Failed to create client');
    }
  }

  // Update client
  Future<Client> updateClient({
    required String id,
    required String name,
    required int limit,
    required int windowSec,
  }) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/v1/clients/$id'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'name': name,
        'limit': limit,
        'window_sec': windowSec,
      }),
    );

    if (response.statusCode == 200) {
      return Client.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to update client');
    }
  }

  // Check rate limit
  Future<RateLimitResponse> checkRateLimit({
    required String clientId,
    String? resource,
  }) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/v1/ratelimit/check'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'client_id': clientId,
        if (resource != null) 'resource': resource,
      }),
    );

    if (response.statusCode == 200 || response.statusCode == 429) {
      return RateLimitResponse.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to check rate limit');
    }
  }

  // Get usage statistics (PROTECTED - needs API key)
  Future<UsageStats> getUsageStats(String clientId, {int days = 30}) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/dashboard/usage/$clientId?days=$days'),
      headers: _getHeaders(),  // Add API key header
    );

    if (response.statusCode == 200) {
      return UsageStats.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to load usage stats');
    }
  }

  // Get trend data (PROTECTED - needs API key)
  Future<Map<String, dynamic>> getTrendData(String clientId,
      {int days = 7}) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/dashboard/trends/$clientId?days=$days'),
      headers: _getHeaders(),  // Add API key header
    );

    if (response.statusCode == 200) {
      return json.decode(response.body);
    } else {
      throw Exception('Failed to load trend data');
    }
  }

  // Get current stats (real-time)
  Future<Map<String, dynamic>> getCurrentStats(String clientId) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/v1/stats/$clientId'),
    );

    if (response.statusCode == 200) {
      return json.decode(response.body);
    } else {
      throw Exception('Failed to load current stats');
    }
  }

  // Health check
  Future<Map<String, dynamic>> healthCheck() async {
    final response = await http.get(
      Uri.parse('$baseUrl/health'),
    );

    if (response.statusCode == 200) {
      return json.decode(response.body);
    } else {
      throw Exception('Health check failed');
    }
  }
}
