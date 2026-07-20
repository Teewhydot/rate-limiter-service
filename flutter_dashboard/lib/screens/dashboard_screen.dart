import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../services/api_service.dart';
import '../models/client_model.dart';
import 'client_details_screen.dart';
import '../widgets/client_card.dart';
import '../widgets/system_health_card.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key});

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  List<Client>? _clients;
  bool _isLoading = true;
  String? _error;
  Map<String, dynamic>? _healthStatus;
  final TextEditingController _apiKeyController = TextEditingController();
  String? _clientId;
  bool _isAuthenticated = false;

  Timer? _pollingTimer;

  @override
  void dispose() {
    _apiKeyController.dispose();
    _pollingTimer?.cancel();
    super.dispose();
  }

  void _startPolling() {
    _pollingTimer?.cancel();
    _pollingTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      if (_isAuthenticated && mounted) {
        _loadData(showLoading: false);
      }
    });
  }

  @override
  void initState() {
    super.initState();
    // Don't load data until API key is entered
  }

  void _setApiKey() async {
    final apiKey = _apiKeyController.text.trim();
    if (apiKey.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Please enter an API key')),
      );
      return;
    }

    setState(() => _isLoading = true);

    try {
      // Set API key and validate it to get the client ID
      final clientId = await context.read<ApiService>().validateApiKey(apiKey);
      
      if (clientId == null) {
        throw Exception('Invalid API Key');
      }

      setState(() {
        _clientId = clientId;
        _isAuthenticated = true;
        _isLoading = false;
      });
      
      // Load dashboard data and start polling
      _loadData();
      _startPolling();
    } catch (e) {
      setState(() {
        _isLoading = false;
        _isAuthenticated = false;
      });
      
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Invalid API key: ${e.toString()}')),
      );
    }
  }

  void _logout() {
    _pollingTimer?.cancel();
    setState(() {
      _apiKeyController.clear();
      _clientId = null;
      _isAuthenticated = false;
      _clients = null;
      _healthStatus = null;
    });
  }

  Future<void> _loadData({bool showLoading = true}) async {
    if (!_isAuthenticated || _clientId == null) {
      return;
    }

    if (showLoading) {
      setState(() {
        _isLoading = true;
        _error = null;
      });
    }

    try {
      final apiService = context.read<ApiService>();
      
      // Get only this client's data
      final client = await apiService.getClient(_clientId!);
      final health = await apiService.healthCheck();

      if (mounted) {
        setState(() {
          _clients = [client]; // Show only logged-in client
          _healthStatus = health;
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          if (showLoading) _error = e.toString();
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Rate Limiter Dashboard',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            if (_isAuthenticated && _clientId != null)
              Text(
                'Client: $_clientId',
                style: const TextStyle(fontSize: 12),
              ),
          ],
        ),
        actions: [
          if (_isAuthenticated)
            IconButton(
              icon: const Icon(Icons.logout),
              onPressed: _logout,
              tooltip: 'Logout',
            ),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _isAuthenticated ? _loadData : null,
            tooltip: 'Refresh',
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: !_isAuthenticated ? _buildLoginForm() : _buildBody(),
    );
  }

  Widget _buildLoginForm() {
    return Center(
      child: Card(
        margin: const EdgeInsets.all(24),
        child: Container(
          constraints: const BoxConstraints(maxWidth: 400),
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Icon(Icons.vpn_key, size: 64, color: Colors.blue),
              const SizedBox(height: 16),
              Text(
                'Enter Your API Key',
                style: Theme.of(context).textTheme.headlineSmall,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              const Text(
                'View your rate limiting stats',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              TextField(
                controller: _apiKeyController,
                decoration: const InputDecoration(
                  labelText: 'API Key',
                  hintText: 'sk_live_...',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.key),
                ),
                obscureText: true,
              ),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: _setApiKey,
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                ),
                child: const Text('View Dashboard'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildBody() {
    if (_isLoading) {
      return const Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            CircularProgressIndicator(),
            SizedBox(height: 16),
            Text('Loading dashboard...'),
          ],
        ),
      );
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(
              Icons.error_outline,
              size: 64,
              color: Colors.red,
            ),
            const SizedBox(height: 16),
            Text(
              'Error loading data',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              _error!,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 24),
            ElevatedButton.icon(
              onPressed: _loadData,
              icon: const Icon(Icons.refresh),
              label: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    if (_clients == null || _clients!.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.person_outline,
              size: 64,
              color: Colors.grey[400],
            ),
            const SizedBox(height: 16),
            Text(
              'Client: $_clientId',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            const Text('Loading your data...'),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: _loadData,
      child: CustomScrollView(
        slivers: [
          // System Health Section
          SliverPadding(
            padding: const EdgeInsets.all(16),
            sliver: SliverToBoxAdapter(
              child: SystemHealthCard(healthStatus: _healthStatus),
            ),
          ),

          // Clients Section Header
          SliverPadding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            sliver: SliverToBoxAdapter(
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    'Your Account',
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                  ),
                ],
              ),
            ),
          ),
          const SliverPadding(padding: EdgeInsets.only(top: 16)),

          // Clients Grid
          SliverPadding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            sliver: SliverGrid(
              gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                maxCrossAxisExtent: 400,
                mainAxisSpacing: 16,
                crossAxisSpacing: 16,
                childAspectRatio: 1.5,
              ),
              delegate: SliverChildBuilderDelegate(
                (context, index) {
                  final client = _clients![index];
                  return ClientCard(
                    client: client,
                    onTap: () {
                      Navigator.push(
                        context,
                        MaterialPageRoute(
                          builder: (context) =>
                              ClientDetailsScreen(client: client),
                        ),
                      ).then((_) => _loadData());
                    },
                  );
                },
                childCount: _clients!.length,
              ),
            ),
          ),
          const SliverPadding(padding: EdgeInsets.only(bottom: 80)),
        ],
      ),
    );
  }
}
