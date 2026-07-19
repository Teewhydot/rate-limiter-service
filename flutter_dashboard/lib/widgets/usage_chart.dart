import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';

class UsageChart extends StatelessWidget {
  final int allowedRequests;
  final int blockedRequests;

  const UsageChart({
    super.key,
    required this.allowedRequests,
    required this.blockedRequests,
  });

  @override
  Widget build(BuildContext context) {
    final total = allowedRequests + blockedRequests;
    if (total == 0) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(32),
          child: Text('No data available'),
        ),
      );
    }

    return SizedBox(
      height: 200,
      child: PieChart(
        PieChartData(
          sections: [
            PieChartSectionData(
              value: allowedRequests.toDouble(),
              title: allowedRequests.toString(),
              color: Colors.green,
              radius: 60,
              titleStyle: const TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.bold,
                color: Colors.white,
              ),
            ),
            PieChartSectionData(
              value: blockedRequests.toDouble(),
              title: blockedRequests.toString(),
              color: Colors.red,
              radius: 60,
              titleStyle: const TextStyle(
                fontSize: 16,
                fontWeight: FontWeight.bold,
                color: Colors.white,
              ),
            ),
          ],
          sectionsSpace: 2,
          centerSpaceRadius: 40,
        ),
      ),
    );
  }
}
