import 'package:flutter/material.dart';

class SystemHealthCard extends StatelessWidget {
  final Map<String, dynamic>? healthStatus;

  const SystemHealthCard({super.key, this.healthStatus});

  @override
  Widget build(BuildContext context) {
    if (healthStatus == null) return const SizedBox();

    final status = healthStatus!['status'] as String;
    final isHealthy = status == 'healthy';

    return Card(
      color: isHealthy ? Colors.green.shade50 : Colors.orange.shade50,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Icon(
              isHealthy ? Icons.check_circle : Icons.warning,
              color: isHealthy ? Colors.green : Colors.orange,
              size: 32,
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'System Status: ${status.toUpperCase()}',
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Redis: ${healthStatus!['redis']} | PostgreSQL: ${healthStatus!['postgres']}',
                    style: TextStyle(
                      fontSize: 12,
                      color: Colors.grey[700],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
