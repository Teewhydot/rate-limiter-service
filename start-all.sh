#!/bin/bash

echo "🚀 Starting Rate Limiter Service with Dashboard..."
echo ""

# Start Docker services
echo "1️⃣  Starting backend services (Docker)..."
docker-compose up -d
echo "   ✓ Backend started on ports 8080, 8081, 8082"
echo ""

# Wait for services to be ready
echo "2️⃣  Waiting for services to initialize..."
sleep 15
echo "   ✓ Services ready"
echo ""

# Check health
echo "3️⃣  Checking backend health..."
HEALTH=$(curl -s http://localhost:8080/health | grep -o '"status":"healthy"')
if [ -n "$HEALTH" ]; then
    echo "   ✓ Backend is healthy"
else
    echo "   ⚠ Backend health check failed"
fi
echo ""

# Start Flutter dashboard
echo "4️⃣  Starting Flutter dashboard..."
echo "   Dashboard will open in Chrome on http://localhost:3000"
echo ""
echo "📊 Opening dashboard..."
cd flutter_dashboard
flutter run -d chrome --web-port 3000 --dart-define=API_URL=http://localhost:8080

# Note: This script will block here until you quit Flutter (press 'q')
