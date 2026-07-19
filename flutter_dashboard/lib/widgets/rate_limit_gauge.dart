import 'package:flutter/material.dart';

class RateLimitGauge extends StatelessWidget {
  final int currentUsed;
  final int limit;
  final int remaining;

  const RateLimitGauge({
    super.key,
    required this.currentUsed,
    required this.limit,
    required this.remaining,
  });

  @override
  Widget build(BuildContext context) {
    final percentage = limit > 0 ? (currentUsed / limit) : 0.0;
    final color = percentage < 0.7
        ? Colors.green
        : percentage < 0.9
            ? Colors.orange
            : Colors.red;

    return Column(
      children: [
        SizedBox(
          height: 150,
          child: Stack(
            alignment: Alignment.center,
            children: [
              SizedBox(
                width: 150,
                height: 150,
                child: CircularProgressIndicator(
                  value: percentage,
                  strokeWidth: 12,
                  backgroundColor: Colors.grey[200],
                  valueColor: AlwaysStoppedAnimation<Color>(color),
                ),
              ),
              Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    currentUsed.toString(),
                    style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: color,
                        ),
                  ),
                  Text(
                    'of $limit',
                    style: TextStyle(
                      color: Colors.grey[600],
                      fontSize: 14,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _buildStat('Used', currentUsed, color),
            _buildStat('Remaining', remaining, Colors.grey),
            _buildStat('Limit', limit, Colors.blue),
          ],
        ),
      ],
    );
  }

  Widget _buildStat(String label, int value, Color color) {
    return Column(
      children: [
        Text(
          value.toString(),
          style: TextStyle(
            fontSize: 20,
            fontWeight: FontWeight.bold,
            color: color,
          ),
        ),
        Text(
          label,
          style: TextStyle(
            fontSize: 12,
            color: Colors.grey[600],
          ),
        ),
      ],
    );
  }
}
